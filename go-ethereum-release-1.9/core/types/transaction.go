// Copyright 2014 The go-ethereum Authors
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

package types

import (
	"container/heap"
	"errors"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate gencodec -type txdata -field-override txdataMarshaling -out gen_tx_json.go

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)

type Transaction struct {
	data txdata //一个不限制大小的字节数组，用来指定消息调用的输入数据
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type txdata struct {
	AccountNonce uint64          `json:"nonce"         gencodec:"required"` //由交易发送者发出的的交易的数量，由 Tn 表示
	Price        *big.Int        `json:"gasPrice"      gencodec:"required"` //为执行这个交易所需要进行的计算步骤消 耗的每单位 gas 的价格，以 Wei 为单位，由 Tp 表 示。
	GasLimit     uint64          `json:"gas"           gencodec:"required"` //用于执行这个交易的最大 gas 数量。这个值须在交易开始前设置，且设定后不能再增加，由Tg 表示。
	Recipient    *common.Address `json:"to"            rlp:"nil"`           // nil means contract creation 160 位的消息调用接收者地址；对与合约创建交易，用 ∅ 表示 B0 的唯一成员。此字段由 Tt 表示
	Amount       *big.Int        `json:"value"         gencodec:"required"` //转移到接收者账户的 Wei 的数量；对于合约 创建，则代表给新建合约地址的初始捐款。由 Tv 表示。
	Payload      []byte          `json:"input"         gencodec:"required"` //如果目标账户包含代码，该代码会执行，payload就是输入数据。如果目标账户是零账户（账户地址是0），交易将创建一个新合约。这个合约地址不是零地址，而是由合约创建者的地址和该地址发出过的交易数量（被称为nonce）计算得到。创建合约交易的payload被当作EVM字节码执行。执行的输出做为合约代码被永久存储。这意味着，为了创建一个合约，你不需要向合约发送真正的合约代码，而是发送能够返回真正代码的代码。
	ID           uint64          `json:"ID"            gencodec:"required"` //交易标识
	ErpkC1       *hexutil.Bytes  `json:"erpkc1"        gencodec:"required"` //接收方地址公钥加密字段C1
	ErpkC2       *hexutil.Bytes  `json:"erpkc2"        gencodec:"required"` //接收方地址公钥加密字段C2
	EspkC1       *hexutil.Bytes  `json:"espkc1"        gencodec:"required"` //发送方地址公钥加密字段C1
	EspkC2       *hexutil.Bytes  `json:"espkc2"        gencodec:"required"` //发送方地址公钥加密字段C2
	CMRpk        *hexutil.Bytes  `json:"cmrpk"         gencodec:"required"` //接收方地址公钥承诺
	CMSpk        *hexutil.Bytes  `json:"cmspk"         gencodec:"required"` //发送方地址公钥承诺
	ErpkEPs0     *hexutil.Bytes  `json:"erpkeps0"      gencodec:"required"` //接收方地址公钥相等证明字段s0
	ErpkEPs1     *hexutil.Bytes  `json:"erpkeps1"      gencodec:"required"` //接收方地址公钥相等证明字段s1
	ErpkEPs2     *hexutil.Bytes  `json:"erpkeps2"      gencodec:"required"` //接收方地址公钥相等证明字段s2
	ErpkEPs3     *hexutil.Bytes  `json:"erpkeps3"      gencodec:"required"` //接收方地址公钥相等证明字段s3
	ErpkEPt      *hexutil.Bytes  `json:"erpkept"       gencodec:"required"` //接收方地址公钥相等证明字段t
	EspkEPs0     *hexutil.Bytes  `json:"espkeps0"      gencodec:"required"` //发送方地址公钥相等证明字段s0
	EspkEPs1     *hexutil.Bytes  `json:"espkeps1"      gencodec:"required"` //发送方地址公钥相等证明字段s1
	EspkEPs2     *hexutil.Bytes  `json:"espkeps2"      gencodec:"required"` //发送方地址公钥相等证明字段s2
	EspkEPs3     *hexutil.Bytes  `json:"espkeps3"      gencodec:"required"` //发送方地址公钥相等证明字段s3
	EspkEPt      *hexutil.Bytes  `json:"espkept"       gencodec:"required"` //发送方地址公钥相等证明字段t
	EvSC1        *hexutil.Bytes  `json:"evsc1"         gencodec:"required"` //发送金额加密字段C1
	EvSC2        *hexutil.Bytes  `json:"evsc2"         gencodec:"required"` //发送金额加密字段C2
	EvRC1        *hexutil.Bytes  `json:"evrc1"         gencodec:"required"` //接收金额加密字段C1
	EvRC2        *hexutil.Bytes  `json:"evrc2"         gencodec:"required"` //接收金额加密字段C2
	CmS          *hexutil.Bytes  `json:"cms"           gencodec:"required"` //发送金额承诺
	CmR          *hexutil.Bytes  `json:"cmr"           gencodec:"required"` //返还（找零）金额承诺
	CMsFPC       *hexutil.Bytes  `json:"cmsfpc"        gencodec:"required"` //发送金额承诺格式证明字段C
	CMsFPZ1      *hexutil.Bytes  `json:"cmsfpz1"       gencodec:"required"` //发送金额承诺格式证明字段Z1
	CMsFPZ2      *hexutil.Bytes  `json:"cmsfpz2"       gencodec:"required"` //发送金额承诺格式证明字段Z2
	CMrFPC       *hexutil.Bytes  `json:"cmrfpc"        gencodec:"required"` //接收金额承诺格式证明字段C
	CMrFPZ1      *hexutil.Bytes  `json:"cmrfpz1"       gencodec:"required"` //接收金额承诺格式证明字段Z1
	CMrFPZ2      *hexutil.Bytes  `json:"cmrfpz2"       gencodec:"required"` //接收金额承诺格式证明字段Z2
	EvsBsC1      *hexutil.Bytes  `json:"evsbsc1"       gencodec:"required"` //接收方公钥加密的发送金额字段C1
	EvsBsC2      *hexutil.Bytes  `json:"evsbsc2"       gencodec:"required"` //接收方公钥加密的发送金额字段C2
	EvOC1        *hexutil.Bytes  `json:"evoc1"         gencodec:"required"` //被花费承诺加密字段C1
	EvOC2        *hexutil.Bytes  `json:"evoc2"         gencodec:"required"` //被花费承诺加密字段C2
	CmO          *hexutil.Bytes  `json:"cmo"           gencodec:"required"` //被花费承诺
	EvOEPs0      *hexutil.Bytes  `json:"evoeps0"       gencodec:"required"` //被花费承诺相等证明字段s0
	EvOEPs1      *hexutil.Bytes  `json:"evoeps1"       gencodec:"required"` //被花费承诺相等证明字段s1
	EvOEPs2      *hexutil.Bytes  `json:"evoeps2"       gencodec:"required"` //被花费承诺相等证明字段s2
	EvOEPs3      *hexutil.Bytes  `json:"evoeps3"       gencodec:"required"` //被花费承诺相等证明字段s3
	EvOEPt       *hexutil.Bytes  `json:"evoept"        gencodec:"required"` //被花费承诺相等证明字段t
	BPC          *hexutil.Bytes  `json:"bpc"           gencodec:"required"` //会计平衡证明字段C
	BPRV         *hexutil.Bytes  `json:"bprv"          gencodec:"required"` //会计平衡证明字段RV
	BPRR         *hexutil.Bytes  `json:"bprr"          gencodec:"required"` //会计平衡证明字段RR
	BPSV         *hexutil.Bytes  `json:"bpsv"          gencodec:"required"` //会计平衡证明字段SV
	BPSR         *hexutil.Bytes  `json:"bpsr"          gencodec:"required"` //会计平衡证明字段SR
	BPSOr        *hexutil.Bytes  `json:"bpsor"         gencodec:"required"` //会计平衡证明字段SOr
	EpkrC1       *hexutil.Bytes  `json:"epkrc1"        gencodec:"required"` //用户公钥加密随机数r后的字段C1
	EpkrC2       *hexutil.Bytes  `json:"epkrc2"        gencodec:"required"` //用户公钥加密随机数r后的字段C2
	EpkpC1       *hexutil.Bytes  `json:"epkpc1"        gencodec:"required"` //利用监管者公钥加密publickey+amount的结果C1
	EpkpC2       *hexutil.Bytes  `json:"epkpc2"        gencodec:"required"` //利用监管者公钥加密publickey+amount的结果C2
	SigM         *hexutil.Bytes  `json:"sigm"          gencodec:"required"` //发行者签名的明文信息
	SigMHash     *hexutil.Bytes  `json:"sigmhash"      gencodec:"required"` //发行者签名明文的hash值
	SigR         *hexutil.Bytes  `json:"sigr"          gencodec:"required"` //发行者签名的密文r
	SigS         *hexutil.Bytes  `json:"sigs"          gencodec:"required"` //发行者签名的密文s
	CmV          *hexutil.Bytes  `json:"cmv"           gencodec:"required"` //监管者公钥生成的本次购币的承诺
	// Signature values
	V *big.Int `json:"v" gencodec:"required"` //v, r, s: 与交易签名相符的若干数值，用于确定交易的发送者，由 Tw，Tr 和 Ts 表示。
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"hash" rlp:"-"`
}

type txdataMarshaling struct {
	AccountNonce hexutil.Uint64
	Price        *hexutil.Big
	GasLimit     hexutil.Uint64
	Amount       *hexutil.Big
	Payload      hexutil.Bytes
	V            *hexutil.Big
	R            *hexutil.Big
	S            *hexutil.Big
}

func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte,
	ID uint64, ErpkC1 *hexutil.Bytes, ErpkC2 *hexutil.Bytes, EspkC1 *hexutil.Bytes, EspkC2 *hexutil.Bytes,
	CMRpk *hexutil.Bytes, CMSpk *hexutil.Bytes, ErpkEPs0 *hexutil.Bytes, ErpkEPs1 *hexutil.Bytes, ErpkEPs2 *hexutil.Bytes,
	ErpkEPs3 *hexutil.Bytes, ErpkEPt *hexutil.Bytes, EspkEPs0 *hexutil.Bytes, EspkEPs1 *hexutil.Bytes, EspkEPs2 *hexutil.Bytes,
	EspkEPs3 *hexutil.Bytes, EspkEPt *hexutil.Bytes, EvSC1 *hexutil.Bytes, EvSC2 *hexutil.Bytes, EvRC1 *hexutil.Bytes,
	EvRC2 *hexutil.Bytes, CmS *hexutil.Bytes, CmR *hexutil.Bytes, CMsFPC *hexutil.Bytes, CMsFPZ1 *hexutil.Bytes,
	CMsFPZ2 *hexutil.Bytes, CMrFPC *hexutil.Bytes, CMrFPZ1 *hexutil.Bytes, CMrFPZ2 *hexutil.Bytes, EvsBsC1 *hexutil.Bytes,
	EvsBsC2 *hexutil.Bytes, EvOC1 *hexutil.Bytes, EvOC2 *hexutil.Bytes, CmO *hexutil.Bytes, EvOEPs0 *hexutil.Bytes,
	EvOEPs1 *hexutil.Bytes, EvOEPs2 *hexutil.Bytes, EvOEPs3 *hexutil.Bytes, EvOEPt *hexutil.Bytes, BPC *hexutil.Bytes,
	BPRV *hexutil.Bytes, BPRR *hexutil.Bytes, BPSV *hexutil.Bytes, BPSR *hexutil.Bytes, BPSOr *hexutil.Bytes,
	EpkrC1 *hexutil.Bytes, EpkrC2 *hexutil.Bytes, EpkpC1 *hexutil.Bytes, EpkpC2 *hexutil.Bytes, SigM *hexutil.Bytes,
	SigMHash *hexutil.Bytes, SigR *hexutil.Bytes, SigS *hexutil.Bytes, CmV *hexutil.Bytes) *Transaction {
	return newTransaction(nonce, &to, amount, gasLimit, gasPrice, data, ID, ErpkC1, ErpkC2, EspkC1, EspkC2, CMRpk, CMSpk, ErpkEPs0, ErpkEPs1, ErpkEPs2, ErpkEPs3, ErpkEPt, EspkEPs0, EspkEPs1, EspkEPs2, EspkEPs3, EspkEPt, EvSC1, EvSC2, EvRC1, EvRC2, CmS, CmR, CMsFPC, CMsFPZ1, CMsFPZ2, CMrFPC, CMrFPZ1, CMrFPZ2, EvsBsC1, EvsBsC2, EvOC1, EvOC2, CmO, EvOEPs0, EvOEPs1, EvOEPs2, EvOEPs3, EvOEPt, BPC, BPRV, BPRR, BPSV, BPSR, BPSOr, EpkrC1, EpkrC2, EpkpC1, EpkpC2, SigM, SigMHash, SigR, SigS, CmV)
}

func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, ID uint64, ErpkC1 *hexutil.Bytes, ErpkC2 *hexutil.Bytes, EspkC1 *hexutil.Bytes, EspkC2 *hexutil.Bytes, CMRpk *hexutil.Bytes, CMSpk *hexutil.Bytes, ErpkEPs0 *hexutil.Bytes, ErpkEPs1 *hexutil.Bytes, ErpkEPs2 *hexutil.Bytes, ErpkEPs3 *hexutil.Bytes, ErpkEPt *hexutil.Bytes, EspkEPs0 *hexutil.Bytes, EspkEPs1 *hexutil.Bytes, EspkEPs2 *hexutil.Bytes, EspkEPs3 *hexutil.Bytes, EspkEPt *hexutil.Bytes, EvSC1 *hexutil.Bytes, EvSC2 *hexutil.Bytes, EvRC1 *hexutil.Bytes, EvRC2 *hexutil.Bytes, CmS *hexutil.Bytes, CmR *hexutil.Bytes, CMsFPC *hexutil.Bytes, CMsFPZ1 *hexutil.Bytes, CMsFPZ2 *hexutil.Bytes, CMrFPC *hexutil.Bytes, CMrFPZ1 *hexutil.Bytes, CMrFPZ2 *hexutil.Bytes, EvsBsC1 *hexutil.Bytes, EvsBsC2 *hexutil.Bytes, EvOC1 *hexutil.Bytes, EvOC2 *hexutil.Bytes, CmO *hexutil.Bytes, EvOEPs0 *hexutil.Bytes, EvOEPs1 *hexutil.Bytes, EvOEPs2 *hexutil.Bytes, EvOEPs3 *hexutil.Bytes, EvOEPt *hexutil.Bytes, BPC *hexutil.Bytes, BPRV *hexutil.Bytes, BPRR *hexutil.Bytes, BPSV *hexutil.Bytes, BPSR *hexutil.Bytes, BPSOr *hexutil.Bytes, EpkrC1 *hexutil.Bytes, EpkrC2 *hexutil.Bytes, EpkpC1 *hexutil.Bytes, EpkpC2 *hexutil.Bytes, SigM *hexutil.Bytes,
	SigMHash *hexutil.Bytes, SigR *hexutil.Bytes, SigS *hexutil.Bytes, CmV *hexutil.Bytes) *Transaction {
	return newTransaction(nonce, nil, amount, gasLimit, gasPrice, data, ID, ErpkC1, ErpkC2, EspkC1, EspkC2, CMRpk, CMSpk, ErpkEPs0, ErpkEPs1, ErpkEPs2, ErpkEPs3, ErpkEPt, EspkEPs0, EspkEPs1, EspkEPs2, EspkEPs3, EspkEPt, EvSC1, EvSC2, EvRC1, EvRC2, CmS, CmR, CMsFPC, CMsFPZ1, CMsFPZ2, CMrFPC, CMrFPZ1, CMrFPZ2, EvsBsC1, EvsBsC2, EvOC1, EvOC2, CmO, EvOEPs0, EvOEPs1, EvOEPs2, EvOEPs3, EvOEPt, BPC, BPRV, BPRR, BPSV, BPSR, BPSOr, EpkrC1, EpkrC2, EpkpC1, EpkpC2, SigM, SigMHash, SigR, SigS, CmV)
}

func newTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, ID uint64, ErpkC1 *hexutil.Bytes, ErpkC2 *hexutil.Bytes, EspkC1 *hexutil.Bytes, EspkC2 *hexutil.Bytes, CMRpk *hexutil.Bytes, CMSpk *hexutil.Bytes, ErpkEPs0 *hexutil.Bytes, ErpkEPs1 *hexutil.Bytes, ErpkEPs2 *hexutil.Bytes, ErpkEPs3 *hexutil.Bytes, ErpkEPt *hexutil.Bytes, EspkEPs0 *hexutil.Bytes, EspkEPs1 *hexutil.Bytes, EspkEPs2 *hexutil.Bytes, EspkEPs3 *hexutil.Bytes, EspkEPt *hexutil.Bytes, EvSC1 *hexutil.Bytes, EvSC2 *hexutil.Bytes, EvRC1 *hexutil.Bytes, EvRC2 *hexutil.Bytes, CmS *hexutil.Bytes, CmR *hexutil.Bytes, CMsFPC *hexutil.Bytes, CMsFPZ1 *hexutil.Bytes, CMsFPZ2 *hexutil.Bytes, CMrFPC *hexutil.Bytes, CMrFPZ1 *hexutil.Bytes, CMrFPZ2 *hexutil.Bytes, EvsBsC1 *hexutil.Bytes, EvsBsC2 *hexutil.Bytes, EvOC1 *hexutil.Bytes, EvOC2 *hexutil.Bytes, CmO *hexutil.Bytes, EvOEPs0 *hexutil.Bytes, EvOEPs1 *hexutil.Bytes, EvOEPs2 *hexutil.Bytes, EvOEPs3 *hexutil.Bytes, EvOEPt *hexutil.Bytes, BPC *hexutil.Bytes, BPRV *hexutil.Bytes, BPRR *hexutil.Bytes, BPSV *hexutil.Bytes, BPSR *hexutil.Bytes, BPSOr *hexutil.Bytes,
	EpkrC1 *hexutil.Bytes, EpkrC2 *hexutil.Bytes, EpkpC1 *hexutil.Bytes, EpkpC2 *hexutil.Bytes, SigM *hexutil.Bytes,
	SigMHash *hexutil.Bytes, SigR *hexutil.Bytes, SigS *hexutil.Bytes, CmV *hexutil.Bytes) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := txdata{
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
		ID:           ID,
		ErpkC1:       ErpkC1,
		ErpkC2:       ErpkC2,
		EspkC1:       EspkC1,
		EspkC2:       EspkC2,
		CMRpk:        CMRpk,
		CMSpk:        CMSpk,
		ErpkEPs0:     ErpkEPs0,
		ErpkEPs1:     ErpkEPs1,
		ErpkEPs2:     ErpkEPs2,
		ErpkEPs3:     ErpkEPs3,
		ErpkEPt:      ErpkEPt,
		EspkEPs0:     EspkEPs0,
		EspkEPs1:     EspkEPs1,
		EspkEPs2:     EspkEPs2,
		EspkEPs3:     EspkEPs3,
		EspkEPt:      EspkEPt,
		EvSC1:        EvSC1,
		EvSC2:        EvSC2,
		EvRC1:        EvRC1,
		EvRC2:        EvRC2,
		CmS:          CmS,
		CmR:          CmR,
		CMsFPC:       CMsFPC,
		CMsFPZ1:      CMsFPZ1,
		CMsFPZ2:      CMsFPZ2,
		CMrFPC:       CMrFPC,
		CMrFPZ1:      CMrFPZ1,
		CMrFPZ2:      CMrFPZ2,
		EvsBsC1:      EvsBsC1,
		EvsBsC2:      EvsBsC2,
		EvOC1:        EvOC1,
		EvOC2:        EvOC2,
		CmO:          CmO,
		EvOEPs0:      EvOEPs0,
		EvOEPs1:      EvOEPs1,
		EvOEPs2:      EvOEPs2,
		EvOEPs3:      EvOEPs3,
		EvOEPt:       EvOEPt,
		BPC:          BPC,
		BPRV:         BPRV,
		BPRR:         BPRR,
		BPSV:         BPSV,
		BPSR:         BPSR,
		BPSOr:        BPSOr,
		EpkrC1:       EpkrC1,
		EpkrC2:       EpkrC2,
		EpkpC1:       EpkpC1,
		EpkpC2:       EpkpC2,
		SigM:         SigM,
		SigMHash:     SigMHash,
		SigR:         SigR,
		SigS:         SigS,
		CmV:          CmV,
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}
	return &Transaction{data: d}
}

// ChainId returns which chain id this transaction was signed for (if at all)
func (tx *Transaction) ChainId() *big.Int {
	return deriveChainId(tx.data.V)
}

// Protected returns whether the transaction is protected from replay protection.
func (tx *Transaction) Protected() bool {
	return isProtectedV(tx.data.V)
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28
	}
	// anything not 27 or 28 is considered protected
	return true
}

// EncodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}

	return err
}

// MarshalJSON encodes the web3 RPC transaction format.
func (tx *Transaction) MarshalJSON() ([]byte, error) {
	hash := tx.Hash()
	data := tx.data
	data.Hash = &hash
	return data.MarshalJSON()
}

// UnmarshalJSON decodes the web3 RPC transaction format.
func (tx *Transaction) UnmarshalJSON(input []byte) error {
	var dec txdata
	if err := dec.UnmarshalJSON(input); err != nil {
		return err
	}

	withSignature := dec.V.Sign() != 0 || dec.R.Sign() != 0 || dec.S.Sign() != 0
	if withSignature {
		var V byte
		if isProtectedV(dec.V) {
			chainID := deriveChainId(dec.V).Uint64()
			V = byte(dec.V.Uint64() - 35 - 2*chainID)
		} else {
			V = byte(dec.V.Uint64() - 27)
		}
		if !crypto.ValidateSignatureValues(V, dec.R, dec.S, false) {
			return ErrInvalidSig
		}
	}

	*tx = Transaction{data: dec}
	return nil
}
func (tx *Transaction) Data() []byte             { return common.CopyBytes(tx.data.Payload) }
func (tx *Transaction) Gas() uint64              { return tx.data.GasLimit }
func (tx *Transaction) GasPrice() *big.Int       { return new(big.Int).Set(tx.data.Price) }
func (tx *Transaction) Value() *big.Int          { return new(big.Int).Set(tx.data.Amount) }
func (tx *Transaction) Nonce() uint64            { return tx.data.AccountNonce }
func (tx *Transaction) ID() uint64               { return tx.data.ID }
func (tx *Transaction) ErpkC1() *hexutil.Bytes   { return tx.data.ErpkC1 }
func (tx *Transaction) ErpkC2() *hexutil.Bytes   { return tx.data.ErpkC2 }
func (tx *Transaction) EspkC1() *hexutil.Bytes   { return tx.data.EspkC1 }
func (tx *Transaction) EspkC2() *hexutil.Bytes   { return tx.data.EspkC2 }
func (tx *Transaction) CMRpk() *hexutil.Bytes    { return tx.data.CMRpk }
func (tx *Transaction) CMSpk() *hexutil.Bytes    { return tx.data.CMSpk }
func (tx *Transaction) ErpkEPs0() *hexutil.Bytes { return tx.data.ErpkEPs0 }
func (tx *Transaction) ErpkEPs1() *hexutil.Bytes { return tx.data.ErpkEPs1 }
func (tx *Transaction) ErpkEPs2() *hexutil.Bytes { return tx.data.ErpkEPs2 }
func (tx *Transaction) ErpkEPs3() *hexutil.Bytes { return tx.data.ErpkEPs3 }
func (tx *Transaction) ErpkEPt() *hexutil.Bytes  { return tx.data.ErpkEPt }
func (tx *Transaction) EspkEPs0() *hexutil.Bytes { return tx.data.EspkEPs0 }
func (tx *Transaction) EspkEPs1() *hexutil.Bytes { return tx.data.EspkEPs1 }
func (tx *Transaction) EspkEPs2() *hexutil.Bytes { return tx.data.EspkEPs2 }
func (tx *Transaction) EspkEPs3() *hexutil.Bytes { return tx.data.EspkEPs3 }
func (tx *Transaction) EspkEPt() *hexutil.Bytes  { return tx.data.EspkEPt }
func (tx *Transaction) EvSC1() *hexutil.Bytes    { return tx.data.EvSC1 }
func (tx *Transaction) EvSC2() *hexutil.Bytes    { return tx.data.EvSC2 }
func (tx *Transaction) EvRC1() *hexutil.Bytes    { return tx.data.EvRC1 }
func (tx *Transaction) EvRC2() *hexutil.Bytes    { return tx.data.EvRC2 }
func (tx *Transaction) CmS() *hexutil.Bytes      { return tx.data.CmS }
func (tx *Transaction) CmR() *hexutil.Bytes      { return tx.data.CmR }
func (tx *Transaction) CMsFPC() *hexutil.Bytes   { return tx.data.CMsFPC }
func (tx *Transaction) CMsFPZ1() *hexutil.Bytes  { return tx.data.CMsFPZ1 }
func (tx *Transaction) CMsFPZ2() *hexutil.Bytes  { return tx.data.CMsFPZ2 }
func (tx *Transaction) CMrFPC() *hexutil.Bytes   { return tx.data.CMrFPC }
func (tx *Transaction) CMrFPZ1() *hexutil.Bytes  { return tx.data.CMrFPZ1 }
func (tx *Transaction) CMrFPZ2() *hexutil.Bytes  { return tx.data.CMrFPZ2 }
func (tx *Transaction) EvsBsC1() *hexutil.Bytes  { return tx.data.EvsBsC1 }
func (tx *Transaction) EvsBsC2() *hexutil.Bytes  { return tx.data.EvsBsC2 }
func (tx *Transaction) EvOC1() *hexutil.Bytes    { return tx.data.EvOC1 }
func (tx *Transaction) EvOC2() *hexutil.Bytes    { return tx.data.EvOC2 }
func (tx *Transaction) CmO() *hexutil.Bytes      { return tx.data.CmO }
func (tx *Transaction) EvOEPs0() *hexutil.Bytes  { return tx.data.EvOEPs0 }
func (tx *Transaction) EvOEPs1() *hexutil.Bytes  { return tx.data.EvOEPs1 }
func (tx *Transaction) EvOEPs2() *hexutil.Bytes  { return tx.data.EvOEPs2 }
func (tx *Transaction) EvOEPs3() *hexutil.Bytes  { return tx.data.EvOEPs3 }
func (tx *Transaction) EvOEPt() *hexutil.Bytes   { return tx.data.EvOEPt }
func (tx *Transaction) BPC() *hexutil.Bytes      { return tx.data.BPC }
func (tx *Transaction) BPRV() *hexutil.Bytes     { return tx.data.BPRV }
func (tx *Transaction) BPRR() *hexutil.Bytes     { return tx.data.BPRR }
func (tx *Transaction) BPSV() *hexutil.Bytes     { return tx.data.BPSV }
func (tx *Transaction) BPSR() *hexutil.Bytes     { return tx.data.BPSR }
func (tx *Transaction) BPSOr() *hexutil.Bytes    { return tx.data.BPSOr }
func (tx *Transaction) EpkrC1() *hexutil.Bytes   { return tx.data.EpkrC1 }
func (tx *Transaction) EpkrC2() *hexutil.Bytes   { return tx.data.EpkrC2 }
func (tx *Transaction) EpkpC1() *hexutil.Bytes   { return tx.data.EpkpC1 }
func (tx *Transaction) EpkpC2() *hexutil.Bytes   { return tx.data.EpkpC2 }
func (tx *Transaction) SigM() *hexutil.Bytes     { return tx.data.SigM }
func (tx *Transaction) SigMHash() *hexutil.Bytes { return tx.data.SigMHash }
func (tx *Transaction) SigR() *hexutil.Bytes     { return tx.data.SigR }
func (tx *Transaction) SigS() *hexutil.Bytes     { return tx.data.SigS }
func (tx *Transaction) CmV() *hexutil.Bytes      { return tx.data.CmV }
func (tx *Transaction) CheckNonce() bool         { return true }

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *Transaction) To() *common.Address {
	if tx.data.Recipient == nil {
		return nil
	}
	to := *tx.data.Recipient
	return &to
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
func (tx *Transaction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// AsMessage returns the transaction as a core.Message.
//
// AsMessage requires a signer to derive the sender.
//
// XXX Rename message to something less arbitrary?
func (tx *Transaction) AsMessage(s Signer) (Message, error) {
	msg := Message{
		nonce:      tx.data.AccountNonce,
		gasLimit:   tx.data.GasLimit,
		gasPrice:   new(big.Int).Set(tx.data.Price),
		to:         tx.data.Recipient,
		amount:     tx.data.Amount,
		data:       tx.data.Payload,
		checkNonce: true,
	}

	var err error
	msg.from, err = Sender(s, tx)
	return msg, err
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be in the [R || S || V] format where V is 0 or 1.
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.Price, new(big.Int).SetUint64(tx.data.GasLimit))
	total.Add(total, tx.data.Amount)
	return total
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

// Transactions is a Transaction slice type for basic sorting.
type Transactions []*Transaction

// Len returns the length of s.
func (s Transactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s.
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp.
func (s Transactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

// TxDifference returns a new set which is the difference between a and b.
func TxDifference(a, b Transactions) Transactions {
	keep := make(Transactions, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// TxByNonce implements the sort interface to allow sorting a list of transactions
// by their nonces. This is usually only useful for sorting transactions from a
// single account, otherwise a nonce comparison doesn't make much sense.
type TxByNonce Transactions

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].data.AccountNonce < s[j].data.AccountNonce }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// TxByPrice implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type TxByPrice Transactions

func (s TxByPrice) Len() int           { return len(s) }
func (s TxByPrice) Less(i, j int) bool { return s[i].data.Price.Cmp(s[j].data.Price) > 0 }
func (s TxByPrice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s *TxByPrice) Push(x interface{}) {
	*s = append(*s, x.(*Transaction))
}

func (s *TxByPrice) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

// TransactionsByPriceAndNonce represents a set of transactions that can return
// transactions in a profit-maximizing sorted order, while supporting removing
// entire batches of transactions for non-executable accounts.
type TransactionsByPriceAndNonce struct {
	txs    map[common.Address]Transactions // Per account nonce-sorted list of transactions
	heads  TxByPrice                       // Next transaction for each unique account (price heap)
	signer Signer                          // Signer for the set of transactions
}

// NewTransactionsByPriceAndNonce creates a transaction set that can retrieve
// price sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the caller should not interact any more with
// if after providing it to the constructor.
func NewTransactionsByPriceAndNonce(signer Signer, txs map[common.Address]Transactions) *TransactionsByPriceAndNonce {
	// Initialize a price based heap with the head transactions
	heads := make(TxByPrice, 0, len(txs))
	for from, accTxs := range txs {
		heads = append(heads, accTxs[0])
		// Ensure the sender address is from the signer
		acc, _ := Sender(signer, accTxs[0])
		txs[acc] = accTxs[1:]
		if from != acc {
			delete(txs, from)
		}
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &TransactionsByPriceAndNonce{
		txs:    txs,
		heads:  heads,
		signer: signer,
	}
}

// Peek returns the next transaction by price.
func (t *TransactionsByPriceAndNonce) Peek() *Transaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *TransactionsByPriceAndNonce) Shift() {
	acc, _ := Sender(t.signer, t.heads[0])
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		t.heads[0], t.txs[acc] = txs[0], txs[1:]
		heap.Fix(&t.heads, 0)
	} else {
		heap.Pop(&t.heads)
	}
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *TransactionsByPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
}

// Message is a fully derived transaction and implements core.Message
//
// NOTE: In a future PR this will be removed.
type Message struct {
	to         *common.Address
	from       common.Address
	nonce      uint64
	amount     *big.Int
	gasLimit   uint64
	gasPrice   *big.Int
	data       []byte
	checkNonce bool
}

func NewMessage(from common.Address, to *common.Address, nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, checkNonce bool) Message {
	return Message{
		from:       from,
		to:         to,
		nonce:      nonce,
		amount:     amount,
		gasLimit:   gasLimit,
		gasPrice:   gasPrice,
		data:       data,
		checkNonce: checkNonce,
	}
}

func (m Message) From() common.Address { return m.from }
func (m Message) To() *common.Address  { return m.to }
func (m Message) GasPrice() *big.Int   { return m.gasPrice }
func (m Message) Value() *big.Int      { return m.amount }
func (m Message) Gas() uint64          { return m.gasLimit }
func (m Message) Nonce() uint64        { return m.nonce }
func (m Message) Data() []byte         { return m.data }
func (m Message) CheckNonce() bool     { return m.checkNonce }
