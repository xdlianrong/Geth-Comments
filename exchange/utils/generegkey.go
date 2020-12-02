package utils

import (
	"encoding/json"
	"exchange/crypto"
	"exchange/params"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strings"
)

type RegulatorInfo struct {
	G1             string `json:"G1"`
	G2             string `json:"G2"`
	Bigprimenumber string `json:"bigprimenumber"`
	Publickey      string `json:"publickey"`
}

func GenerateRegKey() (pub crypto.PublicKey, err error) {
	filePtr, err := os.OpenFile("regpub.json", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		defer filePtr.Close()
		fmt.Printf("Open file failed [Err:%s]\n", err.Error())
		return
	}
	decoder := json.NewDecoder(filePtr)
	var info []RegulatorInfo
	err = decoder.Decode(&info)
	if err != nil {
		pub = GetRegPub()
		info = []RegulatorInfo{{(*pub.G1).String(), (*pub.G2).String(), (*pub.P).String(), (*pub.H).String()}}
		encoder := json.NewEncoder(filePtr)
		err = encoder.Encode(info)
		if err != nil {
			fmt.Println("Encoder failed", err.Error())
		} else {
			fmt.Println("Encoder success")
		}
	} else {
		pub.G1, _ = new(big.Int).SetString(info[0].G1, 10)
		pub.G2, _ = new(big.Int).SetString(info[0].G2, 10)
		pub.P, _ = new(big.Int).SetString(info[0].Bigprimenumber, 10)
		pub.H, _ = new(big.Int).SetString(info[0].Publickey, 10)
	}
	return
}

func SetRegulator() (pub crypto.PublicKey) {
	//做网络请求获取监管者公钥
	client := &http.Client{}
	resp, _ := client.Get(params.Getpuburl)
	if resp != nil {
		defer resp.Body.Close()
	} else {
		fmt.Println("Failed to connect to regulator server")
		return
	}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &pub)
	if pub.G1 == nil || pub.G2 == nil || pub.P == nil || pub.H == nil {
		fmt.Println("Failed to connect to regulator server")
	} else {
		pubK := strings.Replace(string(body), "\"", "", -1)
		pubK = strings.Replace(pubK, "\n", "", -1)
		fmt.Println("Succeed to connect to regulator server", "publicKey", pubK)
	}
	return pub
}
