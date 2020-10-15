package crypto

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"time"
)

type Commitment struct {
	commitment, r []byte
}

type FormatProof struct {
	C, Z1, Z2 []byte
}

type BalanceProof struct {
	C, R_v, R_r, S_v, S_r, S_or []byte
}

type keys interface {
	Commit(v *big.Int, rnd []byte) Commitment
	CommitByUint64(v uint64, rnd []byte) Commitment
	CommitByBytes(b []byte, rnd []byte) Commitment
	VerifyCommitment(commit Commitment, rnd []byte) uint64
}

func (pub PublicKey) Commit(v *big.Int, rnd []byte) Commitment{
	// 通过 big int 生成承诺
	v_ := new(big.Int).Mod(v, pub.P)

	// 承诺随机数
	r_ := new(big.Int).SetBytes(rnd)

	// 生成承诺
	v_g1 := new(big.Int).Exp(pub.G1, v_, pub.P)
	r_h := new(big.Int).Exp(pub.H, r_, pub.P)
	commitment_bigInt := new(big.Int).Mul(v_g1, r_h)
	commitment_bigInt.Mod(commitment_bigInt, pub.P)

	// 规范化
	commitment := commitment_bigInt.Bytes()
	r := r_.Bytes()

	// 合并成结构体
	return Commitment{commitment, r}
}

func (priv PrivateKey) Commit(v *big.Int, rnd []byte) Commitment{
	// 通过 big int 生成承诺
	v_ := new(big.Int).Mod(v, priv.P)

	// 生成承诺随机数
	r_ := new(big.Int).SetBytes(rnd[:])

	// 生成承诺
	v_g1 := new(big.Int).Exp(priv.G1, v_, priv.P)
	r_h := new(big.Int).Exp(priv.H, r_, priv.P)
	commitment_bigInt := new(big.Int).Mul(v_g1, r_h)
	commitment_bigInt.Mod(commitment_bigInt, priv.P)

	// 规范化
	commitment := commitment_bigInt.Bytes()
	r := r_.Bytes()

	// 合并成结构体
	return Commitment{commitment, r}
}

func (pub PublicKey) CommitByUint64(v uint64, rnd []byte) Commitment{
	v_ := new(big.Int).SetUint64(v)
	return pub.Commit(v_, rnd)
}

func (priv PrivateKey) CommitByUint64(v uint64, rnd []byte) Commitment{
	v_ := new(big.Int).SetUint64(v)
	return priv.Commit(v_, rnd)
}

func (pub PublicKey) CommitByBytes(b []byte, rnd []byte) Commitment{
	v_ := new(big.Int).SetBytes(b)
	return pub.Commit(v_, rnd)
}

func (priv PrivateKey) CommitByBytes(b []byte, rnd []byte) Commitment{
	v_ := new(big.Int).SetBytes(b)
	return priv.Commit(v_, rnd)
}

func (pub PublicKey) VerifyCommitment(commit Commitment) uint64{
	R := new(big.Int).SetBytes(commit.r[:])
	Commit := new(big.Int).SetBytes(commit.commitment[:])
	for i := 1;true;i++{
		I := new(big.Int).SetInt64(int64(i))
		v_g1 := new(big.Int).Exp(pub.G1, I, pub.P)
		r_h := new(big.Int).Exp(pub.H, R, pub.P)
		commitment_bigInt := new(big.Int).Mul(v_g1, r_h)
		commitment_bigInt.Mod(commitment_bigInt, pub.P)
		if commitment_bigInt.Cmp(Commit) == 0{
			return uint64(i)
		}
		if i >= 262144 {
			fmt.Printf("该承诺并非价值承诺或承诺价值大于262144")
			return 0
		}
	}
	return 0
}

func (priv PrivateKey) VerifyCommitment(commit Commitment) uint64{
	R := new(big.Int).SetBytes(commit.r[:])
	Commit := new(big.Int).SetBytes(commit.commitment[:])
	for i := 1;true;i++{
		I := new(big.Int).SetInt64(int64(i))
		v_g1 := new(big.Int).Exp(priv.G1, I, priv.P)
		r_h := new(big.Int).Exp(priv.H, R, priv.P)
		commitment_bigInt := new(big.Int).Mul(v_g1, r_h)
		commitment_bigInt.Mod(commitment_bigInt, priv.P)
		if commitment_bigInt.Cmp(Commit) == 0{
			return uint64(i)
		}
		if i >= 262144 {
			fmt.Printf("该承诺并非价值承诺或承诺价值大于262144")
			return 0
		}
	}
	return 0
}

