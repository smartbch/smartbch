package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

func getQueryParam(req *http.Request, key string) (string, error) {
	if req.Method == "POST" {
		if err := req.ParseForm(); err != nil {
			return "", err
		}
		return req.Form.Get(key), nil
	}

	return req.URL.Query().Get(key), nil
}

func sendPost(url string, jsonStr string) (string, error) {
	body := bytes.NewReader([]byte(jsonStr))
	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), err
}
