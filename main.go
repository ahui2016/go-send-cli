package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ahui2016/goutil"
	"github.com/ahui2016/goutil/session"
)

const (
	dataFolderName       = "gosend_data_folder"
	cookieFileName       = "gosend.cookie"
	configFileName       = "config-cli"
	gosendConfigFileName = "config"
)

var (
	config Config
)

var (
	text = flag.String("text", "", "insert a text message")
	pass = flag.String("pass", "", "set password, cannot use empty password")
	addr = flag.String("addr", "", "set the website address of go-send")
)

var (
	dataDir          = filepath.Join(goutil.UserHomeDir(), dataFolderName)
	configPath       = filepath.Join(dataDir, configFileName)
	gosendConfigPath = filepath.Join(dataDir, gosendConfigFileName)
)

type Config struct {
	Cookie   http.Cookie
	Address  string
	Password string
}

type Result struct {
	Message string
}

func init() {
	goutil.MustMkdir(dataDir)
	flag.Parse()
	setPasswordAddr()
	setConfig()
}

func main() {
	cookies := []*http.Cookie{&config.Cookie}

	// 如果未输入 -text 参数，就直接获取第一条文本备忘。
	if *text == "" {
		textMsg, ok := getLastText(cookies)

		// 如果获取失败，很可能是 cookie 过期，重新登录一次。
		if !ok {
			cookies = login()
			textMsg, ok = getLastText(cookies)

			// 重新登录应该成功才对，如果还是失败，原因就要慢慢找了。
			if !ok {
				log.Fatal("无法登录，未知错误")
			}
		}
		_, err := fmt.Fprint(os.Stdout, textMsg)
		goutil.CheckErrorFatal(err)
		return
	}

	// 有 -text 参数，就发送文本备忘。
	ok := sendTextMsg(cookies, *text)
	if !ok {
		cookies = login()
		if ok := sendTextMsg(cookies, *text); !ok {
			log.Fatal("无法登录，未知错误")
		}
	}
}

func setPasswordAddr() {
	cfg := readConfig()

	// 如果未输入密码或网址，则忽略。如果输入了密码或网址，则进行设置。
	if *pass != "" {
		cfg.Password = *pass
		log.Println("Password is set.")
	}
	if *addr != "" {
		cfg.Address = *addr
		log.Println("Address is set.")
	}
	if *pass+*addr != "" {
		saveConfig(&cfg)
		os.Exit(0)
	}
}

func setConfig() {
	configJSON, err := ioutil.ReadFile(configPath)

	// configPath 有内容，就直接使用 configPath 的内容。
	if err == nil && len(configJSON) > 0 {
		goutil.CheckErrorFatal(json.Unmarshal(configJSON, &config))
	} else {
		// configPath 没有文件或内容为空, 则尝试获取 gosendConfigPath 的内容
		gosendConfigJSON, err := ioutil.ReadFile(gosendConfigPath)

		// 如果 configPath 没有内容，而 gosendConfigPath 有内容，就以 gosendConfigPath 的内容为准。
		if err == nil && len(gosendConfigJSON) > 0 {
			goutil.CheckErrorFatal(json.Unmarshal(gosendConfigJSON, &config))
			config.Address = "http://" + config.Address
			saveConfig(nil)
		}
	}

	// 检查密码和网址是否已经设置，如示设置则提示用户进行设置。
	if config.Password+config.Address == "" {
		log.Fatal("password and address is not set 请先设置密码和网址")
	}
	if config.Password == "" {
		log.Fatal("password is not set 请先设置密码")
	}
	if config.Address == "" {
		log.Fatal("address is not set 请先设置网址")
	}
}

func readConfig() (cfg Config) {
	configJSON, err := ioutil.ReadFile(configPath)
	// 忽略 not found 错误。
	if goutil.ErrorContains(err, "cannot find") {
		return
	}
	goutil.CheckErrorFatal(err)
	goutil.CheckErrorFatal(json.Unmarshal(configJSON, &cfg))
	return
}

func saveConfig(cfg *Config) {
	if cfg != nil {
		config = *cfg
	}
	configJSON, err := json.MarshalIndent(config, "", "    ")
	goutil.CheckErrorFatal(err)
	goutil.CheckErrorFatal(
		ioutil.WriteFile(configPath, configJSON, 0600))
	return
}

func login() []*http.Cookie {
	v := url.Values{}
	v.Set("password", config.Password)
	res, err := http.PostForm(config.Address+"/api/login", v)
	goutil.CheckErrorFatal(err)

	body := getResponseBody(res)
	if res.StatusCode != 200 {
		log.Fatal(res.StatusCode, string(body))
	}

	for _, cookie := range res.Cookies() {
		if cookie.Name == session.SessionID {
			saveCookie(cookie)
			return []*http.Cookie{cookie}
		}
	}
	return nil
}

func getLastText(cookies []*http.Cookie) (textMsg string, ok bool) {
	res, err := goutil.HttpGet(config.Address+"/api/last-text", cookies)
	goutil.CheckErrorFatal(err)
	return getResultMessage(res)
}

func sendTextMsg(cookies []*http.Cookie, textMsg string) (ok bool) {
	data := url.Values{}
	data.Set("text-msg", textMsg)
	res, err := goutil.HttpPostForm(config.Address+"/api/add-text-msg", data, cookies)
	goutil.CheckErrorFatal(err)
	body := getResponseBody(res)
	if res.StatusCode != 200 {
		log.Fatal(res.StatusCode, string(body))
	}
	return true
}

func getResponseBody(res *http.Response) []byte {
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	goutil.CheckErrorFatal(err)
	return body
}

func getResultMessage(res *http.Response) (msg string, isLoggedIn bool) {
	var result Result
	body := getResponseBody(res)
	err := json.Unmarshal(body, &result)

	// 成功获取了最新的文本备忘
	if err == nil && res.StatusCode == 200 {
		return result.Message, true
	}

	// 如果 Message 的内容是要求登录
	if err == nil &&
		goutil.NoCaseContains(result.Message, "require login") {
		return "", false
	}

	// 其他情况一律报错
	log.Fatal(res.StatusCode, string(body))
	return "", true
}

func saveCookie(cookie *http.Cookie) {
	config.Cookie = http.Cookie{
		Name:  cookie.Name,
		Value: cookie.Value,
	}
	saveConfig(nil)
}
