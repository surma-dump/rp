package main

import (
	"bytes"
	"testing"
)

func TestParseConfiguration(t *testing.T) {
	for cfg, ok := range map[string]bool{
		`{}`:                                             false,
		`{"default": null}`:                              false,
		`{"default": ""}`:                                false,
		`{"default": "."}`:                               true,
		`{"default": "$"}`:                               false,
		`{"foo.com": "http://bar.com"}`:                  false, // no default
		`{"default": ".", "foo.com": "http://bar.com"}`:  true,
		`{"default": ".", "foo.com": "bar.com"}`:         false, // no schema
		`{"default": ".", "foo.com": "https://bar.com"}`: true,
		`{"default": ".", "foo.com": "http://bar"}`:      true,
		`{"default": ".", "foo": "http://bar"}`:          true,
	} {
		_, err := parseConfiguration(bytes.NewBufferString(cfg))
		if ok && err != nil {
			t.Errorf("%s: %s", cfg, err)
			continue
		}
		if !ok && err == nil {
			t.Errorf("%s: should be invalid, but wasn't")
			continue
		}
	}
}
