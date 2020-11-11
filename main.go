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
	"strings"

	"github.com/ahui2016/goutil"
	"github.com/ahui2016/goutil/session"
)

const (
	website          = "http://127.0.0.1"
	dataFolderName   = "gosend_data_folder"
	cookieFileName   = "gosend.cookie"
	passwordFileName = "password"
)

var (
	localPassword string
)

var (
	text = flag.String("text", "", "insert a text message")
	pass = flag.String("pass", "", "set password, cannot use empty password")
)

var (
	dataDir      = filepath.Join(goutil.UserHomeDir(), dataFolderName)
	cookiePath   = filepath.Join(dataDir, cookieFileName)
	passwordPath = filepath.Join(dataDir, passwordFileName)
)

type Result struct {
	Message string
}

func init() {
	goutil.MustMkdir(dataDir)
	flag.Parse()
	setPassword()
}

func setPassword() {
	// 如果未输入密码参数，则忽略。如果输入了密码，没设置密码。
	if *pass != "" {
		err := ioutil.WriteFile(passwordPath, []byte(*pass), 0600)
		goutil.CheckErrorFatal(err)
		log.Fatal("OK, password is set.")
	}

	// 尝试读取密码，如果设置过密码，此时应能读取，如果读取失败则提示用户设置密码。
	pw, err := ioutil.ReadFile(passwordPath)
	if err != nil || len(pw) == 0 {
		log.Fatal("password is not set 请先设置密码")
		return
	}
	localPassword = string(pw)
}

func main() {

	if goutil.PathIsNotExist(cookiePath) {
		_ = login()
	}
	cookie := readCookie()
	cookies := []*http.Cookie{cookie}

	// 未输入 -text 参数
	if *text == "" {
		textMsg, ok := getLastText(cookies)

		// 如果登录失败，很可能是 cookie 过期，重新登录一次。
		if !ok {
			cookie = login()
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

	// 有 -text 参数
	ok := insertTextMsg(cookies, *text)
	if !ok {
		cookie = login()
		if ok := insertTextMsg(cookies, *text); !ok {
			log.Fatal("无法登录，未知错误")
		}
	}
}

func login() (cookie *http.Cookie) {
	v := url.Values{}
	v.Set("password", localPassword)
	res, err := http.PostForm(website+"/api/login", v)
	goutil.CheckErrorFatal(err)

	// getResultMessage 里面会检查错误，比如密码错误。
	_, _ = getResultMessage(res)

	for _, cookie := range res.Cookies() {
		if cookie.Name == session.SessionID {
			saveCookie(cookie)
			return cookie
		}
	}
	return nil
}

func getLastText(cookies []*http.Cookie) (textMsg string, ok bool) {
	res, err := goutil.HttpGet(website+"/api/last-text", cookies)
	goutil.CheckErrorFatal(err)
	return getResultMessage(res)
}

func insertTextMsg(cookies []*http.Cookie, textMsg string) (ok bool) {
	data := url.Values{}
	data.Set("text-msg", textMsg)
	res, err := goutil.HttpPostForm(website+"/api/add-text-msg", data, cookies)
	goutil.CheckErrorFatal(err)
	_, ok = getResultMessage(res)
	return
}

func getResultMessage(res *http.Response) (msg string, isLoggedIn bool) {
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	goutil.CheckErrorFatal(err)

	var result Result
	err = json.Unmarshal(body, &result)

	// 如果 result 里有 Message, 并且 Message 的内容是要求登录
	if err == nil &&
		goutil.NoCaseContains(result.Message, "require login") {
		return "", false
	}

	// 如果 result 里没有 Message, 或者 Unmarshal 发生其他错误,
	// 或者 status != 200 并且错误原因不是要求登录。
	if err != nil && res.StatusCode != 200 {
		log.Fatal(res.StatusCode, string(body))
	}

	// 至此，可以确定 result 里必然有 Message, 并且 status == 200
	return result.Message, true
}

func saveCookie(cookie *http.Cookie) {
	ck := http.Cookie{
		Name:  cookie.Name,
		Value: cookie.Value,
	}
	blob, err := json.Marshal(ck)
	goutil.CheckErrorFatal(err)
	goutil.CheckErrorFatal(ioutil.WriteFile(cookiePath, blob, 0600))
}

func readCookie() *http.Cookie {
	blob, err := ioutil.ReadFile(cookiePath)
	goutil.CheckErrorFatal(err)

	var cookie http.Cookie
	goutil.CheckErrorFatal(json.Unmarshal(blob, &cookie))
	return &cookie
}

// RequestPost makes a request with a cookie, posts data as "application/x-www-form-urlencoded".
// usage: http.DefaultClient.Do(req)
func requestPost(reqURL string, data url.Values, cookie *http.Cookie) *http.Request {
	body := strings.NewReader(data.Encode())
	req, err := http.NewRequest(http.MethodPost, reqURL, body)
	goutil.CheckErrorFatal(err)
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// RequestGet makes a request with a cookie.
// usage: http.DefaultClient.Do(req)
func requestGet(reqURL string, cookie *http.Cookie) *http.Request {
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	goutil.CheckErrorFatal(err)
	req.AddCookie(cookie)
	return req
}
