package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
)

type HandlerFactory func(p []byte) (http.Handler, error)

type HandlerCollection interface {
	LookupHandler(name string) HandlerFactory
	GenerateFromJSON(map[string]interface{}) (http.Handler, error)
}

type DefaultHandlerCollection struct{}

func (h *DefaultHandlerCollection) GenerateFromJSON(json map[string]interface{}) (http.Handler, error) {
	if len(json) <= 0 || len(json) > 1 {
		return nil, fmt.Errorf("Invalid number of handler")
	}

	handlerName, data := extractKey(json)
	hf := h.LookupHandler(handlerName)
	if hf == nil {
		return nil, fmt.Errorf("Unknown handler %s", handlerName)
	}

	return hf(mustMarshal(data))
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic("Did not marshal")
	}
	return data
}

func extractKey(m map[string]interface{}) (string, interface{}) {
	for k, v := range m {
		return k, v
	}
	panic("Should not happen")
}

func (h *DefaultHandlerCollection) LookupHandler(name string) HandlerFactory {
	// Because security
	if name == "LookupHandler" {
		return nil
	}
	v := reflect.ValueOf(h)
	if _, ok := v.Type().MethodByName(name); !ok {
		return nil
	}
	hf, ok := v.MethodByName(name).Interface().(func([]byte) (http.Handler, error))
	if !ok {
		return nil
	}
	return hf
}

func (h *DefaultHandlerCollection) SimpleCode(p []byte) (http.Handler, error) {
	var x int
	if err := json.Unmarshal(p, &x); err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(x))
		w.Write([]byte(http.StatusText(int(x))))
	}), nil
}

func (h *DefaultHandlerCollection) FileServer(p []byte) (http.Handler, error) {
	var x struct {
		Root string `json:"root"`
	}
	if err := json.Unmarshal(p, &x); err != nil {
		return nil, err
	}

	fi, err := os.Stat(x.Root)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, os.ErrInvalid
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(http.Dir(x.Root)).ServeHTTP(w, r)
	}), nil
}

func (h *DefaultHandlerCollection) BasicAuth(p []byte) (http.Handler, error) {
	var x struct {
		Realm string                 `json:"realm"`
		User  string                 `json:"user"`
		Pass  string                 `json:"pass"`
		Next  map[string]interface{} `json:"next"`
	}
	if err := json.Unmarshal(p, &x); err != nil {
		return nil, err
	}
	if x.Realm == "" {
		x.Realm = "authorization"
	}
	if x.User == "" {
		return nil, fmt.Errorf("basic_auth: 'user' required")
	}
	if x.Pass == "" {
		return nil, fmt.Errorf("basic_auth: 'pass' required")
	}
	next, err := h.GenerateFromJSON(x.Next)
	if err != nil {
		return nil, fmt.Errorf("basic_auth: 'next': %s", err)
	}

	s := fmt.Sprintf("%s:%s", x.User, x.Pass)
	encoded := base64.StdEncoding.EncodeToString([]byte(s))
	authorization := "Basic " + encoded

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := r.Header.Get("Authorization"); a == "" || a != authorization {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, x.Realm))
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}), nil
}

/*
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
*/