func EncryptValue(pub PublicKey, M uint64) (C CypherText, commit Commitment, err error){
	// ElGamal 加密 uint64 类型数据 M，输出密文 C, 承诺 commit
	if M > 262144 {
		err = errors.New("加密价值过大")
		return
	}
	// 生成随机数 k
	limit := new(big.Int).Sub(pub.P,new(big.Int).SetInt64(4))
	rnd_key := 0
	k := new(big.Int)
	for {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()+int64(rnd_key)))
		k.Rand(rnd, limit)
		k.Add(k,new(big.Int).SetInt64(2))
		gcd := (int)(new(big.Int).GCD(nil,nil,k,pub.P).Int64())
		if gcd == 1 {
			break
		}
		rnd_key++
	}

	// 加密
	m := new(big.Int).SetUint64(M)
	c1_ := new(big.Int).Exp(pub.G2, k, pub.P)
	c1 := c1_.Bytes()
	//s := new(big.Int).Exp(pub.H, k, pub.P)
	//c2 := s.Mul(s, m)
	//c2.Mod(c2, pub.P)
	rnd := k.Bytes()
	commit= pub.Commit(m, rnd)


	C = CypherText{c1,commit.commitment}
	return
}

func DecryptValue(priv PrivateKey, C CypherText) (v uint64){
	// ElGamal 解密 C 输出 类型数据 M，输出密文 C
	// 解密
	c1 := new(big.Int).SetBytes(C.C1)
	c2 := new(big.Int).SetBytes(C.C2)
	c1x := new(big.Int).Exp(c1, priv.X, priv.P)
	c1x.ModInverse(c1x, priv.P)
	gv := new(big.Int).Mul(c2,c1x)
	gv.Mod(gv,priv.P)
	for i:=1;true;i++{
		gi := new(big.Int).Exp(priv.G1,new(big.Int).SetInt64(int64(i)),priv.P)
		if gv.Cmp(gi) == 0{
			return uint64(i)
		}
		if i >= 262144 {
			fmt.Printf("该承诺并非价值承诺或承诺价值大于262144")
			return 0
		}
	}
	return
}

func GenerateFormatProof(pub PublicKey, v uint64, r []byte) (fp FormatProof){
	// 生成格式正确证明
	P_1 := new(big.Int).Sub(pub.P, new(big.Int).SetUint64(1))

	// 生成 a,b
	rnd_a := rand.New(rand.NewSource(time.Now().UnixNano()))
	rnd_b := rand.New(rand.NewSource(rnd_a.Int63()))
	limit := new(big.Int).Sub(pub.P,new(big.Int).SetInt64(4))
	a := new(big.Int).Rand(rnd_a,limit)
	a.Add(a, new(big.Int).SetUint64(2))
	b := new(big.Int).Rand(rnd_b,limit)
	b.Add(b, new(big.Int).SetUint64(2))

	// 生成零知识证明

	//t1_p
	g1a := new(big.Int).Exp(pub.G1, a, pub.P)
	hb := new(big.Int).Exp(pub.H, b, pub.P)
	t1_p := new(big.Int).Mul(g1a, hb)
	t1_p.Mod(t1_p,pub.P)

	//t2_p
	t2_p := new(big.Int).Exp(pub.G2, b, pub.P)

	//c
	hash_t1p := t1_p.Bytes()
	hash_t2p := t2_p.Bytes()
	c_ := sha256.Sum256(append(hash_t1p, hash_t2p...))
	c := new(big.Int).SetBytes(c_[:])
	c.Mod(c,pub.P)

	//z1
	vc := new(big.Int).Mul(new(big.Int).SetUint64(v), c)
	vc.Mod(vc, P_1)
	vc.Sub(P_1, vc)
	z1 := new(big.Int).Add(a,vc)
	z1.Mod(z1, P_1)

	//z2
	r1c := new(big.Int).Mul(new(big.Int).SetBytes(r), c)
	r1c.Mod(r1c, P_1)
	r1c.Sub(P_1, r1c)
	z2 := new(big.Int).Add(b,r1c)
	z2.Mod(z2, P_1)

	// 转为 bytes 存储
	fp.C = c.Bytes()
	fp.Z1 = z1.Bytes()
	fp.Z2 =z2.Bytes()
	return
}

