package main

import (
	"log"
	"net"
)

// txts, err := net.LookupTXT("subfwd-default.jpillora.com")

// if err != nil {
// 	log.Print("err: " + err.Error())
// } else {
// 	for _, t := range txts {
// 		log.Print(t)
// 	}
// }

func main() {

	cname, err := net.LookupCNAME("asdasdad.jpillora.com")
	if err != nil {
		log.Print("err: " + err.Error())
	} else {
		log.Print(cname)
	}

	// config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
	// c := new(dns.Client)

	// m := new(dns.Msg)
	// m.SetQuestion(dns.Fqdn("foobar.jpillora.com"), dns.TypeCNAME)
	// m.RecursionDesired = true

	// server := net.JoinHostPort(config.Servers[0], config.Port)

	// r, _, err := c.Exchange(m, server)
	// if r == nil {
	// 	log.Fatalf("*** error: %s\n", err.Error())
	// }

	// if r.Rcode != dns.RcodeSuccess {
	// 	log.Fatalf(" *** invalid answer\n")
	// }
	// for _, a := range r.Answer {
	// 	fmt.Printf("%v\n", a.Header().String())
	// }
}
