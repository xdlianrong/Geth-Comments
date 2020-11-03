package main

import (
	"fmt"
	"math/big"
)

func main() {
	//exmaple_1()
	//example_2()
	example_3()
	//example_4()
	example_5()
}

func exmaple_1() {
	fmt.Printf("\n\n========================= EXAMPLE 1 =========================\n\n")
	pub, priv, err := GenerateKeys("五点共圆")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("公钥：\nP:%x\nG1:%x\nG2:%x\nH:%x\n私钥：\nX:%x\n", pub.P, pub.G1, pub.G2, pub.H, priv.X)

	C := Encrypt(pub, []byte("你们有一个好，全世界甚么地方，你们跑得最快，但是问来问去的问题呀，too simple，sometimes naive，懂得没有？我今天是作为一个长者，我见得太多啦，可以告诉你们一点人生经验，中国人有一句说话叫「闷声发大财」，我就甚么也不说，这是最好的，但是我想我见到你们这样热情，一句话不说也不好，你们刚才在宣传上，将来你们如果在报道上有偏差，你们要负责的。我没有说要钦定，没有任何这样的意思，但是你一定要问我，董先生支持不支持，我们不支持他呀？他现在当特首，我们怎么不支持特首？"))
	fmt.Printf("\n加密后的密文C1为：%x\n加密后的密文C2为：%x\n", C.C1, C.C2)

	M := Decrypt(priv, C)
	M_word := string(M)
	fmt.Printf("\n解密后的明文为：%s\n", M_word)

	sig := Sign(priv, M)
	M_word = string(sig.M)
	Mx_word := new(big.Int).SetBytes(sig.M_hash)
	R_word := new(big.Int).SetBytes(sig.R)
	S_word := new(big.Int).SetBytes(sig.S)
	fmt.Printf("\n明文为：%s\n明文哈希为：%x\n签名R为：%x\n签名S为：%x\n", M_word, Mx_word, R_word, S_word)

	fmt.Printf("\n验证签名是否合法：\n")
	verify := Verify(pub, sig)
	if verify {
		fmt.Println("签名合法!\n")
	} else {
		fmt.Println("签名不合法!\n")
	}

	fmt.Printf("\n篡改签名后验证签名是否合法：\n")
	sig.S[0] += 1
	verify = Verify(pub, sig)
	if verify {
		fmt.Println("签名合法!\n")
	} else {
		fmt.Println("签名不合法!\n")
	}
}

func example_2() {
	fmt.Printf("\n\n========================= EXAMPLE 2 =========================\n\n")
	pub, priv, err := GenerateKeys("人生经验")

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("公钥：\nP:%x\nG1:%x\nG2:%x\nH:%x\n私钥：\nX:%x\n", pub.P, pub.G1, pub.G2, pub.H, priv.X)
	fmt.Printf("\n加密及承诺过程：\n")
	C, commit, err := EncryptValue(pub, 10)
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Printf("\n加密后交易金额：\nc1(=G2*R):%x\nc2(=G1*V+H*R):%x\n\n交易承诺：\nCM_v(=G1*V+H*R):%x\nR:%x\n", C.C1, C.C2, commit.commitment, commit.r)

	v := DecryptValue(priv, C)
	fmt.Printf("解密得到交易金额：%d\n", v)

	fmt.Printf("\n生成格式正确证明：\n")
	fp := GenerateFormatProof(pub, v, commit.r)
	fmt.Printf("\nC:%x\nZ1:%x\nZ2:%x\n", fp.C, fp.Z1, fp.Z2)

	fmt.Printf("\n验证格式正确证明：\n")
	verify := VerifyFormatProof(C, pub, fp)
	if verify {
		fmt.Println("格式正确证明验证通过！")
	} else {
		fmt.Println("格式正确证明验证不通过！")
	}
}

