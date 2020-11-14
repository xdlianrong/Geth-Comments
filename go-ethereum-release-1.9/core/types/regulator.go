package types

import (
	"math/big"
)

type Regulator struct {
	PubK PubKey
	IP   string
	Port int
}

type PubKey struct {
	G1 *big.Int
	G2 *big.Int
	P  *big.Int
	H  *big.Int
}
type Exchange struct {
	PubKey PubKey
	URL string
}
