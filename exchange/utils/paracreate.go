package utils

import (
	"exchange/crypto"
	"math/rand"
	"strconv"
)

// the struct from user post
type Purchase struct {
	Publickey  string `json:"publickey" xml:"publickey" form:"publickey" query:"publickey"`
	Amount     string `json:"amount"    xml:"amount"    form:"amount"    query:"amount"`
}

// create commit_v
func CreateCM_v(regpub crypto.PublicKey, amount string) (CM crypto.Commitment) {
	amounts, _ := strconv.Atoi(amount)
	r_f :=  rand.Uint64()
	r1 := strconv.FormatUint(r_f, 10)
	CM = regpub.CommitByUint64(uint64(amounts), []byte(r1))
	return
}

// create elgamal result
func CreateElgamalC(regpub crypto.PublicKey, amount string, publickey string) (C crypto.CypherText) {
	M := amount + publickey
	C  = crypto.Encrypt(regpub, []byte(M))
	return
}

// create sign result
func CreateSign(privpub crypto.PrivateKey, amount string) (sig crypto.Signature) {
	sig = crypto.Sign(privpub, []byte(amount))
	return
}
