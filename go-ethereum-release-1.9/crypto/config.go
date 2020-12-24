package crypto

import (
	"github.com/pkg/errors"
)

const (
	CRYPTO_ECC_SH3_AES = 0 //原始标准加密算法
	CRYPTO_SM2_SM3_SM4 = 1 //国密标准算法
)

var CryptoType = CRYPTO_ECC_SH3_AES //CRYPTO_ECC_SH3_AES

func SetCryptoType(cryptoType uint8) {
	if(cryptoType == CRYPTO_ECC_SH3_AES){
		CryptoType = CRYPTO_ECC_SH3_AES
	}else if(cryptoType == CRYPTO_SM2_SM3_SM4){
		CryptoType = CRYPTO_SM2_SM3_SM4
	}
}

func GetCryptoType()(int){
	return CryptoType
}

func BaseCheck(cryptoType byte) error {
	if (int(cryptoType) != CRYPTO_ECC_SH3_AES && int(cryptoType) != CRYPTO_SM2_SM3_SM4 ){
		return errors.New("wrong param on kindCrypto")
	}
	return nil
}