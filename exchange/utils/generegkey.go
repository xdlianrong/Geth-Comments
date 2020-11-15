package utils

import (
	"encoding/json"
	"exchange/crypto"
	"fmt"
	"math/big"
	"os"
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
		fmt.Println("Open file failed [Err:%s]", err.Error())
		return
	}
	decoder := json.NewDecoder(filePtr)
	info := []RegulatorInfo{}
	err = decoder.Decode(&info)
	if err != nil {
		defer filePtr.Close()
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
