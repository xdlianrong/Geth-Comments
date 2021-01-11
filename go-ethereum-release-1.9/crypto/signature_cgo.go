// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// +build !nacl,!js,cgo

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto/gm/sm2"
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

// Ecrecover returns the uncompressed public key that created the given signature.
func Ecrecover(hash, sig []byte) ([]byte, error) {
	if CryptoType == CRYPTO_ECC_SH3_AES {
		return secp256k1.RecoverPubkey(hash, sig[:65])
		//return secp256k1.RecoverPubkey(hash, sig)
	}
	if CryptoType == CRYPTO_SM2_SM3_SM4 {
		ee := new(big.Int).SetBytes(sig[65:])
		smpub, err := sm2.SigToPub(hash, sig[:65], nil, ee)
		if err != nil {
			return nil, err
		}
		return FromECDSAPub(smpub), nil
	}
	return nil, nil
}

// SigToPub returns the public key that created the given signature.
func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	if CryptoType == CRYPTO_ECC_SH3_AES {
		s, err := Ecrecover(hash, sig)
		if err != nil {
			return nil, err
		}
		x, y := elliptic.Unmarshal(S256(), s)
		return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}, nil
	}
	if CryptoType == CRYPTO_SM2_SM3_SM4 {
		ee := new(big.Int).SetBytes(sig[65:])
		smpub, err := sm2.SigToPub(hash, sig[:65], nil, ee)
		if err != nil {
			return nil, err
		}
		return smpub, nil
	}
	return nil, nil
}

// Sign calculates an ECDSA signature.
//
// This function is susceptible to chosen plaintext attacks that can leak
// information about the private key that is used for signing. Callers must
// be aware that the given digest cannot be chosen by an adversery. Common
// solution is to hash any input before calculating the signature.
//
// The produced signature is in the [R || S || V] format where V is 0 or 1.
func Sign(digestHash []byte, prv *ecdsa.PrivateKey) (sig []byte, err error) {
	if CryptoType == CRYPTO_ECC_SH3_AES {
		if len(digestHash) != DigestLength {
			return nil, fmt.Errorf("hash is required to be exactly %d bytes (%d)", DigestLength, len(digestHash))
		}
		seckey := math.PaddedBigBytes(prv.D, prv.Params().BitSize/8)
		defer zeroBytes(seckey)
		//return secp256k1.Sign(digestHash, seckey)
		// 拓展签名
		smsign, err := secp256k1.Sign(digestHash, seckey)
		if err != nil {
			return nil, err
		}

		var pad [32]byte
		smsign = append(smsign, pad[:]...)
		return smsign, nil
	}
	if CryptoType == CRYPTO_SM2_SM3_SM4 {
		smsign, e, err := sm2.Sign(sm2.ToSm2privatekey(prv), nil, digestHash)
		if err != nil {
			return nil, err
		}
		var pad [32]byte
		buf := e.Bytes()
		copy(pad[32-len(buf):], buf)
		smsign = append(smsign, pad[:]...)
		return smsign, nil
	}
	return nil, nil
}

// VerifySignature checks that the given public key created signature over digest.
// The public key should be in compressed (33 bytes) or uncompressed (65 bytes) format.
// The signature should have the 64 byte [R || S] format.
func VerifySignature(pubkey, digestHash, signature []byte) bool {
	if CryptoType == CRYPTO_ECC_SH3_AES {
		return secp256k1.VerifySignature(pubkey, digestHash, signature[:64])
		//return secp256k1.VerifySignature(pubkey, digestHash, signature)
	}
	if CryptoType == CRYPTO_SM2_SM3_SM4 {
		if len(pubkey) == 33 {
			smpub, err := DecompressPubkey(pubkey)
			if err != nil {
				return false
			}
			return sm2.Verify(sm2.ToSm2Publickey(smpub), nil, digestHash, signature)
		}
		smpub, err := UnmarshalPubkey(pubkey)
		if err != nil {
			return false
		}
		return sm2.Verify(sm2.ToSm2Publickey(smpub), nil, digestHash, signature)
	}
	return false
}

// DecompressPubkey parses a public key in the 33-byte compressed format.
func DecompressPubkey(pubkey []byte) (*ecdsa.PublicKey, error) {
	if CryptoType == CRYPTO_ECC_SH3_AES {
		x, y := secp256k1.DecompressPubkey(pubkey)
		if x == nil {
			return nil, fmt.Errorf("invalid public key")
		}
		return &ecdsa.PublicKey{X: x, Y: y, Curve: S256()}, nil
	}
	if CryptoType == CRYPTO_SM2_SM3_SM4 {
		return sm2.ToECDSAPublickey(sm2.Decompress(pubkey)), nil
	}
	return nil, nil
}

// CompressPubkey encodes a public key to the 33-byte compressed format.
func CompressPubkey(pubkey *ecdsa.PublicKey) []byte {
	if CryptoType == CRYPTO_ECC_SH3_AES {
		return secp256k1.CompressPubkey(pubkey.X, pubkey.Y)
	}
	if CryptoType == CRYPTO_SM2_SM3_SM4 {
		return sm2.Compress(sm2.ToSm2Publickey(pubkey))

	}
	return nil
}

// S256 returns an instance of the secp256k1 curve.
func S256() elliptic.Curve {
	return secp256k1.S256()
}
