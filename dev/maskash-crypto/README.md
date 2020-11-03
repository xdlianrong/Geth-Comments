# maskash-crypto使用说明

### ElGamal.go 

> ElGamal.go 包含 ElGamal 公私钥生成，加密，解密，签名，验签 算法

|       名称       |  类型  |                  用途                  |
| :--------------: | :----: | :------------------------------------: |
|   `PublicKey`    | 结构体 |              保存公钥信息              |
|   `PrivateKey`   | 结构体 |              保存私钥信息              |
|   `CypherText`   | 结构体 |             保存加密后信息             |
|   `Signature`    | 结构体 |              保存签名信息              |
| **GenerateKeys** |  函数  |          根据字符串生成公私钥          |
|   **Encrypt**    |  函数  | 加密文本*（加密数值用 EncryptValue ）* |
|   **Decrypt**    |  函数  | 解密文本*（解密数值用 DecryptValue ）* |
|     **Sign**     |  函数  |                  签名                  |
|    **Verify**    |  函数  |                  验签                  |

---

#### GenerateKeys

*生成用户公私钥*

```go
func GenerateKeys(info string) (pub PublicKey, priv PrivateKey, err error)
```

##### 输入：

info：生成名

##### 输出：

pub： 用户公钥

priv：用户私钥

err：错误矫正量

##### 示例：

```go
pub, priv, err := GenerateKeys("五点共圆")
if err != nil {
	fmt.Println(err)
	return
}
```

或者

```
pub, priv, _ := GenerateKeys("五点共圆")
```

---

#### Encrypt

*ElGamal加密*

```go
func Encrypt(pub PublicKey, M []byte) (C CypherText)
```

##### 输入：

pub：加密公钥

M：待加密明文

##### 输出：

C：加密后密文

##### 示例：

```go
C := Encrypt(pub, []byte("你们有一个好，全世界甚么地方，你们跑得最快，但是问来问去的问题呀，too simple，sometimes naive，懂得没有？我今天是作为一个长者，我见得太多啦，可以告诉你们一点人生经验，中国人有一句说话叫「闷声发大财」，我就甚么也不说，这是最好的，但是我想我见到你们这样热情，一句话不说也不好，你们刚才在宣传上，将来你们如果在报道上有偏差，你们要负责的。我没有说要钦定，没有任何这样的意思，但是你一定要问我，董先生支持不支持，我们不支持他呀？他现在当特首，我们怎么不支持特首？"))
```

---

#### Decrypt

*ElGamal脱密*

```go
func Decrypt(priv PrivateKey, C CypherText) (M []byte)
```

##### 输入：

priv：脱密私钥

C：待脱密密文

##### 输出：

M：脱密后明文

##### 示例：

```go
M := Decrypt(priv, C)
fmt.Printf("脱密后的明文为：%s\n",string(M))
```

---

#### Sign

*ElGamal签名*

```go
func Sign(priv PrivateKey, m []byte) (sig Signature)
```

##### 输入：

priv：签名私钥

C：待签名字段

##### 输出：

sig：字段签名

##### 示例：

```go
sig := Sign(priv, M)
M_word = string(sig.M)
Mx_word := new(big.Int).SetBytes(sig.M_hash)
R_word := new(big.Int).SetBytes(sig.R)
S_word := new(big.Int).SetBytes(sig.S)
fmt.Printf("明文为：%s\n明文哈希为：%x\n签名R为：%x\n签名S为：%x\n",M_word,Mx_word,R_word,S_word)
```

---

#### Verify

*ElGamal验签*

```go
func Verify(pub PublicKey, sig Signature) bool
```

##### 输入：

pub：验签公钥

sig：待验证签名

##### 输出：

签名是否合法

##### 示例：

```go
var verify bool = Verify(pub, sig)
if verify {
	fmt.Println("签名合法!\n")
} else {
	fmt.Println("签名不合法!\n")
}
```

---

### ZeroKnowledgeProof.go

> 零知识证明，包含一些承诺的生成和零知识证明

