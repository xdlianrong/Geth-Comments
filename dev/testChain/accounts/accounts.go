package accounts

import (
	"fmt"
	"math/big"
)

type Account struct {
	Pub  PublicKey  `json:"Pub"`
	Priv PrivateKey `json:"Priv"`
	Info struct {
		Name    string `json:"Name"`
		ID      string `json:"ID"`
		Hashky  string `json:"Hashky"`
		ExtInfo string `json:"ExtInfo"`
	} `json:"Info"`
}

func GenerateAccount(randString string, name string, id string, extInfo string) Account {
	pub, priv, _ := GenerateKeys(randString)
	fmt.Println("生成账户"+name, "私钥：", priv.X.String())
	return Account{
		Pub:  pub,
		Priv: priv,
		Info: struct {
			Name    string `json:"Name"`
			ID      string `json:"ID"`
			Hashky  string `json:"Hashky"`
			ExtInfo string `json:"ExtInfo"`
		}{
			Name:    name,
			ID:      id,
			Hashky:  pub.H.String(),
			ExtInfo: extInfo,
		},
	}
}

func DecryptValue(priv PrivateKey, C CypherText) (v uint64) {
	// ElGamal 解密 C 输出 类型数据 M，输出密文 C
	// 解密
	c1 := new(big.Int).SetBytes(C.C1)
	c2 := new(big.Int).SetBytes(C.C2)
	c1x := new(big.Int).Exp(c1, priv.X, priv.P)
	c1x.ModInverse(c1x, priv.P)
	gv := new(big.Int).Mul(c2, c1x)
	gv.Mod(gv, priv.P)
	for i := 1; true; i++ {
		gi := new(big.Int).Exp(priv.G1, new(big.Int).SetInt64(int64(i)), priv.P)
		if gv.Cmp(gi) == 0 {
			return uint64(i)
		}
		if i >= 262144 {
			fmt.Printf("该承诺并非价值承诺或承诺价值大于262144")
			return 0
		}
	}
	return
}
