package utils

import (
	"bytes"
	"encoding/json"
	"exchange/crypto"
	"exchange/params"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

// unlock publisher eth_account struct
type toETH struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

// get result from unlock to ethereum
type unlockget struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  bool   `json:"result"`
}

type SendTx struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Value    string `json:"value"`
	ID       string `json:"id"`
	EpkrC1   string `json:"epkrc1"`
	EpkrC2   string `json:"epkrc2"`
	EpkpC1   string `json:"epkpc1"`
	EpkpC2   string `json:"epkpc2"`
	SigM     string `json:"sigm"`
	SigMHash string `json:"sigmhash"`
	SigR     string `json:"sigr"`
	SigS     string `json:"sigs"`
	CmV      string `json:"cmv"`
}

// get result from send exchangetx to ethereum
type SendTxget struct {
	Jsonrpc string   `json:"jsonrpc"`
	Id      int      `json:"id"`
	Result  string   `json:"result"`
	Error   string   `json:"error"`
}

// verify the publickey of usr to regulator
func Verify(publickey string) bool {
	data := make(url.Values)
	data["Hashky"] = []string{publickey}

	resp, err := http.PostForm(params.Verifyurl, data)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body) + ": check publickey right")
	if string(body) == "True" {
		return true
	} else {
		return false
	}
}

// get ragulator publickey--struct
func GetRegPub() crypto.PublicKey {
	resp, _ := http.Get(params.Getpuburl)
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
func UnlockAccount(ethaccount string, ethkey string) bool {
	paramsul := make([]interface{}, 3)
	paramsul[0] = ethaccount
	paramsul[1] = ethkey
	paramsul[2] = 30000

	data := toETH{"2.0", "personal_unlockAccount", paramsul, 67}

	datapost, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return false
	}
	req, err := http.NewRequest("POST", params.Ethurl, bytes.NewBuffer(datapost))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer resp.Body.Close()

	bodyC, _ := ioutil.ReadAll(resp.Body)
	log.SetPrefix("【AccountCenter】")
	var s unlockget
	json.Unmarshal([]byte(bodyC), &s)
	if s.Result == true {
		log.Println(string(bodyC),"Succeed to unlock account",ethaccount)
		return true
	} else {
		log.Println(string(bodyC),"Failed to unlock account",ethaccount)
		return false
	}
}

// send exchange tx to eth
func SendTransaction(elgamalinfo crypto.CypherText, elgamalr crypto.CypherText, sig crypto.Signature, cm crypto.Commitment, ethaccount string) bool {
	paramstx := make([]interface{}, 1)
	epkrc1 := byteto0xstring(elgamalr.C1)
	epkrc2 := byteto0xstring(elgamalr.C2)
	epkpc1 := byteto0xstring(elgamalinfo.C1)
	epkpc2 := byteto0xstring(elgamalinfo.C2)
	sigm := byteto0xstring(sig.M)
	sigmhash := byteto0xstring(sig.M_hash)
	sigr := byteto0xstring(sig.R)
	sigs := byteto0xstring(sig.S)
	cmv := byteto0xstring(cm.Commitment)
	//epkrc1 = strings.TrimLeft(epkrc1, "0x")
	//fmt.Println(hex.DecodeString(epkrc1))
	paramstx[0] = SendTx{ethaccount, params.Ethto, "0x0", "0x0", "0x0", "0x1", epkrc1, epkrc2, epkpc1, epkpc2, sigm, sigmhash, sigr, sigs, cmv}
	data := toETH{"2.0", "eth_sendTransaction", paramstx, 67}
	fmt.Println(data)
	datapost, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return false
	}
	req, err := http.NewRequest("POST", params.Ethurl, bytes.NewBuffer(datapost))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer resp.Body.Close()
	bodyC, _ := ioutil.ReadAll(resp.Body)
	var s SendTxget
	json.Unmarshal([]byte(bodyC), &s)
	if s.Result != "" {
		log.Println(string(bodyC),"Succeed to send exchangetx",s.Result)
		return true
	} else {
		log.Println(string(bodyC),"Failed to send exchangetx",ethaccount)
		return false
	}
}