|               名称               |  类型  |                             用途                             |
| :------------------------------: | :----: | :----------------------------------------------------------: |
|           `Commitment`           | 结构体 |                  保存承诺及其生成随机数信息                  |
|          `FormatProof`           | 结构体 |                     保存格式正确证明信息                     |
|      `LinearEquationProof`       | 结构体 |                     保存线性恒等证明信息                     |
|         `EqualityProof`          | 结构体 |                       保存相等证明信息                       |
|          `BalanceProof`          | 结构体 |                     保存会计平衡证明信息                     |
|        `BalanceProof_old`        | 结构体 |                  保存会计平衡证明信息（旧）                  |
|             *`keys`*             |  接口  |                           承诺接口                           |
|           ***Commit***           |  方法  |          使用\*big.Int生成承诺*（不建议直接使用）*           |
|       ***CommitByUint64***       |  方法  |           使用uint64生成承诺*（建议用来承诺数值）*           |
|       ***CommitByBytes***        |  方法  |           使用[]byte生成承诺*（建议用来承诺地址）*           |
|      ***VerifyCommitment***      |  方法  | 使用承诺和随机数获取uint64类型原承诺值*（仅适用于数值承诺）* |
|         **EncryptValue**         |  函数  |                加密数值，输出数值密文及其承诺                |
|         **DecryptValue**         |  函数  |        解密数值类型*（使用EncryptValue加密的）*的密文        |
|        **EncryptAddress**        |  函数  |                加密地址，输出地址密文及其承诺                |
|        **DecryptAddress**        |  函数  |       解密地址类型*（使用EncryptAddress加密的）*的密文       |
|     **GenerateFormatProof**      |  函数  |                证明者生成格式正确证明（数值）                |
|  **GenerateAddressFormatProof**  |  函数  |                证明者生成格式正确证明（地址）                |
|      **VerifyFormatProof**       |  函数  |                    验证者验证格式正确证明                    |
| **GenerateLinearEquationProof**  |  函数  |             证明者生成线性恒等证明（不直接使用）             |
|  **VerifyLinearEquationProof**   |  函数  |             证明者验证线性恒等证明（不直接使用）             |
|    **GenerateEqualityProof**     |  函数  |                  证明者生成相等证明（数值）                  |
| **GenerateAddressEqualityProof** |  函数  |                  证明者生成相等证明（地址）                  |
|     **VerifyEqualityProof**      |  函数  |                      证明者验证相等证明                      |
|     **GenerateBalanceProof**     |  函数  |                    证明者生成会计平衡证明                    |
|      **VerifyBalanceProof**      |  函数  |                    证明者验证会计平衡证明                    |
|   **GenerateBalanceProof_old**   |  函数  |                 证明者生成会计平衡证明（旧）                 |
|    **VerifyBalanceProof_old**    |  函数  |                 验证者验证会计平衡证明（旧）                 |

---

#### Commit

*生成承诺 （本函数不建议直接使用）*

```go
func (pub PublicKey) Commit(v *big.Int, rnd []byte) Commitment
func (priv PrivateKey) Commit(v *big.Int, rnd []byte) Commitment
```

##### 输入：

pub/priv：生成承诺的密钥

v：生成承诺的值

rnd：生成承诺的随机数

##### 输出：

承诺 Commitment 结构体（包含承诺与生成用随机数）

##### 示例：

```go
var commitment Commitment = pub.Commit(v, rnd)
```

---

#### CommitByUint64

*使用 uint64 生成承诺*

```go
func (pub PublicKey) CommitByUint64(v uint64, rnd []byte) Commitment
func (priv PrivateKey) CommitByUint64(v uint64, rnd []byte) Commitment
```

##### 输入：

pub/priv：生成承诺的密钥

v：生成承诺的值

rnd：生成承诺的随机数

##### 输出：

承诺 Commitment 结构体（包含承诺与生成用随机数）

##### 示例：

```go
var commitment Commitment = pub.CommitByUint64(v, rnd)
```

---

#### CommitByBytes

*使用 []byte 生成承诺*

```go
func (pub PublicKey) CommitByBytes(b []byte, rnd []byte) Commitment
func (priv PrivateKey) CommitByBytes(b []byte, rnd []byte) Commitment
```

##### 输入：

pub/priv：生成承诺的密钥

b：生成承诺的值

rnd：生成承诺的随机数

##### 输出：

承诺 Commitment 结构体（包含承诺与生成用随机数）

##### 示例：

```go
var commitment Commitment = pub.CommitByBytes(b, rnd)
```

---

#### VerifyCommitment

*通过承诺及其生成随机数获取被承诺值*

```go
func (pub PublicKey) VerifyCommitment(commit Commitment) uint64
func (priv PrivateKey) VerifyCommitment(commit Commitment) uint64
```

##### 输入：

pub/priv：生成承诺的密钥

commit：待验证承诺结构体

##### 输出：

生成该承诺的 uint64 值

##### 示例：

```go
var commitment Commitment = pub.CommitByUint64((uint64)114514, rnd)
v := pub.VerifyCommitment(commitment)
fmt.Printf("生成该承诺的值为%d\n", v) //114514
```