func example_3() {
	fmt.Printf("\n\n========================= EXAMPLE 3 =========================\n\n")
	pub, priv, err := GenerateKeys("八国语言")
	setRegulator(&pub)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("公钥：\nP:%x\nG1:%x\nG2:%x\nH:%x\n私钥：\nX:%x\n", pub.P, pub.G1, pub.G2, pub.H, priv.X)

	fmt.Printf("\n设定如下合法交易（找零+付款=原金额）：\n")
	var v_r, v_s uint64 = 11, 16
	var v_o = v_r + v_s
	fmt.Printf("v_r:%d\nv_s:%d\nv_o:%d\n", v_r, v_s, v_o)

	Ev_r, CM_r, err1 := EncryptValue(pub, v_r)
	Ev_s, CM_s, err2 := EncryptValue(pub, v_s)
	Ev_o, CM_o, err3 := EncryptValue(pub, v_o)
	if err1 != nil || err2 != nil || err3 != nil {
		fmt.Print(err)
		return
	}
	fmt.Printf("\nEv_r:%d\nEv_s:%d\nEv_o:%d\n", Ev_r.C2, Ev_s.C2, Ev_o.C2)

	fmt.Printf("\n生成会计平衡证明：\n")
	fmt.Printf("\nCM_o_commitment:%x\nCM_o_r:%x\n", CM_o.commitment, CM_o.r)
	bp := GenerateBalanceProof_old(pub, v_r, v_s, 0, CM_r.r, CM_s.r, CM_o.r)
	fmt.Printf("C:%x\nR_v:%x\nR_r:%x\nS_v:%x\nS_r:%x\nS_or:%x\n", bp.C, bp.R_v, bp.R_r, bp.S_v, bp.S_r, bp.S_or)

	fmt.Printf("\n验证会计平衡证明：\n")
	verify := VerifyBalanceProof_old(CM_r.commitment, CM_s.commitment, CM_o.commitment, pub, bp)
	if verify {
		fmt.Println("会计平衡证明验证通过！")
	} else {
		fmt.Println("会计平衡证明验证不通过！")
	}

	fmt.Printf("\n设定如下不合法交易（找零+付款≠原金额）：\n")
	v_r, v_s, v_o = 11, 16, 20
	fmt.Printf("v_r:%d\nv_s:%d\nv_o:%d\n", v_r, v_s, v_o)

	Ev_r, CM_r, err1 = EncryptValue(pub, v_r)
	Ev_s, CM_s, err2 = EncryptValue(pub, v_s)
	Ev_o, CM_o, err3 = EncryptValue(pub, v_o)
	if err1 != nil || err2 != nil || err3 != nil {
		fmt.Print(err)
		return
	}
	fmt.Printf("\nEv_r:%d\nEv_s:%d\nEv_o:%d\n", Ev_r.C2, Ev_s.C2, Ev_o.C2)

	fmt.Printf("\n生成会计平衡证明：\n")
	bp = GenerateBalanceProof_old(pub, v_r, v_s, v_o, CM_r.r, CM_s.r, CM_o.r)
	fmt.Printf("C:%x\nR_v:%x\nR_r:%x\nS_v:%x\nS_r:%x\nS_or:%x\n", bp.C, bp.R_v, bp.R_r, bp.S_v, bp.S_r, bp.S_or)

	fmt.Printf("\n验证会计平衡证明：\n")
	verify = VerifyBalanceProof_old(CM_r.commitment, CM_s.commitment, CM_o.commitment, pub, bp)
	if verify {
		fmt.Println("会计平衡证明验证通过！")
	} else {
		fmt.Println("会计平衡证明验证不通过！")
	}
}

