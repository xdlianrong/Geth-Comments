package utils

import "encoding/hex"

// the struct receipt to user
type Receipt struct {
	Cmv       string  `json:"cmv"       xml:"cmv"       form:"cmv"       query:"cmv"`
	Epkrc1    string `json:"epkrc1"    xml:"epkrc1"    form:"epkrc1"    query:"epkrc1"`
	Epkrc2    string  `json:"epkrc2"    xml:"epkrc2"    form:"epkrc2"    query:"epkrc2"`
}

// the struct from user post
type Purchase struct {
	Publickey  string `json:"publickey" xml:"publickey" form:"publickey" query:"publickey"`
	Amount     string `json:"amount"    xml:"amount"    form:"amount"    query:"amount"`
}

func byteto0xstring(b []byte) (s string){
	s = "0x" + hex.EncodeToString(b)
	return
}

func Toreceipt(cmv []byte, rc1 []byte, rc2 []byte) (re Receipt){
	re.Cmv    = byteto0xstring(cmv)
	re.Epkrc1 = byteto0xstring(rc1)
	re.Epkrc2 = byteto0xstring(rc2)
	return
}