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

type Message struct {
	Message string
}

func main() {
	flag.Parse()

	if *text == "" {
		if goutil.PathIsNotExist(cookiePath) {
			_ = login()
		}
		cookie := readCookie()
		textMsg, isLoggedIn := getLastText(cookie)

		// 如果登录失败，很可能是 cookie 过期，重新登录一次。
		if !isLoggedIn {
			cookie = login()
			textMsg, isLoggedIn = getLastText(cookie)

			// 重新登录应该成功才对，如果还是失败，原因就要慢慢找了。
			if !isLoggedIn {
				log.Fatal("无法登录，未知错误")
			}
		}
		_, err := fmt.Fprint(os.Stdout, textMsg)
		goutil.CheckErrorFatal(err)
		return
	}

	// _, err := db.InsertTextMsg(*text)
	// goutil.CheckErrorFatal(err)
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

func getLastText(cookie *http.Cookie) (textMsg string, isLoggedIn bool) {
	req, err := http.NewRequest(
		http.MethodGet, website+"/api/last-text", nil)
	goutil.CheckErrorFatal(err)

	req.AddCookie(cookie)
	res, err := http.DefaultClient.Do(req)
	goutil.CheckErrorFatal(err)

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	goutil.CheckErrorFatal(err)

	var m Message
	if err := json.Unmarshal(body, &m); err != nil {
		log.Fatal(string(body))
	}

	if res.StatusCode != 200 {
		if goutil.NoCaseContains(m.Message, "require login") {
			return
		}
		log.Fatal(res.StatusCode, string(body))
	}

	// res.StatusCode == 200
	return m.Message, true
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
