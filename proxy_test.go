package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProxyServer(t *testing.T) {
	m := http.NewServeMux()
	response := "OK"
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(response)) })
	s1 := httptest.NewServer(m)
	defer s1.Close()

	u, _ := url.Parse(s1.URL)
	s2 := httptest.NewServer(proxyServer(*u))
	defer s2.Close()

	resp, err := http.Get(s2.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	buf, _ := ioutil.ReadAll(resp.Body)
	if expected, got := response, string(buf); expected != got {
		t.Fatalf("GET: expected '%s', got '%s'", expected, got)
	}
}
