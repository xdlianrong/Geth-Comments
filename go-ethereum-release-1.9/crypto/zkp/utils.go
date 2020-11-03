package zkp

import (
	_ "fmt"
	"math/big"
)

func EncodeCypherText(C CypherText) *big.Int {
	res := make([]byte, 0, len(C.C1)+len(C.C2))
	res = append(res, C.C1...)
	res = append(res, C.C2...)
	// a := fmt.Sprintf("%x", res)
	a := new(big.Int).SetBytes(res)
	return a
}