func VerifyFormatProof(C CypherText, pub PublicKey, fp FormatProof) bool{
	// 格式正确证明验证
	// 初始化文本
	c1 := new(big.Int).SetBytes(C.C1)
	c2 := new(big.Int).SetBytes(C.C2)
	c := new(big.Int).SetBytes(fp.C)
	z1 := new(big.Int).SetBytes(fp.Z1)
	z2 := new(big.Int).SetBytes(fp.Z2)

	// 证明
	//t1_v
	c1c := new(big.Int).Exp(c2, c, pub.P)
	g1z1 := new(big.Int).Exp(pub.G1, z1, pub.P)
	hz2 := new(big.Int).Exp(pub.H, z2, pub.P)
	c1c.Mul(c1c, g1z1)
	c1c.Mod(c1c, pub.P)
	t1_v := new(big.Int).Mul(c1c, hz2)
	t1_v.Mod(t1_v, pub.P)

	//t2_v
	c2c := new(big.Int).Exp(c1, c, pub.P)
	g2z2 := new(big.Int).Exp(pub.G2, z2, pub.P)
	t2_v := new(big.Int).Mul(c2c, g2z2)
	t2_v.Mod(t2_v, pub.P)

	//c_v
	hash_t1v := t1_v.Bytes()
	hash_t2v := t2_v.Bytes()
	c_ := sha256.Sum256(append(hash_t1v, hash_t2v...))
	c_v := new(big.Int).SetBytes(c_[:])
	c_v.Mod(c_v, pub.P)

	return c.Cmp(c_v) == 0
}

func GenerateBalanceProof(pub PublicKey, v_r, v_s, v_o uint64, r_r, r_s, r_o []byte) (bp BalanceProof) {
	// 生成会计平衡证明
	P_1 := new(big.Int).Sub(pub.P, new(big.Int).SetUint64(1))

	// 生成 a,b,d,e,f
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	limit := new(big.Int).Sub(pub.P,new(big.Int).SetInt64(4))
	limit_5 := new(big.Int).Exp(limit,big.NewInt(5),nil)
	mix := new(big.Int).Rand(rnd, limit_5)
	a := new(big.Int).Mod(mix, limit)
	a.Add(a, new(big.Int).SetUint64(2))
	mix.Div(mix, limit)
	b := new(big.Int).Mod(mix, limit)
	b.Add(b, new(big.Int).SetUint64(2))
	mix.Div(mix, limit)
	d := new(big.Int).Mod(mix, limit)
	d.Add(d, new(big.Int).SetUint64(2))
	mix.Div(mix, limit)
	e := new(big.Int).Mod(mix, limit)
	e.Add(e, new(big.Int).SetUint64(2))
	mix.Div(mix, limit)
	f := new(big.Int).Mod(mix, limit)
	f.Add(f, new(big.Int).SetUint64(2))

	// t1_p
	g1a := new(big.Int).Exp(pub.G1, a, pub.P)
	hb := new(big.Int).Exp(pub.H, b, pub.P)
	t1_p := new(big.Int).Mul(g1a, hb)
	t1_p.Mod(t1_p, pub.P)

	// t2_p
	g1d := new(big.Int).Exp(pub.G1, d, pub.P)
	he := new(big.Int).Exp(pub.H, e, pub.P)
	t2_p := new(big.Int).Mul(g1d, he)
	t2_p.Mod(t2_p, pub.P)

	// t3_p
	ad := new(big.Int).Add(a, d)
	ad.Mod(ad, P_1)
	g1ad := new(big.Int).Exp(pub.G1, ad, pub.P)
	hf := new(big.Int).Exp(pub.H, f, pub.P)
	t3_p := new(big.Int).Mul(g1ad, hf)
	t3_p.Mod(t3_p, pub.P)

	// c
	hash_t1p := t1_p.Bytes()
	hash_t2p := t2_p.Bytes()
	hash_t3p := t3_p.Bytes()
	c_ := sha256.Sum256(append(append(hash_t1p, hash_t2p...), hash_t3p...))
	c := new(big.Int).SetBytes(c_[:])
	c.Mod(c,pub.P)

	// R_v
	cv_r := new(big.Int).Mul(c, new(big.Int).SetUint64(v_r))
	cv_r.Mod(cv_r, P_1)
	cv_r.Sub(P_1, cv_r)
	R_v := new(big.Int).Add(a, cv_r)
	R_v.Mod(R_v, P_1)

	// R_r
	cr_r := new(big.Int).Mul(c, new(big.Int).SetBytes(r_r))
	cr_r.Mod(cr_r, P_1)
	cr_r.Sub(P_1, cr_r)
	R_r := new(big.Int).Add(b, cr_r)
	R_r.Mod(R_r, P_1)

	// S_v
	cv_s := new(big.Int).Mul(c, new(big.Int).SetUint64(v_s))
	cv_s.Mod(cv_s, P_1)
	cv_s.Sub(P_1, cv_s)
	S_v := new(big.Int).Add(d, cv_s)
	S_v.Mod(S_v, P_1)

	// S_r
	cr_s := new(big.Int).Mul(c, new(big.Int).SetBytes(r_s))
	cr_s.Mod(cr_s, P_1)
	cr_s.Sub(P_1, cr_s)
	S_r := new(big.Int).Add(e, cr_s)
	S_r.Mod(S_r, P_1)

	// S_or
	cr_o := new(big.Int).Mul(c, new(big.Int).SetBytes(r_o))
	cr_o.Mod(cr_o, P_1)
	cr_o.Sub(P_1, cr_o)
	S_or := new(big.Int).Add(f, cr_o)
	S_or.Mod(S_or, P_1)

	// 组装
	bp.C = c.Bytes()
	bp.R_v = R_v.Bytes()
	bp.R_r = R_r.Bytes()
	bp.S_v = S_v.Bytes()
	bp.S_r = S_r.Bytes()
	bp.S_or = S_or.Bytes()
	return
}

