package subfwd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/soveran/redisurl"
)

type Redis struct {
	url string
	c   redis.Conn
}

var NotFound error = errors.New("RedisKeyNotFound")

func NewRedis() *Redis {
	url := os.Getenv("REDISTOGO_URL")

	if url == "" {
		log.Fatal("Missing REDISTOGO_URL")
	}

	c, err := redisurl.ConnectToURL(url)
	if err != nil {
		log.Fatal(err)
	}

	r := &Redis{url, c}

	go r.keepalive()

	return r
}

type Keepalive struct {
	StartTime time.Time
	CurrTime  time.Time
}

func (r *Redis) keepalive() {
	start := time.Now()
	for {
		k := &Keepalive{start, time.Now()}
		r.set("KEEPALIVE", k)
		time.Sleep(5 * time.Second)
	}
}

func (r *Redis) reconnect() {
	log.Print("Redis: reconnecting...")
	c, err := redisurl.ConnectToURL(r.url)
	if err != nil {
		log.Printf("Redis: reconnect failed: %s", err)
		return
	}
	r.c = c
	log.Print("Redis: reconnected")
}

func (r *Redis) set(key string, val interface{}) error {
	valbytes, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("Redis: set: %s", err)
		return err
	}
	r.c.Send("SET", key, valbytes)
	r.c.Flush()
	v, err := r.c.Receive()
	if err != nil {
		go r.reconnect()
		err = fmt.Errorf("Redis: set: %s", err)
		log.Print(err)
		return err
	}
	if v.(string) != "OK" {
		return errors.New("Redis: set: server rejected: " + key)
	}

	return nil
}

func (r *Redis) get(key string, val interface{}) error {
	r.c.Send("GET", key)
	r.c.Flush()
	v, err := r.c.Receive()
	if err != nil {
		go r.reconnect()
		err = fmt.Errorf("Redis: get: %s", err)
		log.Print(err)
		return err
	}
	if v == nil {
		return NotFound
	}

	b := v.([]byte)
	err = json.Unmarshal(b, val)
	if err != nil {
		return fmt.Errorf("Redis: get: %s", err)
	}
	return nil
}
