package main

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

type BalanceProof_old struct {
	C, R_v, R_r, S_v, S_r, S_or []byte
}

type LinearEquationProof struct {
	s [][]byte
	t []byte
}

type EqualityProof struct {
	LinearEquationProof
}

type BalanceProof struct {
	LinearEquationProof
}

type keys interface {
	Commit(v *big.Int, rnd []byte) Commitment
	CommitByUint64(v uint64, rnd []byte) Commitment
	CommitByBytes(b []byte, rnd []byte) Commitment
	VerifyCommitment(commit Commitment, rnd []byte) uint64
}

func RandomBigInt(n int, pub PublicKey) []*big.Int {
	num := make([]*big.Int, n)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	limit := new(big.Int).Sub(pub.P, new(big.Int).SetInt64(4))
	limit_5 := new(big.Int).Exp(limit, big.NewInt(5), nil)
	mix := new(big.Int).Rand(rnd, limit_5)
	for i := 0; i < n; i++ {
		buf := new(big.Int).Mod(mix, limit)
		num[i] = big.NewInt(0)
		num[i].Add(buf, new(big.Int).SetUint64(2))
		if i < n-1 {
			mix.Div(mix, limit)
		}
	}
	return num
}

func (pub PublicKey) Commit(v *big.Int, rnd []byte) Commitment {
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

func (priv PrivateKey) Commit(v *big.Int, rnd []byte) Commitment {
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

func (pub PublicKey) CommitByUint64(v uint64, rnd []byte) Commitment {
	v_ := new(big.Int).SetUint64(v)
	return pub.Commit(v_, rnd)
}

func (priv PrivateKey) CommitByUint64(v uint64, rnd []byte) Commitment {
	v_ := new(big.Int).SetUint64(v)
	return priv.Commit(v_, rnd)
}

func (pub PublicKey) CommitByBytes(b []byte, rnd []byte) Commitment {
	v_ := new(big.Int).SetBytes(b)
	return pub.Commit(v_, rnd)
}

func (priv PrivateKey) CommitByBytes(b []byte, rnd []byte) Commitment {
	v_ := new(big.Int).SetBytes(b)
	return priv.Commit(v_, rnd)
}

func (pub PublicKey) VerifyCommitment(commit Commitment) uint64 {
	R := new(big.Int).SetBytes(commit.r[:])
	Commit := new(big.Int).SetBytes(commit.commitment[:])
	for i := 1; true; i++ {
		I := new(big.Int).SetInt64(int64(i))
		v_g1 := new(big.Int).Exp(pub.G1, I, pub.P)
		r_h := new(big.Int).Exp(pub.H, R, pub.P)
		commitment_bigInt := new(big.Int).Mul(v_g1, r_h)
		commitment_bigInt.Mod(commitment_bigInt, pub.P)
		if commitment_bigInt.Cmp(Commit) == 0 {
			return uint64(i)
		}
		if i >= 262144 {
			fmt.Printf("该承诺并非价值承诺或承诺价值大于262144")
			return 0
		}
	}
	return 0
}

func (priv PrivateKey) VerifyCommitment(commit Commitment) uint64 {
	R := new(big.Int).SetBytes(commit.r[:])
	Commit := new(big.Int).SetBytes(commit.commitment[:])
	for i := 1; true; i++ {
		I := new(big.Int).SetInt64(int64(i))
		v_g1 := new(big.Int).Exp(priv.G1, I, priv.P)
		r_h := new(big.Int).Exp(priv.H, R, priv.P)
		commitment_bigInt := new(big.Int).Mul(v_g1, r_h)
		commitment_bigInt.Mod(commitment_bigInt, priv.P)
		if commitment_bigInt.Cmp(Commit) == 0 {
			return uint64(i)
		}
		if i >= 262144 {
			fmt.Printf("该承诺并非价值承诺或承诺价值大于262144")
			return 0
		}
	}
	return 0
}

func EncryptValue(pub PublicKey, M uint64) (C CypherText, commit Commitment, err error) {
	// ElGamal 加密 uint64 类型数据 M，输出密文 C, 承诺 commit
	if M > 262144 {
		err = errors.New("加密价值过大")
		return
	}
	// 生成随机数 k
	limit := new(big.Int).Sub(pub.P, new(big.Int).SetInt64(4))
	rnd_key := 0
	k := new(big.Int)
	for {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(rnd_key)))
		k.Rand(rnd, limit)
		k.Add(k, new(big.Int).SetInt64(2))
		gcd := (int)(new(big.Int).GCD(nil, nil, k, pub.P).Int64())
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
	commit = pub.Commit(m, rnd)

	C = CypherText{c1, commit.commitment}
	return
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

func EncryptAddress(pub PublicKey, addr []byte) (C CypherText, commit Commitment, err error) {
	// 加密地址
	limit := new(big.Int).Sub(pub.P, new(big.Int).SetInt64(4))
	rnd_key := 0
	k := new(big.Int)
	for {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(rnd_key)))
		k.Rand(rnd, limit)
		k.Add(k, new(big.Int).SetInt64(2))
		gcd := (int)(new(big.Int).GCD(nil, nil, k, pub.P).Int64())
		if gcd == 1 {
			break
		}
		rnd_key++
	}

	// 加密
	m := new(big.Int).SetBytes(addr)
	c1_ := new(big.Int).Exp(pub.G2, k, pub.P)
	c1 := c1_.Bytes()
	//s := new(big.Int).Exp(pub.H, k, pub.P)
	//c2 := s.Mul(s, m)
	//c2.Mod(c2, pub.P)
	rnd := k.Bytes()
	commit = pub.Commit(m, rnd)

	C = CypherText{c1, commit.commitment}
	return
}

