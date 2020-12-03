// Copyright 2018 The go-ethereum Authors
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

package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type ValidationInfo struct {
	Typ     string `json:"type"`
	Message string `json:"message"`
}
type ValidationMessages struct {
	Messages []ValidationInfo
}

const (
	WARN = "WARNING"
	CRIT = "CRITICAL"
	INFO = "Info"
)

func (vs *ValidationMessages) Crit(msg string) {
	vs.Messages = append(vs.Messages, ValidationInfo{CRIT, msg})
}
func (vs *ValidationMessages) Warn(msg string) {
	vs.Messages = append(vs.Messages, ValidationInfo{WARN, msg})
}
func (vs *ValidationMessages) Info(msg string) {
	vs.Messages = append(vs.Messages, ValidationInfo{INFO, msg})
}

/// getWarnings returns an error with all messages of type WARN of above, or nil if no warnings were present
func (v *ValidationMessages) getWarnings() error {
	var messages []string
	for _, msg := range v.Messages {
		if msg.Typ == WARN || msg.Typ == CRIT {
			messages = append(messages, msg.Message)
		}
	}
	if len(messages) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(messages, ","))
	}
	return nil
}

// SendTxArgs represents the arguments to submit a transaction
type SendTxArgs struct {
	From     common.MixedcaseAddress  `json:"from"`
	To       *common.MixedcaseAddress `json:"to"`
	Gas      hexutil.Uint64           `json:"gas"`
	GasPrice hexutil.Big              `json:"gasPrice"`
	Value    hexutil.Big              `json:"value"`
	Nonce    hexutil.Uint64           `json:"nonce"`
	// We accept "data" and "input" for backwards-compatibility reasons.
	Data     *hexutil.Bytes  `json:"data"`
	Input    *hexutil.Bytes  `json:"input,omitempty"`
	ID       *hexutil.Uint64 `json:"ID"`
	ErpkC1   *hexutil.Bytes  `json:"erpkc1"`
	ErpkC2   *hexutil.Bytes  `json:"erpkc2"`
	EspkC1   *hexutil.Bytes  `json:"espkc1"`
	EspkC2   *hexutil.Bytes  `json:"espkc2"`
	CMRpk    *hexutil.Bytes  `json:"cmrpk"`
	CMSpk    *hexutil.Bytes  `json:"cmspk"`
	ErpkEPs0 *hexutil.Bytes  `json:"erpkeps0"`
	ErpkEPs1 *hexutil.Bytes  `json:"erpkeps1"`
	ErpkEPs2 *hexutil.Bytes  `json:"erpkeps2"`
	ErpkEPs3 *hexutil.Bytes  `json:"erpkeps3"`
	ErpkEPt  *hexutil.Bytes  `json:"erpkept"`
	EspkEPs0 *hexutil.Bytes  `json:"espkeps0"`
	EspkEPs1 *hexutil.Bytes  `json:"espkeps1"`
	EspkEPs2 *hexutil.Bytes  `json:"espkeps2"`
	EspkEPs3 *hexutil.Bytes  `json:"espkeps3"`
	EspkEPt  *hexutil.Bytes  `json:"espkept"`
	EvSC1    *hexutil.Bytes  `json:"evsc1"`
	EvSC2    *hexutil.Bytes  `json:"evsc2"`
	EvRC1    *hexutil.Bytes  `json:"evrc1"`
	EvRC2    *hexutil.Bytes  `json:"evrc2"`
	CmS      *hexutil.Bytes  `json:"cms"`
	CmR      *hexutil.Bytes  `json:"cmr"`
	CMsFPC   *hexutil.Bytes  `json:"cmsfpc"`
	CMsFPZ1  *hexutil.Bytes  `json:"cmsfpz1"`
	CMsFPZ2  *hexutil.Bytes  `json:"cmsfpz2"`
	CMrFPC   *hexutil.Bytes  `json:"cmrfpc"`
	CMrFPZ1  *hexutil.Bytes  `json:"cmrfpz1"`
	CMrFPZ2  *hexutil.Bytes  `json:"cmrfpz2"`
	EvsBsC1  *hexutil.Bytes  `json:"evsbsc1"`
	EvsBsC2  *hexutil.Bytes  `json:"evsbsc2"`
	EvOC1    *hexutil.Bytes  `json:"evoc1"`
	EvOC2    *hexutil.Bytes  `json:"evoc2"`
	CmO      *hexutil.Bytes  `json:"cmo"`
	EvOEPs0  *hexutil.Bytes  `json:"evoeps0"`
	EvOEPs1  *hexutil.Bytes  `json:"evoeps1"`
	EvOEPs2  *hexutil.Bytes  `json:"evoeps2"`
	EvOEPs3  *hexutil.Bytes  `json:"evoeps3"`
	EvOEPt   *hexutil.Bytes  `json:"evoept"`
	BPC      *hexutil.Bytes  `json:"bpc"`
	BPRV     *hexutil.Bytes  `json:"bprv"`
	BPRR     *hexutil.Bytes  `json:"bprr"`
	BPSV     *hexutil.Bytes  `json:"bpsv"`
	BPSR     *hexutil.Bytes  `json:"bpsr"`
	BPSOr    *hexutil.Bytes  `json:"bpsor"`
	EpkrC1   *hexutil.Bytes  `json:"epkrc1"`
	EpkrC2   *hexutil.Bytes  `json:"epkrc2"`
	EpkpC1   *hexutil.Bytes  `json:"epkpc1"`
	EpkpC2   *hexutil.Bytes  `json:"epkpc2"`
	SigM     *hexutil.Bytes  `json:"sigm"`
	SigMHash *hexutil.Bytes  `json:"sigmhash"`
	SigR     *hexutil.Bytes  `json:"sigr"`
	SigS     *hexutil.Bytes  `json:"sigs"`
	CmV      *hexutil.Bytes  `json:"cmv"`
	CmSRC1   *hexutil.Bytes  `json:"cmsrc1"`
	CmSRC2   *hexutil.Bytes  `json:" cmsrc2"`
	CmRRC1   *hexutil.Bytes  `json:" cmrrc1"`
	CmRRC2   *hexutil.Bytes  `json:" cmrrc2"`
}

