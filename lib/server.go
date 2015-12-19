package subfwd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/jpillora/go-tld"
	"github.com/jpillora/subfwd/lib/heroku"

	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const appName = "subfwd"
const appDomain = appName + ".herokuapp.com"

//Subfwd is an HTTP server
type Subfwd struct {
	server     *http.Server
	fileserver http.Handler
	logf       func(string, ...interface{})
	stats      struct {
		Uptime   string
		Forwards uint
	}
}

//New creates a new sandbox
func New() *Subfwd {
	s := &Subfwd{}
	s.fileserver = http.FileServer(http.Dir("."))
	s.stats.Uptime = time.Now().UTC().Format(time.RFC822)
	s.logf = log.New(os.Stdout, appName+": ", 0).Printf //log.LstdFlags
	return s
}

//ListenAndServe and sandbox API and frontend
func (s *Subfwd) ListenAndServe(port string) error {

	if !heroku.ValidCreds() {
		log.Fatal("Invalid Heroku credentials")
	}

	server := &http.Server{
		Addr:           ":" + port,
		Handler:        http.HandlerFunc(s.route),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	s.logf("Listening at %s...", port)
	if port == "3000" {
		s.logf("View locally at http://lvho.st:3000")
	}

	return server.ListenAndServe()
}

//route request
func (s *Subfwd) route(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		w.WriteHeader(404)
	} else if r.Host == appDomain || r.Host == "lvho.st:3000" {
		s.admin(w, r)
	} else {
		s.redirect(w, r)
	}
}

//admin request
func (s *Subfwd) admin(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/static") {
		//serve admin files
		s.fileserver.ServeHTTP(w, r)
		return
	} else if r.URL.Path == "/stats" {
		//show stats
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		b, _ := json.Marshal(s.stats)
		w.Write(b)
	} else if r.URL.Path == "/headers" {
		//echo request
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		b, _ := json.Marshal(r.Header)
		w.Write(b)
	} else if r.URL.Path == "/setup" {
		//perform setup check on domain
		err := s.setup(r.URL.Query().Get("domain"))
		if err == nil {
			w.WriteHeader(200)
		} else {
			s.logf("setup failed: %s", err)
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
		}
	} else {
		w.WriteHeader(404)
	}
}

//setup request
func (s *Subfwd) setup(domain string) error {

	u, err := tld.Parse("http://" + domain)
	if err != nil {
		return errors.New("URL_ERROR")
	}

	if u.Subdomain != "" || u.Domain == "" || u.TLD == "" {
		return errors.New("DOMAIN_ERROR")
	}

	//check wildcard CNAME
	cname, err := net.LookupCNAME(randHex() + "." + domain)
	if err != nil {
		return errors.New("NO_CNAME")
	}
	cname = strings.TrimSuffix(cname, ".")
	if cname != "handle."+appDomain && !strings.HasSuffix(cname, "herokuapp.com") {
		s.logf("WRONG_CNAME: %s", cname)
		return errors.New("WRONG_CNAME")
	}

	//check heroku
	if !heroku.HasDomain(domain) {
		s.logf("Adding new domain: %s", domain)
		if !heroku.SetDomain(domain) {
			return errors.New("HEROKU_ERROR")
		}
	}

	return nil
}

//redirect request
func (s *Subfwd) redirect(w http.ResponseWriter, r *http.Request) {

	u, err := tld.Parse("http://" + r.Host)
	if err != nil {
		s.logf("URL parse failed on %s (%s)", r.Host, err)
		w.WriteHeader(500)
		w.Write([]byte("This shouldn't happen..."))
		return
	}

	//the domain is the host without the port
	domain := u.Domain + "." + u.TLD

	//debug swap (local dns is too hard - use live records)
	if domain == "lvho.st" {
		domain = appDomain
	}

	subdomain := appName + "-" + u.Subdomain + "." + domain

	//try incoming subdomain
	txts, err := net.LookupTXT(subdomain)

	//fallback to default
	if err != nil {
		txts, err = net.LookupTXT(appName + "-default." + domain)
	}

	//not found!
	if err != nil {
		s.logf("TXT Lookup failed on %s (%s)", subdomain, err)
		w.WriteHeader(404)
		w.Write([]byte("Redirect failed [Not found]"))
		return
	}

	for _, u := range txts {
		if strings.HasPrefix(u, "http") {
			u = substitiute(u, r)
			if _, err := url.Parse(u); err != nil {
				s.logf("Invalid URL '%s'", u)
				w.WriteHeader(400)
				w.Write([]byte("Redirect failed [Invalid URL]"))
				return
			}
			w.Header().Set("Location", u)
			w.WriteHeader(302)
			w.Write([]byte("You are being redirected to " + u))
			s.logf("redirecting %s -> %s", domain, u)
			s.stats.Forwards++
			return
		}
	}

	s.logf("No URL set on %s", subdomain)
	w.WriteHeader(404)
	w.Write([]byte("Redirect failed [No URL]"))
	return
}

//=============

var trimPort = regexp.MustCompile(`\:\d+$`)
var urlVars = regexp.MustCompile(`\$(IP|DATE|HEADER\[[\w-]+\])`)

func substitiute(url string, r *http.Request) string {
	return string(urlVars.ReplaceAllFunc([]byte(url), func(input []byte) []byte {
		s := string(input[1:])
		var output []byte
		switch s {
		case "IP":
			ip := r.Header.Get("X-Forwarded-For") //use real IP
			if ip == "" {
				ip = trimPort.ReplaceAllString(r.RemoteAddr, "") //else connection IP
			}
			output = []byte(ip)
		case "DATE":
			output = []byte(fmt.Sprintf("%d", time.Now().UnixNano()/1e6))
		default: //"HEADER"
			name := s[7:]
			name = name[:len(name)-1]
			output = []byte(r.Header.Get(name))
		}
		return output
	}))
}

func randHex() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