func DecryptAddress(priv PrivateKey, C CypherText, PkPool [][]byte) []byte {
	// 解密地址
	c1 := new(big.Int).SetBytes(C.C1)
	c2 := new(big.Int).SetBytes(C.C2)
	c1x := new(big.Int).Exp(c1, priv.X, priv.P)
	c1x.ModInverse(c1x, priv.P)
	gv := new(big.Int).Mul(c2, c1x)
	gv.Mod(gv, priv.P)
	for _, pk := range PkPool {
		gi := new(big.Int).Exp(priv.G1, new(big.Int).SetBytes(pk), priv.P)
		if gv.Cmp(gi) == 0 {
			return pk
		}
	}
	return nil
}

func GenerateFormatProof(pub PublicKey, v uint64, r []byte) (fp FormatProof) {
	// 生成格式正确证明
	P_1 := new(big.Int).Sub(pub.P, new(big.Int).SetUint64(1))

	// 生成 a,b
	rnd_a := rand.New(rand.NewSource(time.Now().UnixNano()))
	rnd_b := rand.New(rand.NewSource(rnd_a.Int63()))
	limit := new(big.Int).Sub(pub.P, new(big.Int).SetInt64(4))
	a := new(big.Int).Rand(rnd_a, limit)
	a.Add(a, new(big.Int).SetUint64(2))
	b := new(big.Int).Rand(rnd_b, limit)
	b.Add(b, new(big.Int).SetUint64(2))

	// 生成零知识证明

	//t1_p
	g1a := new(big.Int).Exp(pub.G1, a, pub.P)
	hb := new(big.Int).Exp(pub.H, b, pub.P)
	t1_p := new(big.Int).Mul(g1a, hb)
	t1_p.Mod(t1_p, pub.P)

	//t2_p
	t2_p := new(big.Int).Exp(pub.G2, b, pub.P)

	//c
	hash_t1p := t1_p.Bytes()
	hash_t2p := t2_p.Bytes()
	c_ := sha256.Sum256(append(hash_t1p, hash_t2p...))
	c := new(big.Int).SetBytes(c_[:])
	c.Mod(c, pub.P)

	//z1
	vc := new(big.Int).Mul(new(big.Int).SetUint64(v), c)
	vc.Mod(vc, P_1)
	vc.Sub(P_1, vc)
	z1 := new(big.Int).Add(a, vc)
	z1.Mod(z1, P_1)

	//z2
	r1c := new(big.Int).Mul(new(big.Int).SetBytes(r), c)
	r1c.Mod(r1c, P_1)
	r1c.Sub(P_1, r1c)
	z2 := new(big.Int).Add(b, r1c)
	z2.Mod(z2, P_1)

	// 转为 bytes 存储
	fp.C = c.Bytes()
	fp.Z1 = z1.Bytes()
	fp.Z2 = z2.Bytes()
	return
}

