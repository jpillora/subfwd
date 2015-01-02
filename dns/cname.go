package main

import (
	"log"
	"net"
)

func main() {
	cname, err := net.LookupCNAME("asdasdasdasdasd.jilarra.com")

	if err != nil {
		log.Print("err: " + err.Error())
	} else {
		log.Print(cname)
	}

	txts, err := net.LookupTXT("subfwd.com")

	if err != nil {
		log.Print("err: " + err.Error())
	} else {
		for _, t := range txts {
			log.Print(t)
		}
	}

}
