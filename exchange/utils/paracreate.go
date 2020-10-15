package utils

import (
	"echo-demo/crypto"
	"echo-demo/params"
	"math/rand"
	"strconv"
)

func CreateCM_v(publickey string,amount int)  {
	r :=  rand.Uint64()
	regPub := crypto.PublicKey{params.RegularG1,params.RegularG2,params.RegularBigPrimeNumber,params.RegularPublicKey}
	usrPub, _ := strconv.Atoi(publickey)
	r1 := strconv.FormatUint(r, 10)
	regPub.CommitByUint64(uint64(usrPub), []byte(r1))
}


