package crypto

import (
	"crypto/sha256"
	"errors"
	"fmt"
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

// 加密文本
type CypherText struct {
	C1,C2 []byte
}

// 签名
type Signature struct {
	M, M_hash, R, S []byte
}

func GenerateKeys(info string) (pub PublicKey, priv PrivateKey, err error) {
	// 本函数用于根据用户信息 string 生成一对公私钥 pub 和 priv
	// 从质数表中随机选择大质数P
	var error_bool bool
	pub.P, error_bool = new(big.Int).SetString(select_prime(), 16)
	if !error_bool {
		err = errors.New("大质数P生成错误，请检查质数表 prime_number_list.go")
		return
	}
	priv.P = pub.P

	// 使用string Hash生成G1
	pub.G1 = new(big.Int)
	HashInfoBuf := sha256.Sum256([]byte(info))
	HashInfo := HashInfoBuf[:]
	pub.G1.SetBytes(HashInfo)
	pub.G1.Mod(pub.G1,pub.P)
	for {
		gcd := new(big.Int).GCD(nil,nil,pub.G1,pub.P).Int64()
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
	pub.G2.Mod(pub.G2,pub.P)
	for {
		gcd := new(big.Int).GCD(nil,nil,pub.G2,pub.P).Int64()
		if gcd == 1 {
			break
		}
		pub.G2.Sub(pub.G2, new(big.Int).SetInt64(1))
	}
	priv.G2 = pub.G2

	// 随机选择私钥 X
	priv.X =new(big.Int)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	priv.X.Rand(rnd, pub.P)

	// 计算公钥 H
	pub.H = new(big.Int)
	pub.H.Exp(pub.G2, priv.X, pub.P)
	priv.H =  pub.H

	return
}

func Encrypt(pub PublicKey, M []byte) (C CypherText){
	// ElGamal 加密 []byte 类型数据 M，输出密文 C
	// 构造C1,C2
	C1 := make([]byte, 0, len(M))
	C2 := make([]byte, 0, len(M))
	// 对明文 M 切片并进行分片处理，每片长 28 bytes（224 bits）
	n := ( len(M) + 27 ) / 28
	for i := 0; i < n; i ++{

		// 明文切片
		var m_bytes []byte
		if i == n-1 {
			m_bytes = M[i * 28 :]
		} else {
			m_bytes = M[i * 28 :(i + 1) * 28 ]
		}

		limit := new(big.Int).Sub(pub.P,new(big.Int).SetInt64(4))

		// 生成随机数 k
		rnd_key := 0
		k := new(big.Int)
		for {
			rnd := rand.New(rand.NewSource(time.Now().UnixNano()+int64(i)*8+int64(rnd_key)))
			k.Rand(rnd, limit)
			k.Add(k,new(big.Int).SetInt64(2))
			gcd := (int)(new(big.Int).GCD(nil,nil,k,pub.P).Int64())
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
		copy(c1_bytes[(32-len(c1.Bytes())):],c1.Bytes())
		copy(c2_bytes[(32-len(c2.Bytes())):],c2.Bytes())

		// 填入 C1,C2
		C1 = append(C1, c1_bytes[:]...)
		C2 = append(C2, c2_bytes[:]...)
	}

	// 合并 C1,C2
	C = CypherText{C1,C2}
	return
}

func Decrypt(priv PrivateKey, C CypherText) (M []byte){
	// ElGamal 解密 []byte 类型数据 C，输出明文 M
	// 构造C1,C2,M
	M = make([]byte, 0, len(C.C1))

	n := ( len(C.C1) + 31 ) / 32
	for i := 0; i < n; i ++{
		// 密文切片
		var c1_bytes, c2_bytes []byte
		if i == n-1 {
			c1_bytes = C.C1[i * 32 :]
			c2_bytes = C.C2[i * 32 :]
		} else {
			c1_bytes = C.C1[i * 32 :(i + 1) * 32 ]
			c2_bytes = C.C2[i * 32 :(i + 1) * 32 ]
		}

		// 解密
		c1 := new(big.Int).SetBytes(c1_bytes)
		c2 := new(big.Int).SetBytes(c2_bytes)
		s := new(big.Int).Exp(c1, priv.X, priv.P)
		s.ModInverse(s, priv.P)
		s.Mul(s, c2)
		s.Mod(s, priv.P)

		m := s.Bytes()
		M = append(M, m...)
	}
	return
}

func Sign(priv PrivateKey, m []byte) (sig Signature){
	// ElGamal 签名算法，将消息 m 签名为 Signature 结构体
	// 构造与p-1互质的随机数
	rnd_key := time.Now().UnixNano()
	limit := new(big.Int).Sub(priv.P,new(big.Int).SetInt64(4))
	P_1 := new(big.Int).Sub(priv.P,new(big.Int).SetInt64(1))
	K := new(big.Int)
	for {
		rnd := rand.New(rand.NewSource(rnd_key))
		k := new(big.Int).Rand(rnd, limit)
		k.Add(k,new(big.Int).SetInt64(2))
		gcd := (int)(new(big.Int).GCD(nil,nil,k,P_1).Int64())
		if gcd == 1 {
			K.Set(k)
			break
		}
		rnd_key++
	}

	// 计算消息 m 的哈希
	hash_m := sha256.Sum256(m)
	hash_m_fix := hash_m[:]
	m_0 := new(big.Int).SetBytes(hash_m_fix)

	// 签名
	m_ := new(big.Int).Mod(m_0, priv.P)
	r := new(big.Int).Exp(priv.G2, K, priv.P)
	s := new(big.Int).ModInverse(K, P_1)
	s1 := new(big.Int).Mul(r, priv.X)
	s1.Sub(m_, s1)
	s.Mul(s, s1)
	s.Mod(s, P_1)

	M := m
	M_hash := m_0.Bytes()
	R := r.Bytes()
	S := s.Bytes()
	sig = Signature{M, M_hash,R,S}
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
