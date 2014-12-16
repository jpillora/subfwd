package shortener

import (
	"log"
	"net/http"
	"os"
	"time"
)

var dev = os.Getenv("PROD") != "true"

//Shortener is an HTTP server
type Shortener struct {
	server *http.Server
	redis  *Redis
	log    func(string, ...interface{})
	stats  struct {
		Uptime string
	}
}

//New creates a new sandbox
func New() *Shortener {
	s := &Shortener{}
	s.stats.Uptime = time.Now().UTC().Format(time.RFC822)
	s.log = log.New(os.Stdout, "shortener: ", 0).Printf //log.LstdFlags
	return s
}

//proxy this request onto play.golang
func (s *Shortener) Redirect(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path == "/favicon.ico" {
		w.WriteHeader(404)
		return
	}

	s.log("[%s] %s %s", r.Host, r.Method, r.URL)

	w.Write([]byte("hello world"))

}

//ListenAndServe and sandbox API and frontend
func (s *Shortener) ListenAndServe(addr string) error {

	s.log("Connecting to redis...")
	s.redis = NewRedis()

	s.log("Connected to redis")
	server := &http.Server{
		Addr:           addr,
		Handler:        http.HandlerFunc(s.Redirect),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	s.log("Set FOO: %v", s.redis.set("FOO", "42"))
	s.log("Get FOO: %v", s.redis.get("FOO"))

	s.log("Listening at %s...", addr)
	return server.ListenAndServe()
}
