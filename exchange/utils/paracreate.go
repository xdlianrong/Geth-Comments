package utils

import (
	"exchange/crypto"
	"math/rand"
	"strconv"
)

// create commit_v
func CreateCM_v(regpub crypto.PublicKey, amount string) (CM crypto.Commitment) {
	amounts, _ := strconv.Atoi(amount)
	rF := rand.Uint64()
	r1 := strconv.FormatUint(rF, 10)
	CM = regpub.CommitByUint64(uint64(amounts), []byte(r1))
	return
}

// create elgamal result
func CreateElgamalInfo(regpub crypto.PublicKey, amount string, publickey string) (C crypto.CypherText) {
	M := publickey + amount
	C = crypto.Encrypt(regpub, []byte(M))
	return
}

func CreateElgamalR(regpub crypto.PublicKey, r []byte) (C crypto.CypherText) {
	C = crypto.Encrypt(regpub, r)
	return
}

// create sign result
func CreateSign(privpub crypto.PrivateKey, amount string) (sig crypto.Signature) {
	ID := "1"
	sig = crypto.Sign(privpub, []byte(ID+amount))
	return
}

func CreateUsrPub(g1 string, g2 string, p string, h string) (usrpub crypto.PublicKey) {
	usrpub.G1 = stringtobig(g1)
	usrpub.G2 = stringtobig(g2)
	usrpub.P = stringtobig(p)
	usrpub.H = stringtobig(h)
	return
}
