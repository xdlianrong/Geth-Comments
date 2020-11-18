# ECDSA

#### 1.通过现有编码标准生成ecdsa公私钥

```go
func ToECDSA(d []byte) (*ecdsa.PrivateKey, error) {}
```

#### 2.通过原来编码标准生成ecdsa公私钥（0前缀）

```go
func ToECDSAUnsafe(d []byte) *ecdsa.PrivateKey {}
```

#### 4.通过十六进制字符串生成ecdsa公私钥

```go
func HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {}
```

#### 5.生成一对ecdsa公私钥

```go
func GenerateKey() (*ecdsa.PrivateKey, error) {}
```

#### 5.通过指定参数生产ecdsa公私钥的具体函数

```go
func toECDSA(d []byte, strict bool) (*ecdsa.PrivateKey, error) {
   priv := new(ecdsa.PrivateKey)
    // 设置椭圆曲线secp256k1
   priv.PublicKey.Curve = S256()
   if strict && 8*len(d) != priv.Params().BitSize {
      return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
   }
    //私钥导入
   priv.D = new(big.Int).SetBytes(d)

   // The priv.D must < N
   if priv.D.Cmp(secp256k1N) >= 0 {
      return nil, fmt.Errorf("invalid private key, >=N")
   }
   // The priv.D must not be zero or negative.
   if priv.D.Sign() <= 0 {
      return nil, fmt.Errorf("invalid private key, zero or negative")
   }

    //根据私钥生成公钥
   priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
   if priv.PublicKey.X == nil {
      return nil, errors.New("invalid private key")
   }
   return priv, nil
}
```

#### 6.将私钥转换为字节数组类型

```go
func FromECDSA(priv *ecdsa.PrivateKey) []byte {}
```

#### 7.将非序列化的公钥字节数组转换为椭圆曲线库标准的公钥

```go
func UnmarshalPubkey(pub []byte) (*ecdsa.PublicKey, error) {}
```

#### 8.通过公钥信息和椭圆曲线参数生成指定长度字节数组的算法

```go
func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(S256(), pub.X, pub.Y)
}
```

#### 9.从文件中的指定key加载ecdsa私钥

```go
func LoadECDSA(file string) (*ecdsa.PrivateKey, error) {}
```

#### 10.把私钥写入文件指定文件

```go
func SaveECDSA(file string, key *ecdsa.PrivateKey) error {}
```

#### 11.ecdsa验证签名的参数v，r，s

```go
func ValidateSignatureValues(v byte, r, s *big.Int, homestead bool) bool {}
```

#### 12.通过公钥生成账户地址

```go
func PubkeyToAddress(p ecdsa.PublicKey) common.Address {}
```

#### 13.通过签名hash和签名返回公钥的压缩形式

```go
func Ecrecover(hash, sig []byte) ([]byte, error) {}
```

#### 14.通过签名hash和签名返回公钥

```go
func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {}
```

#### 15.通过私钥和要签名的内容返回ecdsa签名

```go
func Sign(digestHash []byte, prv *ecdsa.PrivateKey) (sig []byte, err error) {}
```

#### 16.验证签名内容

```
func VerifySignature(pubkey, digestHash, signature []byte) bool {}
```

#### 17.返回椭圆曲线参数

```go
func S256() elliptic.Curve {}
```

#### 18.压缩公钥为33字节

```go
func DecompressPubkey(pubkey []byte) (*ecdsa.PublicKey, error) {}
```

#### 19.将33字节的公钥解压缩

```go
func CompressPubkey(pubkey *ecdsa.PublicKey) []byte {}
```

## SHA3/SHA256

#### 20.对输入数据计算并返回Keccak256哈希

```go
func Keccak256(data ...[]byte) []byte{}
```

#### 21.对输入数据计算并返回Keccak256哈希，将其转换为内部哈希数据结构

```go
func Keccak256Hash(data ...[]byte) (h common.Hash) {}
```

#### 22.对输入数据计算并返回Keccak512哈希

```go
func Keccak512(data ...[]byte) []byte {}
```

#### 23.创建一个以太坊地址时，会用到Keccak256

```go
func CreateAddress(b common.Address, nonce uint64) common.Address {
   data, _ := rlp.EncodeToBytes([]interface{}{b, nonce})
   return common.BytesToAddress(Keccak256(data)[12:])
}
```

#### 24.创建合约地址

```go
func CreateAddress2(b common.Address, salt [32]byte, inithash []byte) common.Address {
   return common.BytesToAddress(Keccak256([]byte{0xff}, b.Bytes(), salt[:], inithash)[12:])
}
```

