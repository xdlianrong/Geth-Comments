package utils

import (
	"encoding/json"
	"exchange/crypto"
	"fmt"
	"math/big"
	"os"
)

type PublisherInfo struct {
	G1             string `json:"G1"`
	G2             string `json:"G2"`
	Bigprimenumber string `json:"bigprimenumber"`
	Publickey      string `json:"publickey"`
	Privatekey     string `json:"privatekey"`
}

func GenerateKey(gk string) (pub crypto.PublicKey, priv crypto.PrivateKey, err error) {
	filePtr, err := os.OpenFile("info.json", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		fmt.Printf("Open file failed [Err:%s]\n", err.Error())
		return
	}
	decoder := json.NewDecoder(filePtr)
	var info []PublisherInfo
	err = decoder.Decode(&info)
	if err != nil {
		defer filePtr.Close()
		pub, priv, err = crypto.GenerateKeys(gk)
		info = []PublisherInfo{{(*pub.G1).String(), (*pub.G2).String(), (*pub.P).String(), (*pub.H).String(), (*priv.X).String()}}
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
		priv.X, _ = new(big.Int).SetString(info[0].Privatekey, 10)
		priv.G1, priv.G2, priv.H, priv.P = pub.G1, pub.G2, pub.H, pub.P
	}
	return
}
