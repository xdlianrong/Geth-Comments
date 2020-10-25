package main

import (
	"bytes"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
	"strconv"
)

const url = "http://localhost:8545"

// unlockAccount.json
// sendTransaction.json
func postData() bool {
	path := "u.json"
	data,_ := ioutil.ReadFile(path)
	resp, err := http.Post(url,
		"application/json",
		bytes.NewBuffer(data))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))

	if path == "sendTransaction.json"{
		json := string(body)
		value := gjson.Get(json,"result")
		int64, _ := strconv.ParseInt(value.Value().(string), 10, 64)
		fmt.Println("total txs: ",int64)
	}


	return true
}

func main()  {
	postData()
}