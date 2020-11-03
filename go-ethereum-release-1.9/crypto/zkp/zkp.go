package zkp

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"time"
)

type FormatProof struct {
	C, Z1, Z2 []byte
}

type BalanceProof struct {
	C, R_v, R_r, S_v, S_r, S_or []byte
}

type LinearEquationProof struct {
	S [][]byte
	T []byte
}

type EqualityProof struct {
	LinearEquationProof
}

func GenerateFormatProof(pub PublicKey, v uint64, r []byte) (fp FormatProof) {
	// 生成格式正确证明
	P1 := new(big.Int).Sub(pub.P, new(big.Int).SetUint64(1))

	// 生成 a,b
	rndA := rand.New(rand.NewSource(time.Now().UnixNano()))
	rndB := rand.New(rand.NewSource(rndA.Int63()))
	limit := new(big.Int).Sub(pub.P, new(big.Int).SetInt64(4))
	a := new(big.Int).Rand(rndA, limit)
	a.Add(a, new(big.Int).SetUint64(2))
	b := new(big.Int).Rand(rndB, limit)
	b.Add(b, new(big.Int).SetUint64(2))

	// 生成零知识证明

	//t1_p
	g1a := new(big.Int).Exp(pub.G1, a, pub.P)
	hb := new(big.Int).Exp(pub.H, b, pub.P)
	t1P := new(big.Int).Mul(g1a, hb)
	t1P.Mod(t1P, pub.P)

	//t2_p
	t2P := new(big.Int).Exp(pub.G2, b, pub.P)

	//c
	hashT1p := t1P.Bytes()
	hashT2p := t2P.Bytes()
	c_ := sha256.Sum256(append(hashT1p, hashT2p...))
	c := new(big.Int).SetBytes(c_[:])
	c.Mod(c, pub.P)

	//z1
	vc := new(big.Int).Mul(new(big.Int).SetUint64(v), c)
	vc.Mod(vc, P1)
	vc.Sub(P1, vc)
	z1 := new(big.Int).Add(a, vc)
	z1.Mod(z1, P1)

	//z2
	r1c := new(big.Int).Mul(new(big.Int).SetBytes(r), c)
	r1c.Mod(r1c, P1)
	r1c.Sub(P1, r1c)
	z2 := new(big.Int).Add(b, r1c)
	z2.Mod(z2, P1)

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
	t1V := new(big.Int).Mul(c1c, hz2)
	t1V.Mod(t1V, pub.P)

	//t2_v
	c2c := new(big.Int).Exp(c1, c, pub.P)
	g2z2 := new(big.Int).Exp(pub.G2, z2, pub.P)
	t2V := new(big.Int).Mul(c2c, g2z2)
	t2V.Mod(t2V, pub.P)

	//c_v
	hashT1v := t1V.Bytes()
	hashT2v := t2V.Bytes()
	c_ := sha256.Sum256(append(hashT1v, hashT2v...))
	cV := new(big.Int).SetBytes(c_[:])
	cV.Mod(cV, pub.P)

	return c.Cmp(cV) == 0
}

func GenerateBalanceProof(pub PublicKey, vR, vS, vO uint64, rR, rS, rO []byte) (bp BalanceProof) {
	// 生成会计平衡证明
	P1 := new(big.Int).Sub(pub.P, new(big.Int).SetUint64(1))

	// 生成 a,b,d,e,f
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	limit := new(big.Int).Sub(pub.P, new(big.Int).SetInt64(4))
	limit5 := new(big.Int).Exp(limit, big.NewInt(5), nil)
	mix := new(big.Int).Rand(rnd, limit5)
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
	t1P := new(big.Int).Mul(g1a, hb)
	t1P.Mod(t1P, pub.P)

	// t2_p
	g1d := new(big.Int).Exp(pub.G1, d, pub.P)
	he := new(big.Int).Exp(pub.H, e, pub.P)
	t2P := new(big.Int).Mul(g1d, he)
	t2P.Mod(t2P, pub.P)

	// t3_p
	ad := new(big.Int).Add(a, d)
	ad.Mod(ad, P1)
	g1ad := new(big.Int).Exp(pub.G1, ad, pub.P)
	hf := new(big.Int).Exp(pub.H, f, pub.P)
	t3P := new(big.Int).Mul(g1ad, hf)
	t3P.Mod(t3P, pub.P)

	// c
	hashT1p := t1P.Bytes()
	hashT2p := t2P.Bytes()
	hashT3p := t3P.Bytes()
	c_ := sha256.Sum256(append(append(hashT1p, hashT2p...), hashT3p...))
	c := new(big.Int).SetBytes(c_[:])
	c.Mod(c, pub.P)

	// R_v
	cvR := new(big.Int).Mul(c, new(big.Int).SetUint64(vR))
	cvR.Mod(cvR, P1)
	cvR.Sub(P1, cvR)
	RV := new(big.Int).Add(a, cvR)
	RV.Mod(RV, P1)

	// R_r
	crR := new(big.Int).Mul(c, new(big.Int).SetBytes(rR))
	crR.Mod(crR, P1)
	crR.Sub(P1, crR)
	RR := new(big.Int).Add(b, crR)
	RR.Mod(RR, P1)

	// S_v
	cvS := new(big.Int).Mul(c, new(big.Int).SetUint64(vS))
	cvS.Mod(cvS, P1)
	cvS.Sub(P1, cvS)
	SV := new(big.Int).Add(d, cvS)
	SV.Mod(SV, P1)

	// S_r
	crS := new(big.Int).Mul(c, new(big.Int).SetBytes(rS))
	crS.Mod(crS, P1)
	crS.Sub(P1, crS)
	SR := new(big.Int).Add(e, crS)
	SR.Mod(SR, P1)

	// S_or
	crO := new(big.Int).Mul(c, new(big.Int).SetBytes(rO))
	crO.Mod(crO, P1)
	crO.Sub(P1, crO)
	SOr := new(big.Int).Add(f, crO)
	SOr.Mod(SOr, P1)

	// 组装
	bp.C = c.Bytes()
	bp.R_v = RV.Bytes()
	bp.R_r = RR.Bytes()
	bp.S_v = SV.Bytes()
	bp.S_r = SR.Bytes()
	bp.S_or = SOr.Bytes()
	return
}

