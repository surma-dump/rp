package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/streadway/handy/report"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func unmarshalHandler(submap map[string]interface{}) (http.Handler, error) {
	if len(submap) <= 0 {
		return nil, fmt.Errorf("no handler declared")
	}
	if len(submap) > 1 {
		return nil, fmt.Errorf("multiple handlers declared")
	}
	for k, v := range submap {
		buf, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", k, err)
		}

		switch k {
		case "file_server":
			var h fileServer
			if err := json.Unmarshal(buf, &h); err != nil {
				return nil, fmt.Errorf("%s: %s", k, err)
			}
			return h, nil

		case "redirect":
			var h redirect
			if err := json.Unmarshal(buf, &h); err != nil {
				return nil, fmt.Errorf("%s: %s", k, err)
			}
			return h, nil

		case "handy_report":
			var h handyReport
			if err := json.Unmarshal(buf, &h); err != nil {
				return nil, fmt.Errorf("%s: %s", k, err)
			}
			return h, nil

		case "simple_log":
			var h simpleLog
			if err := json.Unmarshal(buf, &h); err != nil {
				return nil, fmt.Errorf("%s: %s", k, err)
			}
			return h, nil

		case "basic_auth":
			var h basicAuth
			if err := json.Unmarshal(buf, &h); err != nil {
				return nil, fmt.Errorf("%s: %s", k, err)
			}
			return h, nil

		case "simple_code":
			var h simpleCode
			if err := json.Unmarshal(buf, &h); err != nil {
				return nil, fmt.Errorf("%s: %s", k, err)
			}
			return h, nil

		default:
			return nil, fmt.Errorf("invalid handler '%s'", k)
		}
	}
	panic("impossible")
}

type simpleCode int

func (h *simpleCode) UnmarshalJSON(p []byte) error {
	var x int
	if err := json.Unmarshal(p, &x); err != nil {
		return err
	}
	*h = simpleCode(x)
	return nil
}

func (h simpleCode) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(int(h))
	w.Write([]byte(http.StatusText(int(h))))
}

type fileServer struct {
	root string
}

func (h *fileServer) UnmarshalJSON(p []byte) error {
	var x struct {
		Root string `json:"root"`
	}
	if err := json.Unmarshal(p, &x); err != nil {
		return err
	}

	fi, err := os.Stat(x.Root)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return os.ErrInvalid
	}

	h.root = x.Root
	return nil
}

func (h fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.Dir(h.root)).ServeHTTP(w, r)
}

type redirect struct {
	to   string
	code int // default: http.StatusMovedPermanently
}

func (h *redirect) UnmarshalJSON(p []byte) error {
	var x struct {
		To   string `json:"to"`
		Code int    `json:"code"`
	}
	if err := json.Unmarshal(p, &x); err != nil {
		return err
	}
	if _, err := url.Parse(x.To); err != nil {
		return err
	}
	if x.Code == 0 {
		x.Code = http.StatusMovedPermanently
	}

	h.to = x.To
	h.code = x.Code
	return nil
}

func (h redirect) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.RedirectHandler(h.to, h.code)
}

type handyReport struct {
	next http.Handler
}

func (h *handyReport) UnmarshalJSON(p []byte) error {
	var x map[string]interface{}
	if err := json.Unmarshal(p, &x); err != nil {
		return err
	}
	next, err := unmarshalHandler(x)
	if err != nil {
		return err
	}

	h.next = next
	return nil
}

func (h handyReport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	report.JSON(os.Stdout, h.next).ServeHTTP(w, r)
}

type simpleLog struct {
	next http.Handler
}

func (h *simpleLog) UnmarshalJSON(p []byte) error {
	var x map[string]interface{}
	if err := json.Unmarshal(p, &x); err != nil {
		return err
	}
	next, err := unmarshalHandler(x)
	if err != nil {
		return err
	}

	h.next = next
	return nil
}

func (h simpleLog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	h.next.ServeHTTP(w, r)
}

type basicAuth struct {
	realm         string
	authorization string
	next          http.Handler
}

func (h *basicAuth) UnmarshalJSON(p []byte) error {
	var x struct {
		Realm string                 `json:"realm"`
		User  string                 `json:"user"`
		Pass  string                 `json:"pass"`
		Next  map[string]interface{} `json:"next"`
	}
	if err := json.Unmarshal(p, &x); err != nil {
		return err
	}
	if x.Realm == "" {
		x.Realm = "authorization"
	}
	if x.User == "" {
		return fmt.Errorf("basic_auth: 'user' required")
	}
	if x.Pass == "" {
		return fmt.Errorf("basic_auth: 'pass' required")
	}
	next, err := unmarshalHandler(x.Next)
	if err != nil {
		return fmt.Errorf("basic_auth: 'next': %s", err)
	}

	s := fmt.Sprintf("%s:%s", x.User, x.Pass)
	encoded := base64.StdEncoding.EncodeToString([]byte(s))
	authorization := "Basic " + encoded
	h.realm = x.Realm
	h.authorization = authorization
	h.next = next
	return nil
}

func (h basicAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a := r.Header.Get("Authorization"); a == "" || a != h.authorization {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, h.realm))
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	h.next.ServeHTTP(w, r)
}
