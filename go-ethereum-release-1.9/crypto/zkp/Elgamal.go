package zkp

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"math/rand"
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

// 签名
type Signature struct {
	M, M_hash, R, S []byte
}

func Encrypt(pub PublicKey, M []byte) (C CypherText) {
	// ElGamal 加密 []byte 类型数据 M，输出密文 C
	// 构造C1,C2
	C1 := make([]byte, 0, len(M))
	C2 := make([]byte, 0, len(M))
	// 对明文 M 切片并进行分片处理，每片长 28 bytes（224 bits）
	n := (len(M) + 27) / 28
	for i := 0; i < n; i++ {

		// 明文切片
		var m_bytes []byte
		if i == n-1 {
			m_bytes = M[i*28:]
		} else {
			m_bytes = M[i*28 : (i+1)*28]
		}

		limit := new(big.Int).Sub(pub.P, new(big.Int).SetInt64(4))

		// 生成随机数 k
		rnd_key := 0
		k := new(big.Int)
		for {
			rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)*8 + int64(rnd_key)))
			k.Rand(rnd, limit)
			k.Add(k, new(big.Int).SetInt64(2))
			gcd := (int)(new(big.Int).GCD(nil, nil, k, pub.P).Int64())
			if gcd == 1 {
				break
			}
			rnd_key++
		}

		// 加密
		m := new(big.Int).SetBytes(m_bytes)
		c1 := new(big.Int).Exp(pub.G2, k, pub.P)
		s := new(big.Int).Exp(pub.H, k, pub.P)
		c2 := s.Mul(s, m)
		c2.Mod(c2, pub.P)

		//规范化C1,C2
		var c1_bytes [32]byte
		var c2_bytes [32]byte
		copy(c1_bytes[(32-len(c1.Bytes())):], c1.Bytes())
		copy(c2_bytes[(32-len(c2.Bytes())):], c2.Bytes())

		// 填入 C1,C2
		C1 = append(C1, c1_bytes[:]...)
		C2 = append(C2, c2_bytes[:]...)
	}

	// 合并 C1,C2
	C = CypherText{C1, C2}
	return
}

func Verify(pub PublicKey, sig Signature) bool {
	// 验签算法[1，验证sig.M是否为M的哈希，2.验证签名是否正确]
	// 1.验证哈希是否正确
	hash_m := sha256.Sum256(sig.M)
	hash_m_fix := hash_m[:]
	if len(hash_m_fix) != len(sig.M_hash) {
		return false
	}
	for i:=0;i<len(sig.M_hash);i++{
		if hash_m_fix[i] != sig.M_hash[i] {
			fmt.Println("\n消息哈希错误！")
			return false
		}
	}

	// 2.验证签名是否正确
	m_0 := new(big.Int).SetBytes(sig.M_hash)
	m := new(big.Int).Mod(m_0, pub.P)
	r := new(big.Int).SetBytes(sig.R)
	s := new(big.Int).SetBytes(sig.S)
	hr := new(big.Int).Exp(pub.H, r, pub.P)
	rs := new(big.Int).Exp(r, s, pub.P)
	gm := new(big.Int).Exp(pub.G2, m, pub.P)
	hr.Mul(hr, rs)
	hr.Mod(hr, pub.P)
	return hr.Cmp(gm) == 0
}
