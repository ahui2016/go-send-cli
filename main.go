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

	"github.com/ahui2016/go-send/pass"
	"github.com/ahui2016/goutil"
	"github.com/ahui2016/goutil/session"
)

const (
	website        = "http://127.0.0.1"
	dataFolderName = "gosend_data_folder"
	cookieFileName = "gosend.cookie"
)

var (
	text = flag.String("text", "", "insert a text message")
)

var (
	dataDir    = filepath.Join(goutil.UserHomeDir(), dataFolderName)
	cookiePath = filepath.Join(dataDir, cookieFileName)
)

type Result struct {
	Message string
}

func main() {
	flag.Parse()

	if goutil.PathIsNotExist(cookiePath) {
		_ = login()
	}
	cookie := readCookie()

	// 无 -text 参数
	if *text == "" {
		textMsg, ok := getLastText(cookie)

		// 如果登录失败，很可能是 cookie 过期，重新登录一次。
		if !ok {
			cookie = login()
			textMsg, ok = getLastText(cookie)

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
	ok := insertTextMsg(cookie, *text)
	if !ok {
		cookie = login()
		if ok := insertTextMsg(cookie, *text); !ok {
			log.Fatal("无法登录，未知错误")
		}
	}
}

func login() (cookie *http.Cookie) {
	v := url.Values{}
	v.Set("password", pass.Word)
	res, err := http.PostForm(website+"/api/login", v)
	goutil.CheckErrorFatal(err)
	for _, cookie := range res.Cookies() {
		if cookie.Name == session.SessionID {
			saveCookie(cookie)
			return cookie
		}
	}
	return nil
}

func getLastText(cookie *http.Cookie) (textMsg string, ok bool) {
	req := requestGet(website+"/api/last-text", cookie)
	body, ok := httpClientDo(req)
	if !ok {
		return
	}
	var result Result
	err := json.Unmarshal(body, &result)
	goutil.CheckErrorFatal(err)
	return result.Message, true
}

func insertTextMsg(cookie *http.Cookie, textMsg string) (ok bool) {
	data := url.Values{}
	data.Set("text-msg", textMsg)
	req := requestPost(website+"/api/add-text-msg", data, cookie)
	_, ok = httpClientDo(req)
	return
}

func httpClientDo(req *http.Request) (body []byte, isLoggedIn bool) {
	res, err := http.DefaultClient.Do(req)
	goutil.CheckErrorFatal(err)

	body, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	goutil.CheckErrorFatal(err)

	if res.StatusCode != 200 {
		var result Result
		err := json.Unmarshal(body, &result)

		// 如果 result 里有 Message, 并且 Message 的内容是要求登录
		if err == nil &&
			goutil.NoCaseContains(result.Message, "require login") {
			return nil, false
		}

		// 如果 result 里没有 Message 或者发生其他错误
		log.Fatal(res.StatusCode, string(body))
	}

	// res.StatusCode == 200
	return body, true
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
