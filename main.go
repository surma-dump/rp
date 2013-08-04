package main

import (
	"flag"
	"log"
	"net/http"
	"os"
)

var (
	config = flag.String("config", "rp.json", "config file")
	listen = flag.String("listen", ":8080", "HTTP listen address")
)

func main() {
	flag.Parse()
	log.SetFlags(log.Lmicroseconds)

	f, err := os.Open(*config)
	if err != nil {
		log.Fatal(err)
	}
	c, err := parseConfiguration(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on %s", *listen)
	log.Fatal(http.ListenAndServe(*listen, proxy(c)))
}
