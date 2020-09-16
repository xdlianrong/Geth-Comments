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
	Data  *hexutil.Bytes  `json:"data"`
	Input *hexutil.Bytes  `json:"input,omitempty"`
	SnO   *hexutil.Uint64 `json:"SnO"`
	rR1   *hexutil.Uint64 `json:"rR1"`
	CmSpk *hexutil.Uint64 `json:"CmSpk"`
	CmRpk *hexutil.Uint64 `json:"CmRpk"`
	CmO   *hexutil.Uint64 `json:"CmO"`
	CmS   *hexutil.Uint64 `json:"CmS"`
	CmR   *hexutil.Uint64 `json:"CmR"`
	EvR   *hexutil.Uint64 `json:"EvR"`
	EvR0  *hexutil.Uint64 `json:"EvR0"`
	EvR_  *hexutil.Uint64 `json:"EvR_"`
	EvR_0 *hexutil.Uint64 `json:"EvR_0"`
	pi    *hexutil.Uint64 `json:"pi"`
	ID    *hexutil.Uint64 `json:"ID"`
	Sig   *hexutil.Uint64 `json:"Sig"`
	CmV   *hexutil.Uint64 `json:"CmV"`
	EpkV  *hexutil.Uint64 `json:"EpkV"`
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
		return types.NewContractCreation(uint64(args.Nonce), (*big.Int)(&args.Value), uint64(args.Gas), (*big.Int)(&args.GasPrice), input, uint64(*args.SnO), uint64(*args.rR1), uint64(*args.CmSpk), uint64(*args.CmRpk), uint64(*args.CmO), uint64(*args.CmS), uint64(*args.CmR), uint64(*args.EvR), uint64(*args.EvR0), uint64(*args.EvR_), uint64(*args.EvR_0), uint64(*args.pi), uint64(*args.ID), uint64(*args.Sig), uint64(*args.CmV), uint64(*args.EpkV))
	}
	return types.NewTransaction(uint64(args.Nonce), args.To.Address(), (*big.Int)(&args.Value), (uint64)(args.Gas), (*big.Int)(&args.GasPrice), input, uint64(*args.SnO), uint64(*args.rR1), uint64(*args.CmSpk), uint64(*args.CmRpk), uint64(*args.CmO), uint64(*args.CmS), uint64(*args.CmR), uint64(*args.EvR), uint64(*args.EvR0), uint64(*args.EvR_), uint64(*args.EvR_0), uint64(*args.pi), uint64(*args.ID), uint64(*args.Sig), uint64(*args.CmV), uint64(*args.EpkV))
}
