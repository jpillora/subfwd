package subfwd

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
	redis      *Redis
	log        func(string, ...interface{})
	stats      struct {
		Uptime string
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

type Request struct {
	ID   string
	Pass string
}

//admin request
func (s *Subfwd) Admin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/static") {
		s.fileserver.ServeHTTP(w, r)
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(404)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	req := &Request{}

	err = json.Unmarshal(b, req)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	status, err := s.status(req)

	var res interface{} = nil
	switch r.URL.Path {
	case "/status":
		res = status
	case "/add":
		res = status
	case "/del":
		res = status
	default:
		w.WriteHeader(404)
		w.Write([]byte("???"))
		return
	}

	//handler returned error
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	//handler not found
	if res == nil {
		w.WriteHeader(404)
		return
	}

	b, err = json.Marshal(res)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

type Status struct {
	ID         string
	Cnamed     bool
	Hash       string
	Updated    time.Time
	Subdomains []string
}

const randSub = "53ec7bbe77ab264751bea7c1e78c0af334a2f129f."
const txtPrefix = "subfwd-hash="

func (s *Subfwd) status(req *Request) (*Status, error) {
	status := &Status{}
	changed := false

	err := s.redis.get(req.ID, status)
	if err == NotFound {
		status.ID = req.ID
		changed = true
	} else if err != nil {
		return nil, errors.New("REDIS_ERR")
	}

	if !status.Cnamed {
		cname, err := net.LookupCNAME(randSub + status.ID)
		if err != nil {
			return nil, errors.New("NO_CNAME")
		}
		cname = strings.TrimSuffix(cname, ".")
		if cname != "handle.subfwd.com" && !strings.HasSuffix(cname, "herokuapp.com") {
			s.log("WRONG_CNAME: %s", cname)
			return nil, errors.New("WRONG_CNAME")
		}
		status.Cnamed = true
		changed = true
	}

	if req.Pass == "" {
		s.log("awaiting password...")
		return nil, nil
	}

	expectedPass := make([]byte, 32)
	var hexerr error
	if status.Hash != "" {
		_, hexerr = hex.Decode(expectedPass, []byte(status.Hash))
	} else {
		txts, err := net.LookupTXT(status.ID)
		if err != nil {
			return nil, err
		}
		for _, t := range txts {
			if strings.HasPrefix(t, txtPrefix) {
				hstr := strings.TrimPrefix(t, txtPrefix)
				_, hexerr = hex.Decode(expectedPass, []byte(hstr))
				status.Hash = hstr
				changed = true
				break
			}
		}
		if status.Hash == "" {
			return nil, errors.New("NO_PASS")
		}
	}

	if hexerr != nil {
		return nil, fmt.Errorf("NO_HEX")
	}

	mac := hmac.New(sha256.New, []byte("subfwd.com"))
	mac.Write([]byte(req.Pass))
	recievedPass := mac.Sum(nil)

	if !hmac.Equal(expectedPass, recievedPass) {
		return nil, fmt.Errorf("WRONG_PASS")
	}

	if changed {
		status.Updated = time.Now()
		s.redis.set(req.ID, status)
		s.log("status: updated: %s", req.ID)
	}

	return status, nil
}

//redirect request
func (s *Subfwd) Redirect(w http.ResponseWriter, r *http.Request) {
	s.log("[%s] %s %s", r.Host, r.Method, r.URL)
	w.Write([]byte("hello world"))
}

//ListenAndServe and sandbox API and frontend
func (s *Subfwd) ListenAndServe(addr string) error {

	s.log("Connecting to redis...")
	s.redis = NewRedis()

	s.log("Connected to redis")
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