> ##### 注意
>
> Commitment 结构体内包含两个数值，一个是承诺本身，另一个是生成承诺的随机数，在知道随机数的时候可以获取承诺值（遍历），所以最后上链的承诺不应包含随机数，只应上链 Commitment.commitment

---

#### EncryptValue

*对数值做 AH-Elgamal 加密，并做承诺*

```go
func EncryptValue(pub PublicKey, M uint64) (C CypherText, commit Commitment, err error)
```

##### 输入：

pub：加密公钥 / 生成承诺的公钥

M：需要加密的数值

##### 输出：

C：密文

commit：承诺

err：错误传递变量

##### 示例：

```go
C, commit, err := EncryptValue(pub,10)
if err != nil {
	fmt.Print(err)
	return
}
```

---

#### DecryptValue

*对数值做 AH-Elgamal 脱密*

```go
func DecryptValue(priv PrivateKey, C CypherText) (v uint64)
```

##### 输入：

priv：脱密私钥

C：需要脱密的密文

##### 输出：

v：脱密后的数值

##### 示例：

```go
C, commit, _ := EncryptValue(pub,10)
v := DecryptValue(priv, C)
```

---

#### EncryptAddress

*对地址做 AH-Elgamal 加密，并做承诺*

```go
func EncryptAddress(pub PublicKey, addr []byte) (C CypherText, commit Commitment, err error)
```

##### 输入：

pub：加密公钥 / 生成承诺的公钥

addr：需要加密的地址

##### 输出：

C：密文

commit：承诺

err：错误传递变量

##### 示例：

```go
pub1, priv1, _ := GenerateKeys("98年抗洪慷慨宣讲")
// pub2, _, _ := GenerateKeys("90年春晚安详致辞")
pub3, _, _ := GenerateKeys("86年和华莱士谈笑风生")

C1, CM1, _ := EncryptAddress(pub1, pub3.G1.Bytes())
```

---

#### DecryptAddress

*对数值做 AH-Elgamal 脱密*

```go
func DecryptAddress(priv PrivateKey, C CypherText, PkPool [][]byte) []byte
```

##### 输入：

priv：脱密私钥

C：需要脱密的密文

PkPool：监管者的地址公钥池（个人脱密只需要用自己的地址验证即可，即公钥池里只有自己的地址）

##### 输出：

脱密后的地址

##### 示例：

```go
pub1, priv1, _ := GenerateKeys("98年抗洪慷慨宣讲")
pub2, _, _ := GenerateKeys("90年春晚安详致辞")
pub3, _, _ := GenerateKeys("86年和华莱士谈笑风生")

C1, CM1, _ := EncryptAddress(pub1, pub3.G1.Bytes())

PkPool := [][]byte{pub1.G1.Bytes(), pub2.G1.Bytes(), pub3.G1.Bytes()}
p3g1 := DecryptAddress(priv1, C1, PkPool)
```

---

#### GenerateFormatProof

*生成格式正确证明*

```go
func GenerateFormatProof(pub PublicKey, v uint64, r []byte) (fp FormatProof)
```

##### 输入：

pub：生成者公钥

v：生成先前承诺的值

r：生成先前承诺的随机数

##### 输出：

fp：格式正确证明结构体（就是你们需要最后上链的那个）

##### 示例：

```go
C, commit, err := EncryptValue(pub,10)
fp := GenerateFormatProof(pub, v, commit.r)
```

---

#### GenerateAddressFormatProof

*生成地址的格式正确证明*

```go
func GenerateAddressFormatProof(pub PublicKey, addr []byte, r []byte) (fp FormatProof)
```

##### 输入：

pub：生成者公钥

v：生成先前承诺的值

r：生成先前承诺的随机数

##### 输出：

fp：格式正确证明结构体（就是你们需要最后上链的那个）

##### 示例：

```go
pub1, priv1, _ := GenerateKeys("98年抗洪慷慨宣讲")
// pub2, _, _ := GenerateKeys("90年春晚安详致辞")
pub3, _, _ := GenerateKeys("86年和华莱士谈笑风生")
fp := GenerateAddressFormatProof(pub1, pub3.G1.Bytes(), CM1.r)
```

---

#### VerifyFormatProof

*验证格式正确证明*

```go
func VerifyFormatProof(C CypherText, pub PublicKey, fp FormatProof) bool
```

##### 输入：

C：待验证的密文

pub：生成者公钥

fp：生成者生成的格式正确证明

##### 输出：

格式正确证明是否验证通过

##### 示例：

```go
C, commit, err := EncryptValue(pub,10)
fp := GenerateFormatProof(pub, v, commit.r)
verify := VerifyFormatProof(C, pub, fp)
if verify {
   fmt.Println("格式正确证明验证通过！")
} else {
   fmt.Println("格式正确证明验证不通过！")
}
```