func GenerateAddressFormatProof(pub PublicKey, addr []byte, r []byte) (fp FormatProof) {
	// 生成格式正确证明
	P_1 := new(big.Int).Sub(pub.P, new(big.Int).SetUint64(1))

	// 生成 a,b
	rnd_a := rand.New(rand.NewSource(time.Now().UnixNano()))
	rnd_b := rand.New(rand.NewSource(rnd_a.Int63()))
	limit := new(big.Int).Sub(pub.P, new(big.Int).SetInt64(4))
	a := new(big.Int).Rand(rnd_a, limit)
	a.Add(a, new(big.Int).SetUint64(2))
	b := new(big.Int).Rand(rnd_b, limit)
	b.Add(b, new(big.Int).SetUint64(2))

	// 生成零知识证明

	//t1_p
	g1a := new(big.Int).Exp(pub.G1, a, pub.P)
	hb := new(big.Int).Exp(pub.H, b, pub.P)
	t1_p := new(big.Int).Mul(g1a, hb)
	t1_p.Mod(t1_p, pub.P)

	//t2_p
	t2_p := new(big.Int).Exp(pub.G2, b, pub.P)

	//c
	hash_t1p := t1_p.Bytes()
	hash_t2p := t2_p.Bytes()
	c_ := sha256.Sum256(append(hash_t1p, hash_t2p...))
	c := new(big.Int).SetBytes(c_[:])
	c.Mod(c, pub.P)

	//z1
	vc := new(big.Int).Mul(new(big.Int).SetBytes(addr), c)
	vc.Mod(vc, P_1)
	vc.Sub(P_1, vc)
	z1 := new(big.Int).Add(a, vc)
	z1.Mod(z1, P_1)

	//z2
	r1c := new(big.Int).Mul(new(big.Int).SetBytes(r), c)
	r1c.Mod(r1c, P_1)
	r1c.Sub(P_1, r1c)
	z2 := new(big.Int).Add(b, r1c)
	z2.Mod(z2, P_1)

	// 转为 bytes 存储
	fp.C = c.Bytes()
	fp.Z1 = z1.Bytes()
	fp.Z2 = z2.Bytes()
	return
}

