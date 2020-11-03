package zkp

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"
)

type Commitment struct {
	Commitment, R []byte
}

// 加密文本
type CypherText struct {
	C1, C2 []byte
}

// 签名
type Signature struct {
	M, M_hash, R, S []byte
}

func (pub PublicKey) Commit(v *big.Int, rnd []byte) Commitment {
	// 通过 big int 生成承诺
	v_ := new(big.Int).Mod(v, pub.P)

	// 承诺随机数
	r_ := new(big.Int).SetBytes(rnd)

	// 生成承诺
	vG1 := new(big.Int).Exp(pub.G1, v_, pub.P)
	rH := new(big.Int).Exp(pub.H, r_, pub.P)
	commitmentBigint := new(big.Int).Mul(vG1, rH)
	commitmentBigint.Mod(commitmentBigint, pub.P)

	// 规范化
	commitment := commitmentBigint.Bytes()
	r := r_.Bytes()

	// 合并成结构体
	return Commitment{commitment, r}
}

func (pub PublicKey) CommitByUint64(v uint64) Commitment {
	v_ := new(big.Int).SetUint64(v)
	rnd := pub.randBytes()
	return pub.Commit(v_, rnd)
}

func (pub PublicKey) CommitByBytes(b []byte) Commitment {
	v_ := new(big.Int).SetBytes(b)
	rnd := pub.randBytes()
	return pub.Commit(v_, rnd)
}

func (pub PublicKey) VerifyCommitment(commit Commitment) uint64 {
	R := new(big.Int).SetBytes(commit.R[:])
	Commit := new(big.Int).SetBytes(commit.Commitment[:])
	for i := 1; true; i++ {
		I := new(big.Int).SetInt64(int64(i))
		vG1 := new(big.Int).Exp(pub.G1, I, pub.P)
		rH := new(big.Int).Exp(pub.H, R, pub.P)
		commitmentBigint := new(big.Int).Mul(vG1, rH)
		commitmentBigint.Mod(commitmentBigint, pub.P)
		if commitmentBigint.Cmp(Commit) == 0 {
			return uint64(i)
		}
		if i >= 262144 {
			fmt.Printf("该承诺并非价值承诺或承诺价值大于262144")
			return 0
		}
	}
	return 0
}

// randBytes 依赖公钥生成随机数
func (pub PublicKey) randBytes() []byte {
	limit := new(big.Int).Sub(pub.P, new(big.Int).SetInt64(4))
	rndKey := 0
	k := new(big.Int)
	for {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(rndKey)))
		k.Rand(rnd, limit)
		k.Add(k, new(big.Int).SetInt64(2))
		gcd := (int)(new(big.Int).GCD(nil, nil, k, pub.P).Int64())
		if gcd == 1 {
			break
		}
		rndKey++
	}
	rnd := k.Bytes()
	return rnd
}
