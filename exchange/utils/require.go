package utils

import (
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

)


func Verify(publickey string) bool {
	url_regular := verifyurl

	data := make(url.Values)
	data["Hashky"] = []string{publickey}

	resp, err := http.PostForm(url_regular, data)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer resp.Body.Close()
	bodyC, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(bodyC))
	//TODO: talk to regulator
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

func sendTransaction() {

}
