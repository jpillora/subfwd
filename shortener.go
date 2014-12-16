package main

import (
	"log"
	"os"

	"github.com/jpillora/shortener/lib"
)

//run it
func main() {
	s := shortener.New()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Fatal(s.ListenAndServe(":" + port))
}
