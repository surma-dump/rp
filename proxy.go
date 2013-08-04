package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func proxy(c configuration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := strings.Split(r.Host, ":")[0]
		if _, ok := c[host]; !ok {
			host = defaultHost
		}
		logRequest(r, host)
		c[host].ServeHTTP(w, r)
	}
}

type proxyServer url.URL

func (h proxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.URL.Scheme = h.Scheme
	r.URL.Host = h.Host
	r.RequestURI = ""
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(fmt.Sprintf("HTTP %d: %s unavailable (%s)", http.StatusServiceUnavailable, h.Host, err)))
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
