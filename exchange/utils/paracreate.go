package utils

import (
	"echo-demo/crypto"
	"echo-demo/params"
	"math/rand"
	"strconv"
)

type Purchase struct {
	Publickey  string `json:"publickey" xml:"publickey" form:"publickey" query:"publickey"`
	Amount string `json:"amount" xml:"amount" form:"amount" query:"amount"`
}

func CreateCM_v(amount int)  {
	r :=  rand.Uint64()
	regPub := crypto.PublicKey{params.RegularG1,params.RegularG2,params.RegularBigPrimeNumber,params.RegularPublicKey}
	r1 := strconv.FormatUint(r, 10)
	regPub.CommitByUint64(uint64(amount), []byte(r1))
}


