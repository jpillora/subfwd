package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jpillora/subfwd/lib"
)

//run it
func main() {
	rand.Seed(time.Now().UnixNano())
	s := subfwd.New()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Fatal(s.ListenAndServe(":" + port))
}
