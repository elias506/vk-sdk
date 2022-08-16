package vk_sdk

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

func setInt(vs url.Values, key string, v int) {
	vs.Set(key, strconv.Itoa(v))
}

func setInts(vs url.Values, key string, v []int) {
	if len(v) == 0 {
		return
	}

	sInts := make([]string, 0, len(v))

	for _, i := range v {
		sInts = append(sInts, strconv.Itoa(i))
	}

	s := strings.Join(sInts, ",")

	vs.Set(key, s)
}

func setFloat(vs url.Values, key string, v float64) {
	vs.Set(key, strconv.FormatFloat(v, 'f', 6, 64))
}

func setFloats(vs url.Values, key string, v []float64) {
	if len(v) == 0 {
		return
	}

	sFloats := make([]string, 0, len(v))

	for _, f := range v {
		sFloats = append(sFloats, strconv.FormatFloat(f, 'f', 6, 64))
	}

	s := strings.Join(sFloats, ",")

	vs.Set(key, s)
}

func setString(vs url.Values, key string, v string) {
	vs.Set(key, v)
}

func setStrings(vs url.Values, key string, v []string) {
	if len(v) == 0 {
		return
	}

	vs.Set(key, strings.Join(v, ","))
}

func setBool(vs url.Values, key string, v bool) {
	vs.Set(key, strconv.FormatBool(v))
}

func setJSON(vs url.Values, key string, v interface{}) error {
	s, err := json.Marshal(v)

	if err != nil {
		return err
	}

	vs.Set(key, string(s))

	return nil
}

func setOptions(vs url.Values, options []Option) {
	for _, opt := range options {
		vs.Set(opt.name, opt.value)
	}
}