---

#### GenerateLinearEquationProof

<p style="color: red">注意！本函数不需要直接调用，可跳过</p>

*生成线性恒等证明*

```go
func GenerateLinearEquationProof(y []byte, b int, a []int, x [][]byte, g [][]byte, pub PublicKey) (lp LinearEquationProof)
```

---

#### VerifyLinearEquationProof

<p style="color: red">注意！本函数不需要直接调用，可跳过</p>

*验证线性恒等证明*

```go
func VerifyLinearEquationProof(lp LinearEquationProof, y []byte, b int, a []int, g [][]byte, pub PublicKey) bool
```

---

#### GenerateEqualityProof

*生成相等证明*

```go
func GenerateEqualityProof(pub1, pub2 PublicKey, C1, C2 Commitment, v uint) (ep EqualityProof)
```

##### 输入：

pub1, pub2：生成承诺的两个公钥

C1, C2：生成的两个承诺

v：用于生成承诺的金额

##### 输出：

ep：相等证明结构体

##### 示例：

```go
C1, CM1, err1 := EncryptValue(pub1, uint64(value))
C2, CM2, err2 := EncryptValue(pub2, uint64(value))
ep := GenerateEqualityProof(pub1, pub2, CM1, CM2, value)
```

---

#### GenerateAddressEqualityProof

*生成相等证明（地址）*

```go
func GenerateAddressEqualityProof(pub1, pub2 PublicKey, C1, C2 Commitment, addr []byte) (ep EqualityProof)
```

##### 输入：

pub1, pub2：生成承诺的两个公钥

C1, C2：生成的两个承诺

addr：用于生成承诺的地址

##### 输出：

ep：相等证明结构体

##### 示例：

```go
C1, CM1, _ := EncryptAddress(pub1, pub3.G1.Bytes())
C2, CM2, _ := EncryptAddress(pub2, pub3.G1.Bytes())
ep := GenerateAddressEqualityProof(pub1, pub2, CM1, CM2, pub3.G1.Bytes())
```

---

#### VerifyEqualityProof

*验证相等证明*

```go
func VerifyEqualityProof(pub1, pub2 PublicKey, C1, C2 CypherText, ep EqualityProof) bool
```

##### 输入：

pub1, pub2：生成承诺的两个公钥

C1, C2：生成的两个密文

eq：生成者生成的相等证明

##### 输出：

相等证明是否验证通过

##### 示例：

```
ep := GenerateAddressEqualityProof(pub1, pub2, CM1, CM2, pub3.G1.Bytes())
sign := VerifyEqualityProof(pub1, pub2, C1, C2, ep)
if sign{
	fmt.Printf("相等证明验证通过！\n")
}else{
	fmt.Printf("相等证明验证不通过！\n")
}
```

---

#### GenerateBalanceProof

*生成会计平衡证明（证明 ∑v_o = ∑v_s）*

```go
func GenerateBalanceProof(pub PublicKey, C_o, C_s []Commitment, v_o, v_s []uint) (bp BalanceProof)
```

##### 输入：

pub：生成者公钥

C_o：原金额承诺数组

C_s：开销金额承诺数组

v_o：原金额数组

v_s：开销金额数组

##### 输出：

bp：会计平衡证明结构体

##### 示例：

```go
// 1+2+3+4+5 = 3+5+7
var v_o = []uint{1, 2, 3, 4, 5}
var v_s = []uint{3, 5, 7}

C_o := make([]CypherText, len(v_o))
CM_o := make([]Commitment, len(v_o))
C_s := make([]CypherText, len(v_s))
CM_s := make([]Commitment, len(v_s))

for i, vo := range v_o{
	C_o[i], CM_o[i], _ = EncryptValue(pub1, uint64(vo))
}
for i, vs := range v_s{
	C_s[i], CM_s[i], _ = EncryptValue(pub1, uint64(vs))
}

bp := GenerateBalanceProof(pub1, CM_o, CM_s, v_o, v_s)
```

---

#### VefityBalanceProof

*验证会计平衡证明（证明 ∑v_o = ∑v_s）*

```go
func VerifyBalanceProof(pub PublicKey, C_o, C_s []CypherText, bp BalanceProof) bool
```

##### 输入：

pub：生成者公钥

C_o：原金额密文数组

C_s：开销金额密文数组

bp：生成者生成的会计平衡证明

##### 输出：

会计平衡证明是否验证通过

##### 示例：

