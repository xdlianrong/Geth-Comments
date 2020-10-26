package types

import "math/big"

type Regulator struct {
	PubK pubKey
	IP   string
	Port int
}

type pubKey struct {
	G1 *big.Int
	G2 *big.Int
	P  *big.Int
	H  *big.Int
}
