package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/jpillora/opts"
	"github.com/jpillora/subfwd/lib"
)

var VERSION = "0.0.0-src"

type config struct {
	Port string `help:"listening port" env:"PORT"`
}

func main() {
	c := config{Port: "3000"}
	opts.New(&c).Version(VERSION).Parse()

	rand.Seed(time.Now().UnixNano())
	s := subfwd.New()

	log.Fatal(s.ListenAndServe(c.Port))
}
