package main

import (
	"bytes"
	"encoding/json"

	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
)

type Pub struct {
	G1 *big.Int
	G2 *big.Int
	P *big.Int
	H *big.Int
}

const url = "http://localhost:8545"
const buyurl = "http://localhost:1323/buy"
const puburl = "http://localhost:1323/pubpub"

func postData() bool {
	path := "getTx.json"
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
	return true
}

func exchange() bool {
	path := "exchange.json"
	data,_ := ioutil.ReadFile(path)
	resp, err := http.Post(buyurl,
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

func getData() bool {
	client := &http.Client{}
	resp, err := client.Get(puburl)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	exPub := Pub{}
	json.Unmarshal(body,&exPub)
	fmt.Println("G1: ",exPub.G1,"G2: ",exPub.G2,"P: ",exPub.P,"H： ",exPub.H)

	return true
}

func sendTransaction() bool{
	path := "sendTransaction.json"
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
	return true
}

func main()  {
	sendTransaction()
	//postData()
	//getData()
	exchange()
}