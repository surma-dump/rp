package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

const defaultHost = "default"

type configuration map[string]http.Handler

func parseConfiguration(r io.Reader) (configuration, error) {
	var m map[string]string
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return configuration{}, err
	}
	if len(m) <= 0 {
		return configuration{}, fmt.Errorf("no configuration hosts")
	}
	if _, ok := m[defaultHost]; !ok {
		return configuration{}, fmt.Errorf("nothing configured for '%s' host", defaultHost)
	}

	c := configuration{}
	for host, s := range m {
		fi, err := os.Stat(s)
		if err == nil && fi.IsDir() {
			c[host] = http.FileServer(http.Dir(s))
			continue
		}
		url, err := url.Parse(s)
		if err == nil && url.Host != "" {
			c[host] = proxyServer(*url)
			continue
		}
		return configuration{}, fmt.Errorf("%s: can't parse '%s'", host, s)
	}

	return c, nil
}
