package main

import (
	"log"
	"net/http"
	"strings"
)

func logRequest(r *http.Request, proxyTo string) {
	log.Printf("%s %s %s", r.Proto, r.Method, r.URL.String())
	log.Printf("\tHost %s", r.Host)
	log.Printf("\tRemoteAddr %s", r.RemoteAddr)
	log.Printf("\tContentLength %d", r.ContentLength)
	log.Printf("\tUserAgent %s", r.UserAgent())
	log.Printf("\tReferer %s", r.Referer())
	log.Printf("\t%d Header(s)", len(r.Header))
	for key, values := range r.Header {
		log.Printf("\t\t%s: %s", key, strings.Join(values, ", "))
	}
	log.Printf("\t%d Cookie(s)", len(r.Cookies()))
	for _, cookie := range r.Cookies() {
		log.Printf("\t\t%s", cookie.String())
	}
	log.Printf("\tProxyTo %s", proxyTo)
}
