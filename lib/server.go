package subfwd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sync"

	ga "github.com/jpillora/go-ogle-analytics"
	"github.com/jpillora/go-tld"
	"github.com/jpillora/subfwd/lib/heroku"
	"github.com/tomasen/realip"

	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

const appName = "subfwd"
const appDomain = appName + ".jpillora.com"

//Subfwd is an HTTP server
type Subfwd struct {
	server     *http.Server
	fileserver http.Handler
	// cache      *lru.Cache TODO
	tracker *ga.Client
	logf    func(string, ...interface{})
	stats   struct {
		Uptime  string
		Success uint
	}
}

//New creates a new sandbox
func New() *Subfwd {
	s := &Subfwd{}
	// s.cache, _ = lru.New(100)
	s.tracker, _ = ga.NewClient(os.Getenv("GA_TRACKER_ID"))
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
		s.logf("View locally at http://abc.example.com:3000")
	}

	return server.ListenAndServe()
}

//route request
func (s *Subfwd) route(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		w.WriteHeader(404)
	} else if r.Host == appDomain || r.Host == "abc.example.com:3000" {
		s.admin(w, r)
	} else {
		s.execute(w, r)
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
	if cname != "subfwd.herokuapp.com" && cname != appDomain {
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

//execute request
func (s *Subfwd) execute(w http.ResponseWriter, r *http.Request) {
	u, err := tld.Parse("http://" + r.Host)
	if err != nil {
		s.logf("URL parse failed on %s (%s)", r.Host, err)
		w.WriteHeader(500)
		w.Write([]byte("This shouldn't happen..."))
		return
	}
	//the domain is the host without the port
	domain := u.Domain + "." + u.TLD
	if domain == "lvh.me" {
		domain = "jpillora.com" //debug swap (local dns is too hard - use live records)
	}
	subdomain := u.Subdomain + "." + domain

	//lookup 3 txt entries in parallel
	wg := &sync.WaitGroup{}
	wg.Add(3)
	lookup := func(domain string, result *url.URL) {
		defer wg.Done()
		txts, err := net.LookupTXT(domain)
		if err != nil {
			return
		}
		for _, txt := range txts {
			if strings.HasPrefix(txt, "http") {
				txt := substitiute(txt, r)
				u, err := url.Parse(txt)
				if err != nil {
					s.logf("Invalid URL '%s'", u)
					continue
				}
				*result = *u
				return
			}
		}
	}

	var forward, proxy, def url.URL
	go lookup("subfwd-"+u.Subdomain+"."+domain, &forward)
	go lookup("subproxy-"+u.Subdomain+"."+domain, &proxy)
	go lookup("subfwd-default."+domain, &def)
	wg.Wait()
	//find target url
	redirect := true
	var target *url.URL
	if proxy.String() != "" {
		redirect = false
		target = &proxy
		// target, _ = url.Parse("https://echo.jpillora.com/foo/bar")
	} else if forward.String() != "" {
		target = &forward
	} else if def.String() != "" {
		target = &def
	} else {
		s.logf("No TXT set for: %s", subdomain)
		if s.tracker != nil {
			go s.tracker.Send(ga.NewEvent("Fail - No TXT", subdomain))
		}
		w.WriteHeader(404)
		w.Write([]byte("Redirect failed [No TXT]"))
		return
	}
	//log
	action := "Redirect"
	if !redirect {
		action = "Proxy"
	}
	s.stats.Success++
	log.Printf("#%05d [Success - %s] %s -> %s (from %s)", s.stats.Success, action, subdomain, target,
		strings.TrimSpace(realip.RealIP(r)+" "+r.Header.Get("Referer")))
	if s.tracker != nil {
		go s.tracker.Send(ga.NewEvent("Success - "+action, subdomain).Label(target.String()))
	}
	//perform
	if redirect {
		http.Redirect(w, r, target.String(), 302)
	} else {
		p := httputil.NewSingleHostReverseProxy(target)
		r.Host = target.Host //fix hostname
		p.ServeHTTP(w, r)
	}
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
			output = []byte(realip.RealIP(r))
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
