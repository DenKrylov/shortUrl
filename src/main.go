package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/jackc/pgx/v4"
)

type Data struct {
	post      int8
	PostInput string
	GetInput  string
	PostEnter string
	GetEnter  string
}

var data Data

func check(str string) int8 {
	if len(str) == 0 {
		return 1
	}
	for i, v := range str {
		if (v >= 97 && v <= 125) ||
			(v >= 43 && v <= 58) ||
			(v >= 60 && v <= 93) ||
			(v >= 37 && v <= 38) ||
			v == 26 || v == 32 ||
			v == 35 || v == 95 {
			if i > 2048 {
				return 1
			}
		} else {
			return 1
		}
	}
	return 0
}

func postUrl(w http.ResponseWriter, r *http.Request) {
	data.post = 1
	data.PostInput = r.FormValue("postUrl")
	http.Redirect(w, r, "/index", http.StatusSeeOther)
	res := check(data.PostInput)
	if res == 0 {
		connect()
	} else {
		data.PostEnter = "url is not valid"
	}
}

func getUrl(w http.ResponseWriter, r *http.Request) {
	data.post = 2
	data.GetInput = r.FormValue("getUrl")
	http.Redirect(w, r, "/index", http.StatusSeeOther)
	if len(data.GetInput) == 0 {
		data.GetEnter = "short url is empty"
	} else {
		connect()
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	page, _ := template.ParseFiles("index.html")
	page.Execute(w, data)
}

func randomSimvol() string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789")
	str := make([]rune, 6)

	for i := range str {
		str[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(str)
}

func addEndShortUrl() string {
	var letterRunes0 = []rune("abcdefghijklmnopqrstuvwxyz")
	var letterRunes1 = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	var letterRunes2 = []rune("123456789")
	str := make([]rune, 4)

	str[0] = '_'
	str[1] = letterRunes0[rand.Intn(len(letterRunes0))]
	str[2] = letterRunes1[rand.Intn(len(letterRunes1))]
	str[3] = letterRunes2[rand.Intn(len(letterRunes2))]

	return string(str)
}

func createShortUrl() string {
	rand.Seed(time.Now().UnixNano())
	var str string
	str += randomSimvol()
	str += addEndShortUrl()
	return str
}

func connect() {
	var oUrl, sUrl string
	conn, err := pgx.Connect(context.Background(), os.Getenv("172.0.0.1:5432"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	if data.post == 1 {

		conn.QueryRow(context.Background(), `
			SELECT origin_url, short_url FROM url
			WHERE origin_url LIKE '`+data.PostInput+`'
		`).Scan(&oUrl, &sUrl)

		if len(oUrl) == 0 {
			var shortUrl string
			sUrl = ""
			for {
				shortUrl = createShortUrl()
				conn.QueryRow(context.Background(), `
					SELECT short_url FROM url
					WHERE short_url LIKE '`+shortUrl+`'
				`).Scan(&sUrl)
				if len(sUrl) == 0 {
					break
				}
			}
			data.PostEnter = shortUrl
			conn.QueryRow(context.Background(), `
			INSERT INTO url(origin_url, short_url)
			VALUES ('`+data.PostInput+`', '`+shortUrl+`');
			`)
		} else if len(oUrl) != 0 {
			data.PostEnter = sUrl
		}
	} else if data.post == 2 {
		conn.QueryRow(context.Background(), `
			SELECT origin_url, short_url FROM url
			WHERE short_url LIKE '`+data.GetInput+`'
		`).Scan(&oUrl, &sUrl)
		if len(sUrl) == 0 {
			data.GetEnter = "not short url"
		} else {
			data.GetEnter = oUrl
		}
	}
}

func curl(w http.ResponseWriter, r *http.Request) {
	d, _ := ioutil.ReadAll(r.Body)
	res := string(d)
	err := check(res)
	if r.Method == "POST" && err == 0 {
		data.PostInput = res
		data.post = 1
		connect()
		fmt.Fprintln(w, data.PostEnter)
		log.Println(r.Method, data.PostInput, data.PostEnter)
	} else if r.Method == "GET" && err == 0 {
		data.GetInput = res
		data.post = 2
		connect()
		fmt.Fprintln(w, data.GetEnter)
		log.Println(r.Method, data.GetInput, data.GetEnter)
	} else {
		fmt.Fprintln(w, "This request is not being processed")
	}
}

func main() {
	connect()
	http.HandleFunc("/", curl)
	http.HandleFunc("/index", index)
	http.HandleFunc("/postUrl", postUrl)
	http.HandleFunc("/getUrl", getUrl)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
