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

// require url
var(
	verifyurl = "http://39.99.227.43:1423/verify"
	getpuburl = "http://39.99.227.43:1423/regkey?chainID=1"
	ethurl    = "http://localhost:8545"
)

// unlock publisher eth_account struct
type unlock struct {
	Jsonrpc	 string			`json:"jsonrpc"`
	Method	 string	    	`json:"method"`
	Params   []interface{}  `json:"params"`
	Id       int			`json:"id"`
}

// get result from unlock to ethereum
type unlockget struct {
	Jsonrpc	 string			`json:"jsonrpc"`
	Id       int			`json:"id"`
	Result   bool           `json:"result"`
}

// verify the publickey of usr to regulator
func Verify(publickey string) bool {
	data := make(url.Values)
	data["Hashky"] = []string{publickey}

	resp, err := http.PostForm(verifyurl, data)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body) + ": check publickey right")
	if(string(body) == "True" ){
		return true
	}else{
		return false
	}
}

// get ragulator publickey--struct
func GetRegPub() crypto.PublicKey {
	resp, _ := http.Get(getpuburl)
	defer resp.Body.Close()
	reqBody := crypto.PublicKey{}
	body, _ := ioutil.ReadAll(resp.Body)
	err := json.Unmarshal(body, &reqBody)
	fmt.Println(string(body))
	if err != nil {
		fmt.Println("CreateVMProcess: Unmarshal data failed")
	}
	return reqBody
}

// unlock publisher eth_account
func UnlockAccount(ethaccount string, ethkey string) bool{
	params := make([]interface{}, 3)
	params[0] = ethaccount
	params[1] = ethkey
	params[2] = 30000

	data := unlock{"2.0", "personal_unlockAccount", params,67}

	datapost, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return false
	}
	req, err := http.NewRequest("POST", ethurl, bytes.NewBuffer(datapost))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil{
		fmt.Println(err)
		return false
	}
	defer resp.Body.Close()

	bodyC, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(bodyC))
	var s unlockget;
	json.Unmarshal([]byte(bodyC), &s)
	if(s.Result == true){
		return true
	}else{
		return false
	}
}

func SendTransaction() {

}
