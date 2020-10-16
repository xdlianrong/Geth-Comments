package ElGamal

import (
	"crypto/sha256"
	"math/big"
	"math/rand"
	"strconv"
	"time"
)

// PublicKey 公钥
type PublicKey struct {
	G1, G2, P, H *big.Int
}

// PrivateKey 私钥
type PrivateKey struct {
	PublicKey
	X *big.Int
}

func GenerateKeys(info string) (pub PublicKey, priv PrivateKey, err error) {
	// 本函数用于根据用户信息 string 生成一对公私钥 pub 和 priv
	// 从质数表中随机选择大质数P
	var error_bool bool
	pub.P, error_bool = new(big.Int).SetString(select_prime(), 16)
	if !error_bool {
		return
	}
	priv.P = pub.P

	// 使用string Hash生成G1
	pub.G1 = new(big.Int)
	HashInfoBuf := sha256.Sum256([]byte(info))
	HashInfo := HashInfoBuf[:]
	pub.G1.SetBytes(HashInfo)
	pub.G1.Mod(pub.G1, pub.P)
	for {
		gcd := new(big.Int).GCD(nil, nil, pub.G1, pub.P).Int64()
		if gcd == 1 {
			break
		}
		pub.G1.Sub(pub.G1, new(big.Int).SetInt64(1))
	}
	priv.G1 = pub.G1

	// 使用string time Hash生成G2
	pub.G2 = new(big.Int)
	now := time.Now().Unix()
	stringNow := []byte(strconv.FormatInt(now, 10))
	HashInfo = append(HashInfo, stringNow...)
	pub.G2.SetBytes(HashInfo)
	pub.G2.Mod(pub.G2, pub.P)
	for {
		gcd := new(big.Int).GCD(nil, nil, pub.G2, pub.P).Int64()
		if gcd == 1 {
			break
		}
		pub.G2.Sub(pub.G2, new(big.Int).SetInt64(1))
	}
	priv.G2 = pub.G2

	// 随机选择私钥 X
	priv.X = new(big.Int)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	priv.X.Rand(rnd, pub.P)

	// 计算公钥 H
	pub.H = new(big.Int)
	pub.H.Exp(pub.G2, priv.X, pub.P)
	priv.H = pub.H

	return
}