func VerifyBalanceProof(CM_r, CM_s, CM_o []byte, pub PublicKey, bp BalanceProof) bool{
	// 验证会计平衡证明
	// 初始化
	CM_R := new(big.Int).SetBytes(CM_r)
	CM_S := new(big.Int).SetBytes(CM_s)
	CM_O := new(big.Int).SetBytes(CM_o)
	c := new(big.Int).SetBytes(bp.C)
	R_v := new(big.Int).SetBytes(bp.R_v)
	R_r := new(big.Int).SetBytes(bp.R_r)
	S_v := new(big.Int).SetBytes(bp.S_v)
	S_r := new(big.Int).SetBytes(bp.S_r)
	S_or := new(big.Int).SetBytes(bp.S_or)

	// t1_v
	g1rv := new(big.Int).Exp(pub.G1, R_v, pub.P)
	hrr := new(big.Int).Exp(pub.H, R_r, pub.P)
	cmrc := new(big.Int).Exp(CM_R, c, pub.P)
	t1_v := new(big.Int).Mul(g1rv, hrr)
	t1_v.Mod(t1_v, pub.P)
	t1_v.Mul(t1_v, cmrc)
	t1_v.Mod(t1_v, pub.P)

	// t2_v
	g1sv := new(big.Int).Exp(pub.G1, S_v, pub.P)
	hsr := new(big.Int).Exp(pub.H, S_r, pub.P)
	cmsc := new(big.Int).Exp(CM_S, c, pub.P)
	t2_v := new(big.Int).Mul(g1sv, hsr)
	t2_v.Mod(t2_v, pub.P)
	t2_v.Mul(t2_v, cmsc)
	t2_v.Mod(t2_v, pub.P)

	// t3_v
	rvsv := new(big.Int).Add(R_v, S_v)
	g1rvsv := new(big.Int).Exp(pub.G1, rvsv, pub.P)
	hsor := new(big.Int).Exp(pub.H, S_or, pub.P)
	cmoc := new(big.Int).Exp(CM_O, c, pub.P)
	t3_v := new(big.Int).Mul(g1rvsv, hsor)
	t3_v.Mod(t3_v, pub.P)
	t3_v.Mul(t3_v, cmoc)
	t3_v.Mod(t3_v, pub.P)

	// c_v
	hash_t1v := t1_v.Bytes()
	hash_t2v := t2_v.Bytes()
	hash_t3v := t3_v.Bytes()
	c_ := sha256.Sum256(append(append(hash_t1v, hash_t2v...), hash_t3v...))
	c_v := new(big.Int).SetBytes(c_[:])
	c_v.Mod(c_v, pub.P)

	return c.Cmp(c_v) == 0
}