func example_4() {
	fmt.Printf("\n\n========================= EXAMPLE 4 =========================\n\n")
	pub1, _, err1 := GenerateKeys("苟利国家生死以")
	pub2, _, err2 := GenerateKeys("岂因祸福避趋之")
	if err1 != nil {
		fmt.Println(err1)
		return
	}
	if err2 != nil {
		fmt.Println(err2)
		return
	}
	fmt.Printf("\npub1：%x%x%x%x\n", pub1.G1, pub1.G2, pub1.P, pub1.H)
	fmt.Printf("\npub2：%0*x\n%0*x\n%0*x\n%0*x\n", 64, pub2.G1, 64, pub2.G2, 64, pub2.P, 64, pub2.H)
	//234b7f8dcdec50b47127a9ba7f03d629bd751b571ff07ac8879c4ca0a91b146205e72bd1ac5e39bcf34cbbbcf48a13edc865f862a85ce69866be24e078a3942a33333f914834ced561c145797d9b5782719dbd1b43a668d4b01151f9c0e67d9f1569899100a4ce41de3c549b649ff72d5d7c9fe8983c244cc28f2ce84b2a758c
	var value uint = 16
	fmt.Printf("\n设定金额：%d\n", value)
	fmt.Printf("使用不同公钥加密相同内容\n")
	C1, CM1, err1 := EncryptValue(pub1, uint64(value))
	C2, CM2, err2 := EncryptValue(pub2, uint64(value))

	fmt.Printf("生成相等证明：\n")
	ep := GenerateEqualityProof(pub1, pub2, CM1, CM2, value)
	fmt.Printf("s:%x\nt:%x\n", ep.s, ep.t)

	fmt.Printf("\n验证相等证明：\n")
	sign := VerifyEqualityProof(pub1, pub2, C1, C2, ep)
	if sign {
		fmt.Printf("相等证明验证通过！\n")
	} else {
		fmt.Printf("相等证明验证不通过！\n")
	}

	fmt.Printf("\n设定如下合法交易：\n")
	var v_o = []uint{1, 2, 3, 4, 5}
	var v_s = []uint{3, 5, 7}
	fmt.Printf("v_o:%d\nv_s:%d\n", v_o, v_s)
	C_o := make([]CypherText, len(v_o))
	CM_o := make([]Commitment, len(v_o))
	C_s := make([]CypherText, len(v_s))
	CM_s := make([]Commitment, len(v_s))
	for i, vo := range v_o {
		C_o[i], CM_o[i], _ = EncryptValue(pub1, uint64(vo))
	}
	for i, vs := range v_s {
		C_s[i], CM_s[i], _ = EncryptValue(pub1, uint64(vs))
	}
	fmt.Printf("\nC_o:%x\nC_s:%x\n", C_o, C_s)
	fmt.Printf("\nCM_o:%x\nCM_s:%x\n", CM_o, CM_s)

	fmt.Printf("\n生成会计平衡证明：\n")
	bp := GenerateBalanceProof(pub1, CM_o, CM_s, v_o, v_s)
	fmt.Printf("s:%x\nt:%x\n", bp.s, bp.t)

	fmt.Printf("\n验证会计平衡证明：\n")
	verify := VerifyBalanceProof(pub1, C_o, C_s, bp)
	if verify {
		fmt.Println("会计平衡证明验证通过！")
	} else {
		fmt.Println("会计平衡证明验证不通过！")
	}

	fmt.Printf("\n设定如下不合法交易：\n")
	v_o = []uint{1, 2, 3, 4, 5}
	v_s = []uint{3, 6, 7}
	fmt.Printf("v_o:%d\nv_s:%d\n", v_o, v_s)
	C_o = make([]CypherText, len(v_o))
	CM_o = make([]Commitment, len(v_o))
	C_s = make([]CypherText, len(v_s))
	CM_s = make([]Commitment, len(v_s))
	for i, vo := range v_o {
		C_o[i], CM_o[i], _ = EncryptValue(pub1, uint64(vo))
	}
	for i, vs := range v_s {
		C_s[i], CM_s[i], _ = EncryptValue(pub1, uint64(vs))
	}
	fmt.Printf("\nC_o:%x\nC_s:%x\n", C_o, C_s)

	fmt.Printf("\n生成会计平衡证明：\n")
	bp = GenerateBalanceProof(pub1, CM_o, CM_s, v_o, v_s)
	fmt.Printf("s:%x\nt:%x\n", bp.s, bp.t)

	fmt.Printf("\n验证会计平衡证明：\n")
	verify = VerifyBalanceProof(pub1, C_o, C_s, bp)
	if verify {
		fmt.Println("会计平衡证明验证通过！")
	} else {
		fmt.Println("会计平衡证明验证不通过！")
	}
}

func example_5() {
	fmt.Printf("\n\n========================= EXAMPLE 5 =========================\n\n")
	pub1, priv1, _ := GenerateKeys("98年抗洪慷慨宣讲")
	pub2, _, _ := GenerateKeys("90年春晚安详致辞")
	pub3, _, _ := GenerateKeys("86年和华莱士谈笑风生")
	fmt.Printf("\n使用公钥:\npub1:%x\npub2:%x\n对pub3.G1:%x\n进行加密\n", pub1, pub2, pub3.G1)
	aha := pub3.G1.Bytes()
	fmt.Println(aha)
	C1, CM1, _ := EncryptAddress(pub1, pub3.G1.Bytes())
	C2, CM2, _ := EncryptAddress(pub2, pub3.G1.Bytes())

	fmt.Printf("生成相等证明：\n")
	ep := GenerateAddressEqualityProof(pub1, pub2, CM1, CM2, pub3.G1.Bytes())
	fmt.Printf("s:%x\nt:%x\n", ep.s, ep.t)

	fmt.Printf("\n验证相等证明：\n")
	sign := VerifyEqualityProof(pub1, pub2, C1, C2, ep)
	if sign {
		fmt.Printf("相等证明验证通过！\n")
	} else {
		fmt.Printf("相等证明验证不通过！\n")
	}

	fmt.Printf("\n生成格式正确证明：\n")
	fp := GenerateAddressFormatProof(pub1, pub3.G1.Bytes(), CM1.r)
	fmt.Printf("\nC:%x\nZ1:%x\nZ2:%x\n", fp.C, fp.Z1, fp.Z2)

	fmt.Printf("\n验证格式正确证明：\n")
	verify := VerifyFormatProof(C1, pub1, fp)
	if verify {
		fmt.Println("格式正确证明验证通过！")
	} else {
		fmt.Println("格式正确证明验证不通过！")
	}

	fmt.Printf("\n开始解密地址：\n")
	PkPool := [][]byte{pub1.G1.Bytes(), pub2.G1.Bytes(), pub3.G1.Bytes()}
	p3g1 := DecryptAddress(priv1, C1, PkPool)
	fmt.Printf("\n原地址为：%x\n解密后地址为：%x\n", pub3.G1, p3g1)
}
