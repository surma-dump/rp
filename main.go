package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	var (
		config = flag.String("config", "rp.json", "configuration file")
		listen = flag.String("listen", ":80", "HTTP server address")
	)
	flag.Parse()

	f, err := os.Open(*config)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()



	c := configuration{}
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		log.Fatal(err)
	}

	c.install(http.DefaultServeMux)
	log.Printf("listening on %s", *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

type configuration map[string]http.Handler

func (c configuration) UnmarshalJSON(p []byte) error {
	var m map[string]map[string]interface{}
	if err := json.Unmarshal(p, &m); err != nil {
		return err
	}
	for hosts, submap := range m {
		handler, err := unmarshalHandler(submap)
		if err != nil {
			return fmt.Errorf("%s: %s", hosts, err)
		}
		for _, host := range strings.Split(hosts, ",") {
			c[host] = handler
		}
	}
	return nil
}

func (c configuration) install(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		host := strings.Split(r.Host, ":")[0]
		h, ok := c[host]
		if !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		h.ServeHTTP(w, r)
	})
}
