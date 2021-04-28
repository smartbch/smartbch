package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

const indexHTML = `
<!DOCTYPE html>
<html>
<head>
<title>smartBCH testnet faucet</title>
</head>
<body>

<h2>Hi, please send to this address 0.01 smart BCH:</h2>
<form action="/sendBCH" method="post">
  <label for="addr">Address:</label><br>
  <input type="text" id="addr" name="addr" size="100"><br>
  <input type="submit" value="Submit">
</form>

</body>
</html>
`

const resultHTML = `
<!DOCTYPE html>
<html>
<head>
<title>smartBCH testnet faucet</title>
</head>
<body>

<h2>Sent! result:</h2>
<code>
%s
</code>

</body>
</html>
`

const reqTMPL = `{
  "jsonrpc": "2.0",
  "method": "eth_sendTransaction",
  "params":[{
    "from": "0x83b1e2268e976d14cde7c23baa94887404fe71a1",
    "to": "%s",
    "gasPrice": "0x0",
    "value": "0x2386F26FC10000"
  }],
  "id":1}'`

const rpcURL = "http://45.32.38.25:8545"

func hello(w http.ResponseWriter, req *http.Request) {
	_, _ = fmt.Fprint(w, indexHTML)
}

func sendBCH(w http.ResponseWriter, req *http.Request) {
	var addr string
	if req.Method == "POST" {
		if err := req.ParseForm(); err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		addr = req.Form.Get("addr")
	} else {
		addr = req.URL.Query().Get("addr")
	}
	fmt.Println("addr:", addr)
	postBody := fmt.Sprintf(reqTMPL, addr)
	fmt.Println("req:", postBody)

	result, err := post(rpcURL, postBody)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
	} else {
		_, _ = w.Write([]byte(fmt.Sprintf(resultHTML, result)))
	}
}

func main() {
	http.HandleFunc("/faucet", hello)
	http.HandleFunc("/sendBCH", sendBCH)

	http.ListenAndServe(":8080", nil)
}

func post(url string, jsonStr string) (string, error) {
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
