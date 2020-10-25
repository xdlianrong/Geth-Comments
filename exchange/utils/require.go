package utils

import (
	"bytes"
	"encoding/json"
	"exchange/crypto"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

var(
	verifyurl = "http://localhost:1423/verify"
	getpuburl = "http://localhost:1423/regkey?chainID=1"
	ethurl    = "http://localhost:8545"
)

type unlock struct {
	Jsonrpc	 string			`json:"jsonrpc"`
	Method	 string	    	`json:"method"`
	Params   []interface{}  `json:"params"`
	Id       int			`json:"id"`
}

func Verify(publickey string) bool {
	data := make(url.Values)
	data["Hashky"] = []string{publickey}

	resp, err := http.PostForm(verifyurl, data)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer resp.Body.Close()
	bodyC, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(bodyC))
	if(string(bodyC) == "True" ){
		return true
	}else{
		return false
	}
}

func GetRegPub() crypto.PublicKey {
	resp, _ := http.Get(getpuburl)
	defer resp.Body.Close()
	reqBody := crypto.PublicKey{}
	body, _ := ioutil.ReadAll(resp.Body)
	err := json.Unmarshal(body, &reqBody)
	if err != nil {
		fmt.Println("CreateVMProcess: Unmarshal data failed")
	}
	return reqBody
}

func UnlockAccount(ethaccount string, ethkey string) bool{
	params := make([]interface{}, 3)
	params[0] = ethaccount
	params[1] = ethkey
	params[2] = 30000

	data := unlock{"2.0", "personal_unlockAccount", params,67}

	datapost, err := json.Marshal(data)
	if err != nil {
		fmt.Println("err = ", err)
		return true
	}
	req, err := http.NewRequest("POST", ethurl, bytes.NewBuffer(datapost))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil{
		panic(err)
		return false
	}
	defer resp.Body.Close()
	bodyC, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(bodyC))
	return true
}

func SendTransaction() {

}