func (args SendTxArgs) String() string {
	s, err := json.Marshal(args)
	if err == nil {
		return string(s)
	}
	return err.Error()
}

func (args *SendTxArgs) toTransaction() *types.Transaction {
	var input []byte
	if args.Data != nil {
		input = *args.Data
	} else if args.Input != nil {
		input = *args.Input
	}
	if args.To == nil {
		return types.NewContractCreation(uint64(args.Nonce), (*big.Int)(&args.Value), uint64(args.Gas), (*big.Int)(&args.GasPrice), input, uint64(*args.ID), args.ErpkC1, args.ErpkC2, args.EspkC1, args.EspkC2, args.CMRpk, args.CMSpk, args.ErpkEPs0, args.ErpkEPs1, args.ErpkEPs2, args.ErpkEPs3, args.ErpkEPt, args.EspkEPs0, args.EspkEPs1, args.EspkEPs2, args.EspkEPs3, args.EspkEPt, args.EvSC1, args.EvSC2, args.EvRC1, args.EvRC2, args.CmS, args.CmR, args.CMsFPC, args.CMsFPZ1, args.CMsFPZ2, args.CMrFPC, args.CMrFPZ1, args.CMrFPZ2, args.EvsBsC1, args.EvsBsC2, args.EvOC1, args.EvOC2, args.CmO, args.EvOEPs0, args.EvOEPs1, args.EvOEPs2, args.EvOEPs3, args.EvOEPt, args.BPC, args.BPRV, args.BPRR, args.BPSV, args.BPSR, args.BPSOr, args.EpkrC1, args.EpkrC2, args.EpkpC1, args.EpkpC2, args.SigM, args.SigMHash, args.SigR, args.SigS, args.CmV, args.CmSRC1, args.CmSRC2, args.CmRRC1, args.CmRRC2)
	}
	return types.NewTransaction(uint64(args.Nonce), args.To.Address(), (*big.Int)(&args.Value), (uint64)(args.Gas), (*big.Int)(&args.GasPrice), input, uint64(*args.ID), args.ErpkC1, args.ErpkC2, args.EspkC1, args.EspkC2, args.CMRpk, args.CMSpk, args.ErpkEPs0, args.ErpkEPs1, args.ErpkEPs2, args.ErpkEPs3, args.ErpkEPt, args.EspkEPs0, args.EspkEPs1, args.EspkEPs2, args.EspkEPs3, args.EspkEPt, args.EvSC1, args.EvSC2, args.EvRC1, args.EvRC2, args.CmS, args.CmR, args.CMsFPC, args.CMsFPZ1, args.CMsFPZ2, args.CMrFPC, args.CMrFPZ1, args.CMrFPZ2, args.EvsBsC1, args.EvsBsC2, args.EvOC1, args.EvOC2, args.CmO, args.EvOEPs0, args.EvOEPs1, args.EvOEPs2, args.EvOEPs3, args.EvOEPt, args.BPC, args.BPRV, args.BPRR, args.BPSV, args.BPSR, args.BPSOr, args.EpkrC1, args.EpkrC2, args.EpkpC1, args.EpkpC2, args.SigM, args.SigMHash, args.SigR, args.SigS, args.CmV, args.CmSRC1, args.CmSRC2, args.CmRRC1, args.CmRRC2)
}
