package utils

import (
	"encoding/hex"
	"math/big"
)

// the struct receipt to user
type Receipt struct {
	Cmv    string `json:"cmv"       xml:"cmv"       form:"cmv"       query:"cmv"`
	Epkrc1 string `json:"epkrc1"    xml:"epkrc1"    form:"epkrc1"    query:"epkrc1"`
	Epkrc2 string `json:"epkrc2"    xml:"epkrc2"    form:"epkrc2"    query:"epkrc2"`
	Hash   string `json:"hash"    xml:"hash"    form:"hash"    query:"hash"`
}

// the struct from user post
type Purchase struct {
	G1     string `json:"g1"        xml:"g1"        form:"g1"        query:"g1"`
	G2     string `json:"g2"        xml:"g2"        form:"g2"        query:"g2"`
	P      string `json:"p"         xml:"p"         form:"p"         query:"p"`
	H      string `json:"h"         xml:"h"         form:"h"         query:"h"`
	Amount string `json:"amount"    xml:"amount"    form:"amount"    query:"amount"`
}

func byteto0xstring(b []byte) (s string) {
	s = "0x" + hex.EncodeToString(b)
	return
}

func Toreceipt(cmv []byte, rc1 []byte, rc2 []byte, hash string) (re Receipt) {
	re.Cmv = byteto0xstring(cmv)
	re.Epkrc1 = byteto0xstring(rc1)
	re.Epkrc2 = byteto0xstring(rc2)
	re.Hash = hash
	return
}

func stringtobig(s string) (b *big.Int) {
	b = new(big.Int)
	b, _ = b.SetString(s, 10)
	return
}