```go
// 1+2+3+4+5 = 3+5+7
var v_o = []uint{1, 2, 3, 4, 5}
var v_s = []uint{3, 5, 7}

C_o := make([]CypherText, len(v_o))
CM_o := make([]Commitment, len(v_o))
C_s := make([]CypherText, len(v_s))
CM_s := make([]Commitment, len(v_s))

for i, vo := range v_o{
	C_o[i], CM_o[i], _ = EncryptValue(pub1, uint64(vo))
}
for i, vs := range v_s{
	C_s[i], CM_s[i], _ = EncryptValue(pub1, uint64(vs))
}

bp := GenerateBalanceProof(pub1, CM_o, CM_s, v_o, v_s)
verify := VerifyBalanceProof(pub1, C_o, C_s, bp)

if verify {
	fmt.Println("会计平衡证明验证通过！")
} else {
	fmt.Println("会计平衡证明验证不通过！")
}
```

---

#### GenerateBalanceProof_old

*生成会计平衡证明（即证明以下 v_r + v_s = v_o，但不暴露这三个值）*

```go
func GenerateBalanceProof_old(pub PublicKey, v_r, v_s, v_o uint64, r_r, r_s, r_o []byte) (bp BalanceProof_old)
```

##### 输入：

pub：生成者公钥

v_r：找零金额

v_s：花费金额

v_o：原款金额

r_r：生成找零金额承诺的随机数

r_s：生成花费金额承诺的随机数

r_o：生成原款金额承诺的随机数

##### 输出：

bp：会计平衡证明结构体

##### 示例：

```go
fmt.Printf("设定如下合法交易（找零+付款=原金额）：\n")
var v_r, v_s uint64= 11, 16
var v_o uint64= v_r + v_s
fmt.Printf("v_r:%d\nv_s:%d\nv_o:%d\n", v_r, v_s, v_o)

Ev_r, CM_r, err1 := EncryptValue(pub, v_r)
Ev_s, CM_s, err2 := EncryptValue(pub, v_s)
Ev_o, CM_o, err3 := EncryptValue(pub, v_o)
if err1 != nil || err2 != nil || err3 != nil{
    fmt.Print(err)
    return
}

fmt.Printf("\n生成会计平衡证明：\n")
bp := GenerateBalanceProof_old(pub, v_r, v_s, v_o, CM_r.r, CM_s.r, CM_o.r)
```

---

#### VefityBalanceProof_old

*验证会计平衡证明（即证明 v_r + v_s = v_o）*

```go
func VerifyBalanceProof_old(CM_r, CM_s, CM_o []byte, pub PublicKey, bp BalanceProof_old) bool
```

##### 输入：

CM_r：找零金额承诺

CM_s：花费金额承诺

CM_o：原款金额承诺

pub：生成者公钥

bp：生成者生成的会计平衡证明

##### 输出：

会计平衡证明是否验证通过

##### 示例：

```go
Ev_r, CM_r, err1 := EncryptValue(pub, v_r)
Ev_s, CM_s, err2 := EncryptValue(pub, v_s)
Ev_o, CM_o, err3 := EncryptValue(pub, v_o)
if err1 != nil || err2 != nil || err3 != nil{
    fmt.Print(err)
    return
}

fmt.Printf("生成会计平衡证明：\n")
bp := GenerateBalanceProof_old(pub, v_r, v_s, v_o, CM_r.r, CM_s.r, CM_o.r)
fmt.Printf("C:%x\nR_v:%x\nR_r:%x\nS_v:%x\nS_r:%x\nS_or:%x\n",bp.C, bp.R_v, bp.R_r, bp.S_v, bp.S_r, bp.S_or)

fmt.Printf("\n验证会计平衡证明：\n")
verify := VerifyBalanceProof_old(CM_r.commitment, CM_s.commitment, CM_o.commitment, pub, bp)
if verify {
    fmt.Println("会计平衡证明验证通过！")
} else {
    fmt.Println("会计平衡证明验证不通过！")
}
```

---

### example.go

> 一些使用的例子

|   函数名    |                             内容                             |
| :---------: | :----------------------------------------------------------: |
| example_1() |     密钥生成，加密文本，解密文本，对文本签名，验证该签名     |
| example_2() | 密钥生成，加密数值（并做承诺），解密数值，生成格式正确证明，验证格式正确证明 |
| example_3() | 密钥生成，生成承诺，生成会计平衡证明（旧），验证会计平衡证明（旧） |
| example_4() | 密钥生成，加密数值（并做承诺），生成相等证明，验证相等证明，生成会计平衡证明，验证会计平衡证明 |
| example_5() | 密钥生成，加密地址（并作承诺），生成格式正确证明，验证格式正确证明，生成相等证明，验证相等证明 |

