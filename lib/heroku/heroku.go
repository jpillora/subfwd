package heroku

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const apiBase = "https://api.heroku.com"

var apiKey = os.Getenv("HEROKU_API_KEY")
var appName = os.Getenv("HEROKU_APP_NAME")
var bearerToken = "Bearer " + apiKey
var headers http.Header

func request(method, path, body string) (int, string) {

	if apiKey == "" {
		return 0, ""
	}

	c := &http.Client{}

	var r *http.Request
	if body == "" {
		r, _ = http.NewRequest(method, apiBase+path, nil)
	} else {
		r, _ = http.NewRequest(method, apiBase+path, bytes.NewBufferString(body))
	}

	r.Header = headers

	resp, err := c.Do(r)
	if err != nil {
		log.Printf("Heroku: request: %s", err)
		return 0, ""
	}

	b, _ := ioutil.ReadAll(resp.Body)

	return resp.StatusCode, string(b)

}

func ValidCreds() bool {
	if headers == nil {
		headers = make(http.Header)
		headers.Set("Accept", "application/vnd.heroku+json; version=3")
		headers.Set("Content-Type", "application/json")
		headers.Set("Authorization", bearerToken)
	}

	if appName == "" {
		appName = "subfwd"
	}

	status, _ := request("GET", "/account", "")
	return status == 200
}

func HasDomain(domain string) bool {
	status, _ := request("GET", "/apps/"+appName+"/domains/*."+domain, "")
	return status == 200
}

func SetDomain(domain string) bool {
	status, resp := request("POST", "/apps/"+appName+"/domains", `{"hostname":"*.`+domain+`"}`)
	log.Print(resp)
	return status == 201
}