func VerifyBalanceProof(CM_r, CM_s, CM_o []byte, pub PublicKey, bp BalanceProof) bool {
	// 验证会计平衡证明
	// 初始化
	CmR := new(big.Int).SetBytes(CM_r)
	CmS := new(big.Int).SetBytes(CM_s)
	CmO := new(big.Int).SetBytes(CM_o)
	c := new(big.Int).SetBytes(bp.C)
	RV := new(big.Int).SetBytes(bp.R_v)
	RR := new(big.Int).SetBytes(bp.R_r)
	SV := new(big.Int).SetBytes(bp.S_v)
	SR := new(big.Int).SetBytes(bp.S_r)
	SOr := new(big.Int).SetBytes(bp.S_or)

	// t1_v
	g1rv := new(big.Int).Exp(pub.G1, RV, pub.P)
	hrr := new(big.Int).Exp(pub.H, RR, pub.P)
	cmrc := new(big.Int).Exp(CmR, c, pub.P)
	t1V := new(big.Int).Mul(g1rv, hrr)
	t1V.Mod(t1V, pub.P)
	t1V.Mul(t1V, cmrc)
	t1V.Mod(t1V, pub.P)

	// t2_v
	g1sv := new(big.Int).Exp(pub.G1, SV, pub.P)
	hsr := new(big.Int).Exp(pub.H, SR, pub.P)
	cmsc := new(big.Int).Exp(CmS, c, pub.P)
	t2V := new(big.Int).Mul(g1sv, hsr)
	t2V.Mod(t2V, pub.P)
	t2V.Mul(t2V, cmsc)
	t2V.Mod(t2V, pub.P)

	// t3_v
	rvsv := new(big.Int).Add(RV, SV)
	g1rvsv := new(big.Int).Exp(pub.G1, rvsv, pub.P)
	hsor := new(big.Int).Exp(pub.H, SOr, pub.P)
	cmoc := new(big.Int).Exp(CmO, c, pub.P)
	t3V := new(big.Int).Mul(g1rvsv, hsor)
	t3V.Mod(t3V, pub.P)
	t3V.Mul(t3V, cmoc)
	t3V.Mod(t3V, pub.P)

	// c_v
	hashT1v := t1V.Bytes()
	hashT2v := t2V.Bytes()
	hashT3v := t3V.Bytes()
	c_ := sha256.Sum256(append(append(hashT1v, hashT2v...), hashT3v...))
	cV := new(big.Int).SetBytes(c_[:])
	cV.Mod(cV, pub.P)

	return c.Cmp(cV) == 0
}

func EncryptValue(pub PublicKey, M uint64) (C CypherText, commit Commitment, err error) {
	// ElGamal 加密 uint64 类型数据 M，输出密文 C, 承诺 Commit
	if M > 262144 {
		err = errors.New("加密价值过大")
		return
	}
	// 生成随机数 k
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
	// 加密
	m := new(big.Int).SetUint64(M)
	c1_ := new(big.Int).Exp(pub.G2, k, pub.P)
	c1 := c1_.Bytes()
	//S := new(big.Int).Exp(pub.H, k, pub.P)
	//c2 := S.Mul(S, m)
	//c2.Mod(c2, pub.P)
	rnd := k.Bytes()
	commit = pub.Commit(m, rnd)

	C = CypherText{c1, commit.Commitment}
	return
}