func VerifyFormatProof(C CypherText, pub PublicKey, fp FormatProof) bool {
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

func GenerateLinearEquationProof(y []byte, b int, a []int, x [][]byte, g [][]byte, pub PublicKey) (lp LinearEquationProof) {
	// 生成线性证明
	// 输入合法判断
	if len(x) != len(g) || len(g) != len(a) {
		fmt.Printf("\n%d,%d,%d,%d", len(x), len(a), len(g))
		lp = LinearEquationProof{
			s: nil,
			t: nil,
		}
		fmt.Printf("\n线性证明生成失败，输入数据不合法！\n")
		return
	}
	// 计算生成随机数列 v
	v := make([]*big.Int, len(a))
	ss := make([]bool, len(a))
	P_1 := new(big.Int).Sub(pub.P, big.NewInt(1))
	ssnum := 0
	for i, ai := range a {
		if ai == 0 {
			ss[i] = false
		} else {
			ss[i] = true
			ssnum += 1
		}
	}
	var rbi []*big.Int
	if ssnum == 0 {
		rbi = RandomBigInt(len(a), pub)
	} else {
		rbi = RandomBigInt(len(a)-1, pub)
	}
	line := 0
	last := big.NewInt(0)
	for i, ai := range a {
		if ai == 0 {
			v[i] = rbi[line]
			line++
		} else {
			if ssnum == 1 {
				Ai := big.NewInt(int64(ai))
				Ai.ModInverse(Ai, P_1)
				v[i] = new(big.Int).Mul(last, Ai)
				v[i].Mod(v[i], P_1)
				ssnum--
			} else {
				v[i] = rbi[line]
				line++
				ssnum--
				buf := new(big.Int).Mul(big.NewInt(int64(a[i])), v[i])
				last.Sub(last, buf)
				last.Mod(last, P_1)
			}
		}
	}
	// 验证
	vrf := big.NewInt(0)
	for i, vi := range v {
		buf := new(big.Int).Mul(vi, big.NewInt(int64(a[i])))
		vrf.Add(vrf, buf)
		vrf.Mod(vrf, P_1)
	}
	// 计算t
	t := big.NewInt(1)
	var c_mash []byte
	for i, gi := range g {
		Gi := new(big.Int).SetBytes(gi)
		Gi.Exp(Gi, v[i], pub.P)
		t.Mul(t, Gi)
		t.Mod(t, pub.P)
		c_mash = append(c_mash, gi...)
	}
	lp.t = t.Bytes()
	// 计算 c
	c_mash = append(c_mash, y...)
	c_mash = append(c_mash, lp.t...)
	c_32bit := sha256.Sum256(c_mash)
	c := c_32bit[:]
	c_bi := new(big.Int).SetBytes(c)
	c_bi.Mod(c_bi, pub.P)
	// 计算 s
	for i, vi := range v {
		mash := new(big.Int).SetBytes(x[i])
		mash.Mul(c_bi, mash)
		mash.Sub(vi, mash)
		mash.Mod(mash, P_1)
		lp.s = append(lp.s, mash.Bytes())
	}
	return
}

func VerifyLinearEquationProof(lp LinearEquationProof, y []byte, b int, a []int, g [][]byte, pub PublicKey) bool {
	// 验证线性证明
	if len(lp.s) != len(g) || len(g) != len(a) {
		fmt.Printf("\n%d,%d,%d,%d", len(lp.s), len(a), len(g))
		fmt.Printf("\n线性证明验证失败，输入数据不合法！\n")
		return false
	}
	P_1 := new(big.Int).Sub(pub.P, big.NewInt(1))
	// 计算 c
	var c_mash []byte
	for _, gi := range g {
		c_mash = append(c_mash, gi...)
	}
	c_mash = append(c_mash, y...)
	c_mash = append(c_mash, lp.t...)
	c_32bit := sha256.Sum256(c_mash)
	c := c_32bit[:]
	c_bi := new(big.Int).SetBytes(c)
	c_bi.Mod(c_bi, pub.P)
	// 验证 t 值
	t_verify := new(big.Int).SetBytes(y)
	t_verify.Exp(t_verify, c_bi, pub.P)
	for i, gi := range g {
		buf := new(big.Int).SetBytes(gi)
		buf.Exp(buf, new(big.Int).SetBytes(lp.s[i]), pub.P)
		t_verify.Mul(t_verify, buf)
		t_verify.Mod(t_verify, pub.P)
	}
	if t_verify.Cmp(new(big.Int).SetBytes(lp.t)) != 0 {
		fmt.Printf("\nt验证失败！\n")
		return false
	}
	// 验证 s 值
	cb := new(big.Int).SetUint64(uint64(b))
	cb.Mul(c_bi, cb)
	cb.Neg(cb)
	mix := big.NewInt(0)
	for i, ai := range a {
		Ai := new(big.Int).SetInt64(int64(ai))
		Si := new(big.Int).SetBytes(lp.s[i])
		aisi := new(big.Int).Mul(Ai, Si)
		aisi.Mod(aisi, P_1)
		mix.Add(mix, aisi)
		mix.Mod(mix, P_1)
	}
	if mix.Cmp(cb) != 0 {
		fmt.Printf("\ns验证失败！\n")
		return false
	}
	return true
}

func GenerateEqualityProof(pub1, pub2 PublicKey, C1, C2 Commitment, v uint) (ep EqualityProof) {
	if pub1.P.Cmp(pub2.P) != 0 {
		fmt.Printf("\n零知识证明的两个密码公钥必须在同一个循环群内！\n")
		ep.s = nil
		ep.t = nil
		return
	}
	V := big.NewInt(int64(v))
	Y := new(big.Int).Mul(new(big.Int).SetBytes(C1.commitment), new(big.Int).SetBytes(C2.commitment))
	Y.Mod(Y, pub1.P)
	g := [][]byte{pub1.G1.Bytes(), pub2.G1.Bytes(), pub1.H.Bytes(), pub2.H.Bytes()}
	x := [][]byte{V.Bytes(), V.Bytes(), C1.r, C2.r}
	a := []int{1, -1, 0, 0}
	b := 0
	y := Y.Bytes()
	pub := pub1
	ep.LinearEquationProof = GenerateLinearEquationProof(y, b, a, x, g, pub)
	return ep
}

func GenerateAddressEqualityProof(pub1, pub2 PublicKey, C1, C2 Commitment, addr []byte) (ep EqualityProof) {
	if pub1.P.Cmp(pub2.P) != 0 {
		fmt.Printf("\n零知识证明的两个密码公钥必须在同一个循环群内！\n")
		ep.s = nil
		ep.t = nil
		return
	}
	V := new(big.Int).SetBytes(addr)
	Y := new(big.Int).Mul(new(big.Int).SetBytes(C1.commitment), new(big.Int).SetBytes(C2.commitment))
	Y.Mod(Y, pub1.P)
	g := [][]byte{pub1.G1.Bytes(), pub2.G1.Bytes(), pub1.H.Bytes(), pub2.H.Bytes()}
	x := [][]byte{V.Bytes(), V.Bytes(), C1.r, C2.r}
	a := []int{1, -1, 0, 0}
	b := 0
	y := Y.Bytes()
	pub := pub1
	ep.LinearEquationProof = GenerateLinearEquationProof(y, b, a, x, g, pub)
	return ep
}

func VerifyEqualityProof(pub1, pub2 PublicKey, C1, C2 CypherText, ep EqualityProof) bool {
	if pub1.P.Cmp(pub2.P) != 0 {
		fmt.Printf("\n零知识证明的两个密码公钥必须在同一个循环群内！\n")
		return false
	}
	Y := new(big.Int).Mul(new(big.Int).SetBytes(C1.C2), new(big.Int).SetBytes(C2.C2))
	Y.Mod(Y, pub1.P)
	y := Y.Bytes()
	g := [][]byte{pub1.G1.Bytes(), pub2.G1.Bytes(), pub1.H.Bytes(), pub2.H.Bytes()}
	a := []int{1, -1, 0, 0}
	b := 0
	return VerifyLinearEquationProof(ep.LinearEquationProof, y, b, a, g, pub1)
}

func GenerateBalanceProof(pub PublicKey, C_o, C_s []Commitment, v_o, v_s []uint) (bp BalanceProof) {
	if len(C_o) != len(v_o) || len(C_s) != len(v_s) {
		fmt.Printf("\n输入错误！长度不匹配！\n")
		bp.t = nil
		bp.s = nil
		return
	}
	P_1 := new(big.Int).Sub(pub.P, big.NewInt(1))
	vr := 0
	rr := big.NewInt(0)
	cr := big.NewInt(1)
	for i, vo := range v_o {
		vr += int(vo)
		rr.Add(rr, new(big.Int).SetBytes(C_o[i].r))
		rr.Mod(rr, P_1)
		cr.Mul(cr, new(big.Int).SetBytes(C_o[i].commitment))
		cr.Mod(cr, pub.P)
	}
	csc := big.NewInt(1)
	for i, vs := range v_s {
		vr -= int(vs)
		rr.Sub(rr, new(big.Int).SetBytes(C_s[i].r))
		rr.Mod(rr, P_1)
		csc.Mul(csc, new(big.Int).SetBytes(C_s[i].commitment))
		csc.Mod(csc, pub.P)
	}
	x := [][]byte{big.NewInt(int64(vr)).Bytes(), rr.Bytes()}
	csc.ModInverse(csc, pub.P)
	Y := new(big.Int).Mul(cr, csc)
	Y.Mod(Y, pub.P)
	y := Y.Bytes()
	a := []int{1, 0}
	b := 0
	g := [][]byte{pub.G1.Bytes(), pub.H.Bytes()}
	bp.LinearEquationProof = GenerateLinearEquationProof(y, b, a, x, g, pub)
	return
}

func VerifyBalanceProof(pub PublicKey, C_o, C_s []CypherText, bp BalanceProof) bool {
	cr := big.NewInt(1)
	for _, co := range C_o {
		cr.Mul(cr, new(big.Int).SetBytes(co.C2))
		cr.Mod(cr, pub.P)
	}
	csc := big.NewInt(1)
	for _, cs := range C_s {
		csc.Mul(csc, new(big.Int).SetBytes(cs.C2))
		csc.Mod(csc, pub.P)
	}
	csc.ModInverse(csc, pub.P)
	Y := new(big.Int).Mul(cr, csc)
	Y.Mod(Y, pub.P)
	y := Y.Bytes()
	a := []int{1, 0}
	b := 0
	g := [][]byte{pub.G1.Bytes(), pub.H.Bytes()}
	return VerifyLinearEquationProof(bp.LinearEquationProof, y, b, a, g, pub)
}

func GenerateBalanceProof_old(pub PublicKey, v_r, v_s, v_o uint64, r_r, r_s, r_o []byte) (bp BalanceProof_old) {
	// 生成会计平衡证明
	P_1 := new(big.Int).Sub(pub.P, new(big.Int).SetUint64(1))

	// 生成 a,b,d,e,f
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	limit := new(big.Int).Sub(pub.P, new(big.Int).SetInt64(4))
	limit_5 := new(big.Int).Exp(limit, big.NewInt(5), nil)
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
	c.Mod(c, pub.P)

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

func VerifyBalanceProof_old(CM_r, CM_s, CM_o []byte, pub PublicKey, bp BalanceProof_old) bool {
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
