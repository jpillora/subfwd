package main

import (
	"log"
	"os"

	"github.com/jpillora/subfwd/lib"
)

//run it
func main() {
	s := subfwd.New()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Fatal(s.ListenAndServe(":" + port))
}
