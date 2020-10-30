package types

import (
	"github.com/ethereum/go-ethereum/common"
)

// author : zr
// CM 承诺结构，包含承诺字段以及判断该承诺是否被使用过的spent字段，true表示已使用
type CM struct {
	Cm    uint64
	Spent bool
}

func NewDefaultCM(Cm uint64) *CM {
	return &CM{
		Cm:    Cm,
		Spent: false,
	}
}

func NewCM(Cm uint64, Spent bool) *CM {
	return &CM{
		Cm:    Cm,
		Spent: Spent,
	}
}

func (cm *CM) Hash() common.Hash {
	return rlpHash(cm.Cm)
}
