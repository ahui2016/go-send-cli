package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ahui2016/goutil"
)

const (
	dataFolderName       = "gosend_data_folder"
	configFileName       = "config-cli"
	gosendConfigFileName = "config"
)

var (
	config Config
)

var (
	file = flag.String("file", "", "send a file")
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
	Address  string
	Password string
}

func init() {
	goutil.MustMkdir(dataDir)
	flag.Parse()
	checkFlagsCombination()
	setPasswordAddr()
	setConfig()
}

func main() {

	// 如果提供了文件，就发送文件。
	// -file 与 -text 不可同时使用，这在 checkFlagsCombination 中进行检查。
	if *file != "" {
		sendFile(*file)
		return
	}

	// 有 -text 参数，就发送文本备忘。
	if *text != "" {
		sendTextMsg(*text)
		return
	}

	// 默认（未输入任何参数）状态下获取最近一条文本消息
	textMsg := getLastText()
	_, err := fmt.Fprint(os.Stdout, textMsg)
	goutil.CheckErrorFatal(err)
}

// checkFlagsCombination 检查命令参数的组合有无问题
func checkFlagsCombination() {
	if (*pass+*addr != "") && *text != "" {
		log.Fatal("Cannot use -text with -pass or -addr 设置密码和网址的功能与收发消息功能不可同时使用")
	}
	if (*pass+*addr != "") && *file != "" {
		log.Fatal("Cannot use -file with -pass or -addr 设置密码和网址的功能与收发文件功能不可同时使用")
	}
	if *text != "" && *file != "" {
		log.Fatal("Cannot use -text and -file at the same time 发送文本与发送文件功能不可同时使用")
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
	if goutil.ErrorContains(err, "cannot find") ||
		goutil.ErrorContains(err, "no such file") {
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

func getLastText() string {
	data := url.Values{}
	data.Set("password", config.Password)

	res, err := http.PostForm(config.Address+"/api/last-text", data)
	goutil.CheckErrorFatal(err)

	body := string(getResponseBody(res))
	if res.StatusCode != 200 {
		log.Fatal(res.StatusCode, body)
	}
	return body
}

func sendTextMsg(textMsg string) {
	data := url.Values{}
	data.Set("password", config.Password)
	data.Set("text-msg", textMsg)

	res, err := http.PostForm(config.Address+"/api/add-text", data)
	goutil.CheckErrorFatal(err)

	body := getResponseBody(res)
	if res.StatusCode != 200 {
		log.Fatal(res.StatusCode, string(body))
	}
}

func sendFile(file string) {
	formData, contentType, err1 := newMultipartForm(file)
	res, err2 := http.Post(
		config.Address+"/api/add-photo", contentType, formData)
	goutil.CheckErrorFatal(goutil.WrapErrors(err1, err2))

	body := getResponseBody(res)
	if res.StatusCode != 200 {
		log.Fatal(res.StatusCode, string(body))
	}
}

func getResponseBody(res *http.Response) []byte {
	body, err := ioutil.ReadAll(res.Body)
	_ = res.Body.Close()
	goutil.CheckErrorFatal(err)
	return body
}

// newMultipartForm create a multipart form, which can be read by multipart.Reader.
func newMultipartForm(filePath string) (
	formData *bytes.Buffer, contentType string, err error) {

	formData = bytes.NewBuffer([]byte{})
	w := multipart.NewWriter(formData)
	defer func() { _ = w.Close() }()

	// 填写密码
	err1 := w.WriteField("password", config.Password)

	// 添加文件
	contents, err2 := ioutil.ReadFile(filePath)
	filename := filepath.Base(filePath)
	fileWriter, err3 := w.CreateFormFile("file", filename)
	if err = goutil.WrapErrors(err1, err2, err3); err != nil {
		return
	}
	if _, err = fileWriter.Write(contents); err != nil {
		return
	}

	return formData, w.FormDataContentType(), nil
}
