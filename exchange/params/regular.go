package params

import "math/big"


var (
	RegularG1              = big.NewInt(1111)   // The bound divisor of the difficulty, used in the update calculations.
	RegularG2		       = big.NewInt(2222) // Difficulty of the Genesis block.
	RegularBigPrimeNumber  = big.NewInt(0x3906) // The minimum that the difficulty may ever be.
	RegularPublicKey       = big.NewInt(13)     // The decision boundary on the blocktime duration used to determine whether difficulty should go up or not.
)
