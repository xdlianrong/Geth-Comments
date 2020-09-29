package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

const url = "http://localhost:8545"


func postData() bool {
	data,_ := ioutil.ReadFile("unlockaccount.json")
	fmt.Println(string(data))
	resp, err := http.Post(url,
		"application/json",
		bytes.NewBuffer(data))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
	return true
}

func main()  {
	postData()
}