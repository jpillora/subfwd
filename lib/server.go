package subfwd

import (
	"errors"
	"regexp"

	"github.com/jpillora/go-tld"
	"github.com/jpillora/subfwd/lib/heroku"

	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var dev = os.Getenv("PROD") != "true"

//Subfwd is an HTTP server
type Subfwd struct {
	server     *http.Server
	fileserver http.Handler
	log        func(string, ...interface{})
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
	s.log = log.New(os.Stdout, "subfwd: ", 0).Printf //log.LstdFlags
	return s
}

//ListenAndServe and sandbox API and frontend
func (s *Subfwd) ListenAndServe(addr string) error {

	if !heroku.ValidCreds() {
		log.Fatal("Invalid Heroku credentials")
	}

	server := &http.Server{
		Addr:           addr,
		Handler:        http.HandlerFunc(s.Route),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	s.log("Listening at %s...", addr)
	return server.ListenAndServe()
}

//route request
func (s *Subfwd) Route(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		w.WriteHeader(404)
	} else if r.Host == "subfwd.com" || r.Host == "subfwd.lvho.st:3000" {
		s.Admin(w, r)
	} else {
		s.Redirect(w, r)
	}
}

//admin request
func (s *Subfwd) Admin(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/static") {
		//serve admin files
		s.fileserver.ServeHTTP(w, r)
		return
		//perform setup check on domain
	} else if r.URL.Path == "/setup" {
		err := s.setup(r.URL.Query().Get("domain"))
		if err == nil {
			w.WriteHeader(200)
		} else {
			s.log("setup failed: %s", err)
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
		}
	} else {
		w.WriteHeader(404)
	}
}

//setup request
func (s *Subfwd) setup(domain string) error {

	url, err := tld.Parse("http://" + domain)
	if err != nil {
		return errors.New("URL_ERROR")
	}

	if url.Subdomain != "" || url.Domain == "" || url.TLD == "" {
		return errors.New("DOMAIN_ERROR")
	}

	//check heroku
	if !heroku.HasDomain(domain) {
		s.log("Adding new domain: %s", domain)
		if !heroku.SetDomain(domain) {
			return errors.New("HEROKU_ERROR")
		}
	}

	return nil
}

var trimPort = regexp.MustCompile(`\:\d+$`)

//redirect request
func (s *Subfwd) Redirect(w http.ResponseWriter, r *http.Request) {

	domain := trimPort.ReplaceAllString(r.Host, "")

	//debug swap
	domain = strings.Replace(domain, ".lvho.st", ".subfwd.com", 1)

	txts, err := net.LookupTXT(domain)
	if err != nil {
		s.log("TXT Lookup failed on %s (%s)", domain, err)
		w.WriteHeader(500)
		w.Write([]byte("Subdomain lookup failed"))
		return
	}

	for _, url := range txts {
		if strings.HasPrefix(url, "http") {
			w.Header().Set("Location", url)
			w.WriteHeader(302)
			w.Write([]byte("You are being redirected to " + url))
			s.stats.Forwards++
			return
		}
	}

	s.log("No URL set on %s", domain)
	w.WriteHeader(404)
	w.Write([]byte("Not found"))
	return
}
