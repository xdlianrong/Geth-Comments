package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"regulator/utils/ElGamal"
	"time"
)

func Hash(str string) string {
	//使用sha256哈希函数
	h := sha256.New()
	h.Write([]byte(str))
	sum := h.Sum(nil)

	//由于是十六进制表示，因此需要转换
	s := hex.EncodeToString(sum)
	fmt.Println(s)
	return s
}
func randContent() string {
	h := md5.New()
	io.WriteString(h, "crazyof.me")
	io.WriteString(h, time.Now().String())
	return fmt.Sprintf("%x", h.Sum(nil))
}
func GenElgKeys() (pub ElGamal.PublicKey, priv ElGamal.PrivateKey, err error) {
	return ElGamal.GenerateKeys(randContent())
}
