package shortener

import (
	"log"
	"os"

	"github.com/garyburd/redigo/redis"
	"github.com/soveran/redisurl"
)

type Redis struct {
	c redis.Conn
}

func NewRedis() *Redis {
	url := os.Getenv("REDISTOGO_URL")

	if url == "" {
		log.Fatal("Missing REDISTOGO_URL")
	}

	c, err := redisurl.ConnectToURL(url)
	if err != nil {
		log.Fatal(err)
	}

	return &Redis{c}
}

func (r *Redis) set(key, val string) interface{} {
	r.c.Send("SET", key, val)
	r.c.Flush()
	v, err := r.c.Receive()
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func (r *Redis) get(key string) interface{} {
	r.c.Send("GET", key)
	r.c.Flush()
	v, err := r.c.Receive()
	if err != nil {
		log.Fatal(err)
	}
	return v
}
