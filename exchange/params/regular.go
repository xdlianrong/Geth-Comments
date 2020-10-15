package params

import "math/big"

var (
	RegularG1              = big.NewInt(1111)   // The bound divisor of the difficulty, used in the update calculations.
	RegularG2		       = big.NewInt(2222) // Difficulty of the Genesis block.
	RegularBigPrimeNumber  = big.NewInt(0x39061f1c854fae629b599d29cefe1f12bc4809aa681809bfaaeb1b7087be6fed) // The minimum that the difficulty may ever be.
	RegularPublicKey       = big.NewInt(13)     // The decision boundary on the blocktime duration used to determine whether difficulty should go up or not.
)
