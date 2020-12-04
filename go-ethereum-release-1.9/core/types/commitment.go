package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// author : zr
// CM 承诺结构，包含承诺字段以及判断该承诺是否被使用过的spent字段，true表示已使用
type CM struct {
	Cm    *hexutil.Bytes
	Spent bool
	Lock  bool
}

func NewDefaultCM(Cm *hexutil.Bytes) *CM {
	return &CM{
		Cm:    Cm,
		Spent: false,
		Lock:  false,
	}
}

func NewCM(Cm *hexutil.Bytes, Spent bool) *CM {
	return &CM{
		Cm:    Cm,
		Spent: Spent,
		Lock:  false,
	}
}

func (cm *CM) Hash() common.Hash {
	return rlpHash(cm.Cm)
}