func GenerateEqualityProof(pub1, pub2 PublicKey, C1, C2 Commitment, v uint) (ep EqualityProof) {
	if pub1.P.Cmp(pub2.P) != 0 {
		fmt.Printf("\n零知识证明的两个密码公钥必须在同一个循环群内！\n")
		ep.S = nil
		ep.T = nil
		return
	}
	V := big.NewInt(int64(v))
	Y := new(big.Int).Mul(new(big.Int).SetBytes(C1.Commitment), new(big.Int).SetBytes(C2.Commitment))
	Y.Mod(Y, pub1.P)
	g := [][]byte{pub1.G1.Bytes(), pub2.G1.Bytes(), pub1.H.Bytes(), pub2.H.Bytes()}
	x := [][]byte{V.Bytes(), V.Bytes(), C1.R, C2.R}
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
func GenerateAddressEqualityProof(pub1, pub2 PublicKey, C1, C2 Commitment, addr []byte) (ep EqualityProof) {
	if pub1.P.Cmp(pub2.P) != 0 {
		fmt.Printf("\n零知识证明的两个密码公钥必须在同一个循环群内！\n")
		ep.S = nil
		ep.T = nil
		return
	}
	V := new(big.Int).SetBytes(addr)
	Y := new(big.Int).Mul(new(big.Int).SetBytes(C1.Commitment), new(big.Int).SetBytes(C2.Commitment))
	Y.Mod(Y, pub1.P)
	g := [][]byte{pub1.G1.Bytes(), pub2.G1.Bytes(), pub1.H.Bytes(), pub2.H.Bytes()}
	x := [][]byte{V.Bytes(), V.Bytes(), C1.R, C2.R}
	a := []int{1, -1, 0, 0}
	b := 0
	y := Y.Bytes()
	pub := pub1
	ep.LinearEquationProof = GenerateLinearEquationProof(y, b, a, x, g, pub)
	return ep
}
func GenerateLinearEquationProof(y []byte, b int, a []int, x [][]byte, g [][]byte, pub PublicKey) (lp LinearEquationProof) {
	// 生成线性证明
	// 输入合法判断
	if len(x) != len(g) || len(g) != len(a) {
		fmt.Printf("\n%d,%d,%d,%d", len(x), len(a), len(g))
		lp = LinearEquationProof{
			S: nil,
			T: nil,
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
	lp.T = t.Bytes()
	// 计算 c
	c_mash = append(c_mash, y...)
	c_mash = append(c_mash, lp.T...)
	c_32bit := sha256.Sum256(c_mash)
	c := c_32bit[:]
	c_bi := new(big.Int).SetBytes(c)
	c_bi.Mod(c_bi, pub.P)
	// 计算 S
	for i, vi := range v {
		mash := new(big.Int).SetBytes(x[i])
		mash.Mul(c_bi, mash)
		mash.Sub(vi, mash)
		mash.Mod(mash, P_1)
		lp.S = append(lp.S, mash.Bytes())
	}
	return
}

func VerifyLinearEquationProof(lp LinearEquationProof, y []byte, b int, a []int, g [][]byte, pub PublicKey) bool {
	// 验证线性证明
	if len(lp.S) != len(g) || len(g) != len(a) {
		fmt.Printf("\n%d,%d,%d,%d", len(lp.S), len(a), len(g))
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
	c_mash = append(c_mash, lp.T...)
	c_32bit := sha256.Sum256(c_mash)
	c := c_32bit[:]
	c_bi := new(big.Int).SetBytes(c)
	c_bi.Mod(c_bi, pub.P)
	// 验证 T 值
	t_verify := new(big.Int).SetBytes(y)
	t_verify.Exp(t_verify, c_bi, pub.P)
	for i, gi := range g {
		buf := new(big.Int).SetBytes(gi)
		buf.Exp(buf, new(big.Int).SetBytes(lp.S[i]), pub.P)
		t_verify.Mul(t_verify, buf)
		t_verify.Mod(t_verify, pub.P)
	}
	if t_verify.Cmp(new(big.Int).SetBytes(lp.T)) != 0 {
		fmt.Printf("\nt验证失败！\n")
		return false
	}
	// 验证 S 值
	cb := new(big.Int).SetUint64(uint64(b))
	cb.Mul(c_bi, cb)
	cb.Neg(cb)
	mix := big.NewInt(0)
	for i, ai := range a {
		Ai := new(big.Int).SetInt64(int64(ai))
		Si := new(big.Int).SetBytes(lp.S[i])
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
	//S := new(big.Int).Exp(pub.H, k, pub.P)
	//c2 := S.Mul(S, m)
	//c2.Mod(c2, pub.P)
	rnd := k.Bytes()
	commit = pub.Commit(m, rnd)

	C = CypherText{c1, commit.Commitment}
	return
}
