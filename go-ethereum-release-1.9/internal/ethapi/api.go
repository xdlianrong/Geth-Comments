// Copyright 2015 The go-ethereum Authors
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

package ethapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/scwallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/zkp"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/tyler-smith/go-bip39"
)

const (
	defaultGasPrice = params.GWei
)

// PublicEthereumAPI provides an API to access Ethereum related information.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicEthereumAPI struct {
	b Backend
}

// NewPublicEthereumAPI creates a new Ethereum protocol API.
func NewPublicEthereumAPI(b Backend) *PublicEthereumAPI {
	return &PublicEthereumAPI{b}
}

// GasPrice returns a suggestion for a gas price.
func (s *PublicEthereumAPI) GasPrice(ctx context.Context) (*hexutil.Big, error) {
	price, err := s.b.SuggestPrice(ctx)
	return (*hexutil.Big)(price), err
}

// ProtocolVersion returns the current Ethereum protocol version this node supports
func (s *PublicEthereumAPI) ProtocolVersion() hexutil.Uint {
	return hexutil.Uint(s.b.ProtocolVersion())
}

// ProtocolVersion1 returns the current Ethereum protocol version this node supports
func (s *PublicEthereumAPI) GetCMState() map[string]hexutil.Uint {
	valid, invalid := s.b.GetCMState()
	return map[string]hexutil.Uint{
		"valid":   hexutil.Uint(valid),
		"invalid": hexutil.Uint(invalid),
	}
}

// Syncing returns false in case the node is currently not syncing with the network. It can be up to date or has not
// yet received the latest block headers from its pears. In case it is synchronizing:
// - startingBlock: block number this node started to synchronise from
// - currentBlock:  block number this node is currently importing
// - highestBlock:  block number of the highest block header this node has received from peers
// - pulledStates:  number of state entries processed until now
// - knownStates:   number of known state entries that still need to be pulled
func (s *PublicEthereumAPI) Syncing() (interface{}, error) {
	progress := s.b.Downloader().Progress()

	// Return not syncing if the synchronisation already completed
	if progress.CurrentBlock >= progress.HighestBlock {
		return false, nil
	}
	// Otherwise gather the block sync stats
	return map[string]interface{}{
		"startingBlock": hexutil.Uint64(progress.StartingBlock),
		"currentBlock":  hexutil.Uint64(progress.CurrentBlock),
		"highestBlock":  hexutil.Uint64(progress.HighestBlock),
		"pulledStates":  hexutil.Uint64(progress.PulledStates),
		"knownStates":   hexutil.Uint64(progress.KnownStates),
	}, nil
}

// PublicTxPoolAPI offers and API for the transaction pool. It only operates on data that is non confidential.
type PublicTxPoolAPI struct {
	b Backend
}

// NewPublicTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewPublicTxPoolAPI(b Backend) *PublicTxPoolAPI {
	return &PublicTxPoolAPI{b}
}

// Content returns the transactions contained within the transaction pool.
func (s *PublicTxPoolAPI) Content() map[string]map[string]map[string]*RPCTransaction {
	content := map[string]map[string]map[string]*RPCTransaction{
		"pending": make(map[string]map[string]*RPCTransaction),
		"queued":  make(map[string]map[string]*RPCTransaction),
	}
	pending, queue := s.b.TxPoolContent()

	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = newRPCPendingTransaction(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = newRPCPendingTransaction(tx)
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// Status returns the number of pending and queued transaction in the pool.
func (s *PublicTxPoolAPI) Status() map[string]hexutil.Uint {
	pending, queue := s.b.Stats()
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(pending),
		"queued":  hexutil.Uint(queue),
	}
}

// Inspect retrieves the content of the transaction pool and flattens it into an
// easily inspectable list.
func (s *PublicTxPoolAPI) Inspect() map[string]map[string]map[string]string {
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string),
		"queued":  make(map[string]map[string]string),
	}
	pending, queue := s.b.TxPoolContent()

	// Define a formatter to flatten a transaction into a string
	var format = func(tx *types.Transaction) string {
		if to := tx.To(); to != nil {
			return fmt.Sprintf("%s: %v wei + %v gas × %v wei", tx.To().Hex(), tx.Value(), tx.Gas(), tx.GasPrice())
		}
		return fmt.Sprintf("contract creation: %v wei + %v gas × %v wei", tx.Value(), tx.Gas(), tx.GasPrice())
	}
	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// PublicAccountAPI provides an API to access accounts managed by this node.
// It offers only methods that can retrieve accounts.
type PublicAccountAPI struct {
	am *accounts.Manager
}

// NewPublicAccountAPI creates a new PublicAccountAPI.
func NewPublicAccountAPI(am *accounts.Manager) *PublicAccountAPI {
	return &PublicAccountAPI{am: am}
}

// Accounts returns the collection of accounts this node manages
func (s *PublicAccountAPI) Accounts() []common.Address {
	return s.am.Accounts()
}

// PrivateAccountAPI provides an API to access accounts managed by this node.
// It offers methods to create, (un)lock en list accounts. Some methods accept
// passwords and are therefore considered private by default.
type PrivateAccountAPI struct {
	am        *accounts.Manager
	nonceLock *AddrLocker
	b         Backend
}

// NewPrivateAccountAPI create a new PrivateAccountAPI.
func NewPrivateAccountAPI(b Backend, nonceLock *AddrLocker) *PrivateAccountAPI {
	return &PrivateAccountAPI{
		am:        b.AccountManager(),
		nonceLock: nonceLock,
		b:         b,
	}
}

// listAccounts will return a list of addresses for accounts this node manages.
func (s *PrivateAccountAPI) ListAccounts() []common.Address {
	return s.am.Accounts()
}

// rawWallet is a JSON representation of an accounts.Wallet interface, with its
// data contents extracted into plain fields.
type rawWallet struct {
	URL      string             `json:"url"`
	Status   string             `json:"status"`
	Failure  string             `json:"failure,omitempty"`
	Accounts []accounts.Account `json:"accounts,omitempty"`
}

// ListWallets will return a list of wallets this node manages.
func (s *PrivateAccountAPI) ListWallets() []rawWallet {
	wallets := make([]rawWallet, 0) // return [] instead of nil if empty
	for _, wallet := range s.am.Wallets() {
		status, failure := wallet.Status()

		raw := rawWallet{
			URL:      wallet.URL().String(),
			Status:   status,
			Accounts: wallet.Accounts(),
		}
		if failure != nil {
			raw.Failure = failure.Error()
		}
		wallets = append(wallets, raw)
	}
	return wallets
}

// OpenWallet initiates a hardware wallet opening procedure, establishing a USB
// connection and attempting to authenticate via the provided passphrase. Note,
// the method may return an extra challenge requiring a second open (e.g. the
// Trezor PIN matrix challenge).
func (s *PrivateAccountAPI) OpenWallet(url string, passphrase *string) error {
	wallet, err := s.am.Wallet(url)
	if err != nil {
		return err
	}
	pass := ""
	if passphrase != nil {
		pass = *passphrase
	}
	return wallet.Open(pass)
}

// DeriveAccount requests a HD wallet to derive a new account, optionally pinning
// it for later reuse.
func (s *PrivateAccountAPI) DeriveAccount(url string, path string, pin *bool) (accounts.Account, error) {
	wallet, err := s.am.Wallet(url)
	if err != nil {
		return accounts.Account{}, err
	}
	derivPath, err := accounts.ParseDerivationPath(path)
	if err != nil {
		return accounts.Account{}, err
	}
	if pin == nil {
		pin = new(bool)
	}
	return wallet.Derive(derivPath, *pin)
}

// NewAccount will create a new account and returns the address for the new account.
func (s *PrivateAccountAPI) NewAccount(password string) (common.Address, error) {
	acc, err := fetchKeystore(s.am).NewAccount(password)
	if err == nil {
		log.Info("Your new key was generated", "address", acc.Address)
		log.Warn("Please backup your key file!", "path", acc.URL.Path)
		log.Warn("Please remember your password!")
		return acc.Address, nil
	}
	return common.Address{}, err
}

// fetchKeystore retrives the encrypted keystore from the account manager.
func fetchKeystore(am *accounts.Manager) *keystore.KeyStore {
	return am.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
}

// ImportRawKey stores the given hex encoded ECDSA key into the key directory,
// encrypting it with the passphrase.
func (s *PrivateAccountAPI) ImportRawKey(privkey string, password string) (common.Address, error) {
	key, err := crypto.HexToECDSA(privkey)
	if err != nil {
		return common.Address{}, err
	}
	acc, err := fetchKeystore(s.am).ImportECDSA(key, password)
	return acc.Address, err
}

// UnlockAccount will unlock the account associated with the given address with
// the given password for duration seconds. If duration is nil it will use a
// default of 300 seconds. It returns an indication if the account was unlocked.
func (s *PrivateAccountAPI) UnlockAccount(ctx context.Context, addr common.Address, password string, duration *uint64) (bool, error) {
	// When the API is exposed by external RPC(http, ws etc), unless the user
	// explicitly specifies to allow the insecure account unlocking, otherwise
	// it is disabled.
	if s.b.ExtRPCEnabled() && !s.b.AccountManager().Config().InsecureUnlockAllowed {
		return false, errors.New("account unlock with HTTP access is forbidden")
	}

	const max = uint64(time.Duration(math.MaxInt64) / time.Second)
	var d time.Duration
	if duration == nil {
		d = 300 * time.Second
	} else if *duration > max {
		return false, errors.New("unlock duration too large")
	} else {
		d = time.Duration(*duration) * time.Second
	}
	err := fetchKeystore(s.am).TimedUnlock(accounts.Account{Address: addr}, password, d)
	if err != nil {
		log.Warn("Failed account unlock attempt", "address", addr, "err", err)
	}
	return err == nil, err
}

// LockAccount will lock the account associated with the given address when it's unlocked.
func (s *PrivateAccountAPI) LockAccount(addr common.Address) bool {
	return fetchKeystore(s.am).Lock(addr) == nil
}

// signTransaction sets defaults and signs the given transaction
// NOTE: the caller needs to ensure that the nonceLock is held, if applicable,
// and release it after the transaction has been submitted to the tx pool
func (s *PrivateAccountAPI) signTransaction(ctx context.Context, args *SendTxArgs, passwd string) (*types.Transaction, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.From}
	wallet, err := s.am.Find(account)
	if err != nil {
		return nil, err
	}
	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b); err != nil {
		return nil, err
	}
	// Assemble the transaction and sign with the wallet
	tx, err := args.toTransaction()
	if err != nil {
		return nil, err
	}
	return wallet.SignTxWithPassphrase(account, passwd, tx, s.b.ChainConfig().ChainID)
}

// SendTransaction will create a transaction from the given arguments and
// tries to sign it with the key associated with args.To. If the given passwd isn't
// able to decrypt the key it fails.
func (s *PrivateAccountAPI) SendTransaction(ctx context.Context, args SendTxArgs, passwd string) (common.Hash, error) {
	if args.Nonce == nil {
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.From)
		defer s.nonceLock.UnlockAddr(args.From)
	}
	signed, err := s.signTransaction(ctx, &args, passwd)
	if err != nil {
		log.Warn("Failed transaction send attempt", "from", args.From, "to", args.To, "value", args.Value.ToInt(), "err", err)
		return common.Hash{}, err
	}
	return SubmitTransaction(ctx, s.b, signed)
}

// SignTransaction will create a transaction from the given arguments and
// tries to sign it with the key associated with args.To. If the given passwd isn't
// able to decrypt the key it fails. The transaction is returned in RLP-form, not broadcast
// to other nodes
func (s *PrivateAccountAPI) SignTransaction(ctx context.Context, args SendTxArgs, passwd string) (*SignTransactionResult, error) {
	// No need to obtain the noncelock mutex, since we won't be sending this
	// tx into the transaction pool, but right back to the user
	if args.Gas == nil {
		return nil, fmt.Errorf("gas not specified")
	}
	if args.GasPrice == nil {
		return nil, fmt.Errorf("gasPrice not specified")
	}
	if args.Nonce == nil {
		return nil, fmt.Errorf("nonce not specified")
	}
	signed, err := s.signTransaction(ctx, &args, passwd)
	if err != nil {
		log.Warn("Failed transaction sign attempt", "from", args.From, "to", args.To, "value", args.Value.ToInt(), "err", err)
		return nil, err
	}
	data, err := rlp.EncodeToBytes(signed)
	if err != nil {
		return nil, err
	}
	return &SignTransactionResult{data, signed}, nil
}

// Sign calculates an Ethereum ECDSA signature for:
// keccack256("\x19Ethereum Signed Message:\n" + len(message) + message))
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The key used to calculate the signature is decrypted with the given password.
//
// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_sign
func (s *PrivateAccountAPI) Sign(ctx context.Context, data hexutil.Bytes, addr common.Address, passwd string) (hexutil.Bytes, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Assemble sign the data with the wallet
	signature, err := wallet.SignTextWithPassphrase(account, passwd, data)
	if err != nil {
		log.Warn("Failed data sign attempt", "address", addr, "err", err)
		return nil, err
	}
	signature[crypto.RecoveryIDOffset] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return signature, nil
}

// EcRecover returns the address for the account that was used to create the signature.
// Note, this function is compatible with eth_sign and personal_sign. As such it recovers
// the address of:
// hash = keccak256("\x19Ethereum Signed Message:\n"${message length}${message})
// addr = ecrecover(hash, signature)
//
// Note, the signature must conform to the secp256k1 curve R, S and V values, where
// the V value must be 27 or 28 for legacy reasons.
//
// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_ecRecover
func (s *PrivateAccountAPI) EcRecover(ctx context.Context, data, sig hexutil.Bytes) (common.Address, error) {
	if len(sig) != crypto.SignatureLength {
		return common.Address{}, fmt.Errorf("signature must be %d bytes long", crypto.SignatureLength)
	}
	if sig[crypto.RecoveryIDOffset] != 27 && sig[crypto.RecoveryIDOffset] != 28 {
		return common.Address{}, fmt.Errorf("invalid Ethereum signature (V is not 27 or 28)")
	}
	sig[crypto.RecoveryIDOffset] -= 27 // Transform yellow paper V from 27/28 to 0/1

	rpk, err := crypto.SigToPub(accounts.TextHash(data), sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*rpk), nil
}

// SignAndSendTransaction was renamed to SendTransaction. This method is deprecated
// and will be removed in the future. It primary goal is to give clients time to update.
func (s *PrivateAccountAPI) SignAndSendTransaction(ctx context.Context, args SendTxArgs, passwd string) (common.Hash, error) {
	return s.SendTransaction(ctx, args, passwd)
}

// InitializeWallet initializes a new wallet at the provided URL, by generating and returning a new private key.
func (s *PrivateAccountAPI) InitializeWallet(ctx context.Context, url string) (string, error) {
	wallet, err := s.am.Wallet(url)
	if err != nil {
		return "", err
	}

	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", err
	}

	seed := bip39.NewSeed(mnemonic, "")

	switch wallet := wallet.(type) {
	case *scwallet.Wallet:
		return mnemonic, wallet.Initialize(seed)
	default:
		return "", fmt.Errorf("specified wallet does not support initialization")
	}
}

// Unpair deletes a pairing between wallet and geth.
func (s *PrivateAccountAPI) Unpair(ctx context.Context, url string, pin string) error {
	wallet, err := s.am.Wallet(url)
	if err != nil {
		return err
	}

	switch wallet := wallet.(type) {
	case *scwallet.Wallet:
		return wallet.Unpair([]byte(pin))
	default:
		return fmt.Errorf("specified wallet does not support pairing")
	}
}

// PublicBlockChainAPI provides an API to access the Ethereum blockchain.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicBlockChainAPI struct {
	b Backend
}

// NewPublicBlockChainAPI creates a new Ethereum blockchain API.
func NewPublicBlockChainAPI(b Backend) *PublicBlockChainAPI {
	return &PublicBlockChainAPI{b}
}

// ChainId returns the chainID value for transaction replay protection.
func (s *PublicBlockChainAPI) ChainId() *hexutil.Big {
	return (*hexutil.Big)(s.b.ChainConfig().ChainID)
}

// BlockNumber returns the block number of the chain head.
func (s *PublicBlockChainAPI) BlockNumber() hexutil.Uint64 {
	header, _ := s.b.HeaderByNumber(context.Background(), rpc.LatestBlockNumber) // latest header should always be available
	return hexutil.Uint64(header.Number.Uint64())
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (s *PublicBlockChainAPI) GetBalance(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	state, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	return (*hexutil.Big)(state.GetBalance(address)), state.Error()
}

// Result structs for GetProof
type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}
type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// GetProof returns the Merkle-proof for a given account and optionally some storage keys.
func (s *PublicBlockChainAPI) GetProof(ctx context.Context, address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (*AccountResult, error) {
	state, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}

	storageTrie := state.StorageTrie(address)
	storageHash := types.EmptyRootHash
	codeHash := state.GetCodeHash(address)
	storageProof := make([]StorageResult, len(storageKeys))

	// if we have a storageTrie, (which means the account exists), we can update the storagehash
	if storageTrie != nil {
		storageHash = storageTrie.Hash()
	} else {
		// no storageTrie means the account does not exist, so the codeHash is the hash of an empty bytearray.
		codeHash = crypto.Keccak256Hash(nil)
	}

	// create the proof for the storageKeys
	for i, key := range storageKeys {
		if storageTrie != nil {
			proof, storageError := state.GetStorageProof(address, common.HexToHash(key))
			if storageError != nil {
				return nil, storageError
			}
			storageProof[i] = StorageResult{key, (*hexutil.Big)(state.GetState(address, common.HexToHash(key)).Big()), common.ToHexArray(proof)}
		} else {
			storageProof[i] = StorageResult{key, &hexutil.Big{}, []string{}}
		}
	}

	// create the accountProof
	accountProof, proofErr := state.GetProof(address)
	if proofErr != nil {
		return nil, proofErr
	}

	return &AccountResult{
		Address:      address,
		AccountProof: common.ToHexArray(accountProof),
		Balance:      (*hexutil.Big)(state.GetBalance(address)),
		CodeHash:     codeHash,
		Nonce:        hexutil.Uint64(state.GetNonce(address)),
		StorageHash:  storageHash,
		StorageProof: storageProof,
	}, state.Error()
}

// GetHeaderByNumber returns the requested canonical block header.
// * When blockNr is -1 the chain head is returned.
// * When blockNr is -2 the pending chain head is returned.
func (s *PublicBlockChainAPI) GetHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (map[string]interface{}, error) {
	header, err := s.b.HeaderByNumber(ctx, number)
	if header != nil && err == nil {
		response := s.rpcMarshalHeader(header)
		if number == rpc.PendingBlockNumber {
			// Pending header need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

// GetHeaderByHash returns the requested header by hash.
func (s *PublicBlockChainAPI) GetHeaderByHash(ctx context.Context, hash common.Hash) map[string]interface{} {
	header, _ := s.b.HeaderByHash(ctx, hash)
	if header != nil {
		return s.rpcMarshalHeader(header)
	}
	return nil
}

// GetBlockByNumber returns the requested canonical block.
// * When blockNr is -1 the chain head is returned.
// * When blockNr is -2 the pending chain head is returned.
// * When fullTx is true all transactions in the block are returned, otherwise
//   only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := s.b.BlockByNumber(ctx, number)
	if block != nil && err == nil {
		response, err := s.rpcMarshalBlock(block, true, fullTx)
		if err == nil && number == rpc.PendingBlockNumber {
			// Pending blocks need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := s.b.BlockByHash(ctx, hash)
	if block != nil {
		return s.rpcMarshalBlock(block, true, fullTx)
	}
	return nil, err
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := s.b.BlockByNumber(ctx, blockNr)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			log.Debug("Requested uncle not found", "number", blockNr, "hash", block.Hash(), "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return s.rpcMarshalBlock(block, false, false)
	}
	return nil, err
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := s.b.BlockByHash(ctx, blockHash)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			log.Debug("Requested uncle not found", "number", block.Number(), "hash", blockHash, "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return s.rpcMarshalBlock(block, false, false)
	}
	return nil, err
}

// GetUncleCountByBlockNumber returns number of uncles in the block for the given block number
func (s *PublicBlockChainAPI) GetUncleCountByBlockNumber(ctx context.Context, blockNr rpc.BlockNumber) *hexutil.Uint {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n
	}
	return nil
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
func (s *PublicBlockChainAPI) GetUncleCountByBlockHash(ctx context.Context, blockHash common.Hash) *hexutil.Uint {
	if block, _ := s.b.BlockByHash(ctx, blockHash); block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n
	}
	return nil
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (s *PublicBlockChainAPI) GetCode(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	state, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	code := state.GetCode(address)
	return code, state.Error()
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block
// numbers are also allowed.
func (s *PublicBlockChainAPI) GetStorageAt(ctx context.Context, address common.Address, key string, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	state, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	res := state.GetState(address, common.HexToHash(key))
	return res[:], state.Error()
}

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From     *common.Address `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Data     *hexutil.Bytes  `json:"data"`
}

// account indicates the overriding fields of account during the execution of
// a message call.
// Note, state and stateDiff can't be specified at the same time. If state is
// set, message execution will only use the data in the given state. Otherwise
// if statDiff is set, all diff will be applied first and then execute the call
// message.
type account struct {
	Nonce     *hexutil.Uint64              `json:"nonce"`
	Code      *hexutil.Bytes               `json:"code"`
	Balance   **hexutil.Big                `json:"balance"`
	State     *map[common.Hash]common.Hash `json:"state"`
	StateDiff *map[common.Hash]common.Hash `json:"stateDiff"`
}

func DoCall(ctx context.Context, b Backend, args CallArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides map[common.Address]account, vmCfg vm.Config, timeout time.Duration, globalGasCap *big.Int) ([]byte, uint64, bool, error) {
	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	state, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, 0, false, err
	}
	// Set sender address or use a default if none specified
	var addr common.Address
	if args.From == nil {
		if wallets := b.AccountManager().Wallets(); len(wallets) > 0 {
			if accounts := wallets[0].Accounts(); len(accounts) > 0 {
				addr = accounts[0].Address
			}
		}
	} else {
		addr = *args.From
	}
	// Override the fields of specified contracts before execution.
	for addr, account := range overrides {
		// Override account nonce.
		if account.Nonce != nil {
			state.SetNonce(addr, uint64(*account.Nonce))
		}
		// Override account(contract) code.
		if account.Code != nil {
			state.SetCode(addr, *account.Code)
		}
		// Override account balance.
		if account.Balance != nil {
			state.SetBalance(addr, (*big.Int)(*account.Balance))
		}
		if account.State != nil && account.StateDiff != nil {
			return nil, 0, false, fmt.Errorf("account %s has both 'state' and 'stateDiff'", addr.Hex())
		}
		// Replace entire state if caller requires.
		if account.State != nil {
			state.SetStorage(addr, *account.State)
		}
		// Apply state diff into specified accounts.
		if account.StateDiff != nil {
			for key, value := range *account.StateDiff {
				state.SetState(addr, key, value)
			}
		}
	}
	// Set default gas & gas price if none were set
	gas := uint64(math.MaxUint64 / 2)
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}
	if globalGasCap != nil && globalGasCap.Uint64() < gas {
		log.Warn("Caller gas above allowance, capping", "requested", gas, "cap", globalGasCap)
		gas = globalGasCap.Uint64()
	}
	gasPrice := new(big.Int).SetUint64(defaultGasPrice)
	if args.GasPrice != nil {
		gasPrice = args.GasPrice.ToInt()
	}

	value := new(big.Int)
	if args.Value != nil {
		value = args.Value.ToInt()
	}

	var data []byte
	if args.Data != nil {
		data = []byte(*args.Data)
	}

	// Create new call message
	msg := types.NewMessage(addr, args.To, 0, value, gas, gasPrice, data, false)

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	// Get a new instance of the EVM.
	evm, vmError, err := b.GetEVM(ctx, msg, state, header)
	if err != nil {
		return nil, 0, false, err
	}
	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	// Setup the gas pool (also for unmetered requests)
	// and apply the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	res, gas, failed, err := core.ApplyMessage(evm, msg, gp)
	if err := vmError(); err != nil {
		return nil, 0, false, err
	}
	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, 0, false, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	return res, gas, failed, err
}

// Call executes the given transaction on the state for the given block number.
//
// Additionally, the caller can specify a batch of contract for fields overriding.
//
// Note, this function doesn't make and changes in the state/blockchain and is
// useful to execute and retrieve values.
func (s *PublicBlockChainAPI) Call(ctx context.Context, args CallArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *map[common.Address]account) (hexutil.Bytes, error) {
	var accounts map[common.Address]account
	if overrides != nil {
		accounts = *overrides
	}
	result, _, _, err := DoCall(ctx, s.b, args, blockNrOrHash, accounts, vm.Config{}, 5*time.Second, s.b.RPCGasCap())
	return (hexutil.Bytes)(result), err
}

func DoEstimateGas(ctx context.Context, b Backend, args CallArgs, blockNrOrHash rpc.BlockNumberOrHash, gasCap *big.Int) (hexutil.Uint64, error) {
	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		lo  uint64 = params.TxGas - 1
		hi  uint64
		cap uint64
	)
	if args.Gas != nil && uint64(*args.Gas) >= params.TxGas {
		hi = uint64(*args.Gas)
	} else {
		// Retrieve the block to act as the gas ceiling
		block, err := b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if err != nil {
			return 0, err
		}
		hi = block.GasLimit()
	}
	if gasCap != nil && hi > gasCap.Uint64() {
		log.Warn("Caller gas above allowance, capping", "requested", hi, "cap", gasCap)
		hi = gasCap.Uint64()
	}
	cap = hi

	// Set sender address or use a default if none specified
	if args.From == nil {
		if wallets := b.AccountManager().Wallets(); len(wallets) > 0 {
			if accounts := wallets[0].Accounts(); len(accounts) > 0 {
				args.From = &accounts[0].Address
			}
		}
	}
	// Use zero-address if none other is available
	if args.From == nil {
		args.From = &common.Address{}
	}
	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64) bool {
		args.Gas = (*hexutil.Uint64)(&gas)

		_, _, failed, err := DoCall(ctx, b, args, blockNrOrHash, nil, vm.Config{}, 0, gasCap)
		if err != nil || failed {
			return false
		}
		return true
	}
	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		if !executable(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
		if !executable(hi) {
			return 0, fmt.Errorf("gas required exceeds allowance (%d) or always failing transaction", cap)
		}
	}
	return hexutil.Uint64(hi), nil
}

// EstimateGas returns an estimate of the amount of gas needed to execute the
// given transaction against the current pending block.
func (s *PublicBlockChainAPI) EstimateGas(ctx context.Context, args CallArgs) (hexutil.Uint64, error) {
	blockNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.PendingBlockNumber)
	return DoEstimateGas(ctx, s.b, args, blockNrOrHash, s.b.RPCGasCap())
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
type ExecutionResult struct {
	Gas         uint64         `json:"gas"`
	Failed      bool           `json:"failed"`
	ReturnValue string         `json:"returnValue"`
	StructLogs  []StructLogRes `json:"structLogs"`
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc      uint64             `json:"pc"`
	Op      string             `json:"op"`
	Gas     uint64             `json:"gas"`
	GasCost uint64             `json:"gasCost"`
	Depth   int                `json:"depth"`
	Error   error              `json:"error,omitempty"`
	Stack   *[]string          `json:"stack,omitempty"`
	Memory  *[]string          `json:"memory,omitempty"`
	Storage *map[string]string `json:"storage,omitempty"`
}

// FormatLogs formats EVM returned structured logs for json output
func FormatLogs(logs []vm.StructLog) []StructLogRes {
	formatted := make([]StructLogRes, len(logs))
	for index, trace := range logs {
		formatted[index] = StructLogRes{
			Pc:      trace.Pc,
			Op:      trace.Op.String(),
			Gas:     trace.Gas,
			GasCost: trace.GasCost,
			Depth:   trace.Depth,
			Error:   trace.Err,
		}
		if trace.Stack != nil {
			stack := make([]string, len(trace.Stack))
			for i, stackValue := range trace.Stack {
				stack[i] = fmt.Sprintf("%x", math.PaddedBigBytes(stackValue, 32))
			}
			formatted[index].Stack = &stack
		}
		if trace.Memory != nil {
			memory := make([]string, 0, (len(trace.Memory)+31)/32)
			for i := 0; i+32 <= len(trace.Memory); i += 32 {
				memory = append(memory, fmt.Sprintf("%x", trace.Memory[i:i+32]))
			}
			formatted[index].Memory = &memory
		}
		if trace.Storage != nil {
			storage := make(map[string]string)
			for i, storageValue := range trace.Storage {
				storage[fmt.Sprintf("%x", i)] = fmt.Sprintf("%x", storageValue)
			}
			formatted[index].Storage = &storage
		}
	}
	return formatted
}

// RPCMarshalHeader converts the given header to the RPC output .
func RPCMarshalHeader(head *types.Header) map[string]interface{} {
	return map[string]interface{}{
		"number":           (*hexutil.Big)(head.Number),
		"hash":             head.Hash(),
		"parentHash":       head.ParentHash,
		"nonce":            head.Nonce,
		"mixHash":          head.MixDigest,
		"sha3Uncles":       head.UncleHash,
		"logsBloom":        head.Bloom,
		"stateRoot":        head.Root,
		"miner":            head.Coinbase,
		"difficulty":       (*hexutil.Big)(head.Difficulty),
		"extraData":        hexutil.Bytes(head.Extra),
		"size":             hexutil.Uint64(head.Size()),
		"gasLimit":         hexutil.Uint64(head.GasLimit),
		"gasUsed":          hexutil.Uint64(head.GasUsed),
		"timestamp":        hexutil.Uint64(head.Time),
		"transactionsRoot": head.TxHash,
		"receiptsRoot":     head.ReceiptHash,
	}
}

// RPCMarshalBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func RPCMarshalBlock(block *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	fields := RPCMarshalHeader(block.Header())
	fields["size"] = hexutil.Uint64(block.Size())

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}
		if fullTx {
			formatTx = func(tx *types.Transaction) (interface{}, error) {
				return newRPCTransactionFromBlockHash(block, tx.Hash()), nil
			}
		}
		txs := block.Transactions()
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range txs {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}
		fields["transactions"] = transactions
	}
	uncles := block.Uncles()
	uncleHashes := make([]common.Hash, len(uncles))
	for i, uncle := range uncles {
		uncleHashes[i] = uncle.Hash()
	}
	fields["uncles"] = uncleHashes

	return fields, nil
}

// rpcMarshalHeader uses the generalized output filler, then adds the total difficulty field, which requires
// a `PublicBlockchainAPI`.
func (s *PublicBlockChainAPI) rpcMarshalHeader(header *types.Header) map[string]interface{} {
	fields := RPCMarshalHeader(header)
	fields["totalDifficulty"] = (*hexutil.Big)(s.b.GetTd(header.Hash()))
	return fields
}

// rpcMarshalBlock uses the generalized output filler, then adds the total difficulty field, which requires
// a `PublicBlockchainAPI`.
func (s *PublicBlockChainAPI) rpcMarshalBlock(b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	fields, err := RPCMarshalBlock(b, inclTx, fullTx)
	if err != nil {
		return nil, err
	}
	if inclTx {
		fields["totalDifficulty"] = (*hexutil.Big)(s.b.GetTd(b.Hash()))
	}
	return fields, err
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        *common.Hash    `json:"blockHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              hexutil.Uint64  `json:"gas"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	Hash             common.Hash     `json:"hash"`
	Input            hexutil.Bytes   `json:"input"`
	Nonce            hexutil.Uint64  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
	Value            *hexutil.Big    `json:"value"`
	V                *hexutil.Big    `json:"v"`
	R                *hexutil.Big    `json:"r"`
	S                *hexutil.Big    `json:"s"`
	ID               hexutil.Uint64  `json:"ID"`
	ErpkC1           *hexutil.Bytes  `json:"erpkc1"`
	ErpkC2           *hexutil.Bytes  `json:"erpkc2"`
	EspkC1           *hexutil.Bytes  `json:"espkc1"`
	EspkC2           *hexutil.Bytes  `json:"espkc2"`
	CMRpk            *hexutil.Bytes  `json:"cmrpk"`
	CMSpk            *hexutil.Bytes  `json:"cmspk"`
	ErpkEPs0         *hexutil.Bytes  `json:"erpkeps0"`
	ErpkEPs1         *hexutil.Bytes  `json:"erpkeps1"`
	ErpkEPs2         *hexutil.Bytes  `json:"erpkeps2"`
	ErpkEPs3         *hexutil.Bytes  `json:"erpkeps3"`
	ErpkEPt          *hexutil.Bytes  `json:"erpkept"`
	EspkEPs0         *hexutil.Bytes  `json:"espkeps0"`
	EspkEPs1         *hexutil.Bytes  `json:"espkeps1"`
	EspkEPs2         *hexutil.Bytes  `json:"espkeps2"`
	EspkEPs3         *hexutil.Bytes  `json:"espkeps3"`
	EspkEPt          *hexutil.Bytes  `json:"espkept"`
	EvSC1            *hexutil.Bytes  `json:"evsc1"`
	EvSC2            *hexutil.Bytes  `json:"evsc2"`
	EvRC1            *hexutil.Bytes  `json:"evrc1"`
	EvRC2            *hexutil.Bytes  `json:"evrc2"`
	CmS              *hexutil.Bytes  `json:"cms"`
	CmR              *hexutil.Bytes  `json:"cmr"`
	CMsFPC           *hexutil.Bytes  `json:"cmsfpc"`
	CMsFPZ1          *hexutil.Bytes  `json:"cmsfpz1"`
	CMsFPZ2          *hexutil.Bytes  `json:"cmsfpz2"`
	CMrFPC           *hexutil.Bytes  `json:"cmrfpc"`
	CMrFPZ1          *hexutil.Bytes  `json:"cmrfpz1"`
	CMrFPZ2          *hexutil.Bytes  `json:"cmrfpz2"`
	EvsBsC1          *hexutil.Bytes  `json:"evsbsc1"`
	EvsBsC2          *hexutil.Bytes  `json:"evsbsc2"`
	EvOC1            *hexutil.Bytes  `json:"evoc1"`
	EvOC2            *hexutil.Bytes  `json:"evoc2"`
	CmO              *hexutil.Bytes  `json:"cmo"`
	EvOEPs0          *hexutil.Bytes  `json:"evoeps0"`
	EvOEPs1          *hexutil.Bytes  `json:"evoeps1"`
	EvOEPs2          *hexutil.Bytes  `json:"evoeps2"`
	EvOEPs3          *hexutil.Bytes  `json:"evoeps3"`
	EvOEPt           *hexutil.Bytes  `json:"evoept"`
	BPC              *hexutil.Bytes  `json:"bpc"`
	BPRV             *hexutil.Bytes  `json:"bprv"`
	BPRR             *hexutil.Bytes  `json:"bprr"`
	BPSV             *hexutil.Bytes  `json:"bpsv"`
	BPSR             *hexutil.Bytes  `json:"bpsr"`
	BPSOr            *hexutil.Bytes  `json:"bpsor"`
	EpkrC1           *hexutil.Bytes  `json:"epkrc1"`
	EpkrC2           *hexutil.Bytes  `json:"epkrc2"`
	EpkpC1           *hexutil.Bytes  `json:"epkpc1"`
	EpkpC2           *hexutil.Bytes  `json:"epkpc2"`
	SigM             *hexutil.Bytes  `json:"sigm"`
	SigMHash         *hexutil.Bytes  `json:"sigmhash"`
	SigR             *hexutil.Bytes  `json:"sigr"`
	SigS             *hexutil.Bytes  `json:"sigs"`
	CmV              *hexutil.Bytes  `json:"cmv"`
}

// newRPCTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func newRPCTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber uint64, index uint64) *RPCTransaction {
	var signer types.Signer = types.FrontierSigner{}
	if tx.Protected() {
		signer = types.NewEIP155Signer(tx.ChainId())
	}
	from, _ := types.Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()

	result := &RPCTransaction{
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
		ID:       hexutil.Uint64(tx.ID()),
		ErpkC1:   tx.ErpkC1(),
		ErpkC2:   tx.ErpkC2(),
		EspkC1:   tx.EspkC1(),
		EspkC2:   tx.EspkC2(),
		CMRpk:    tx.CMRpk(),
		CMSpk:    tx.CMSpk(),
		ErpkEPs0: tx.ErpkEPs0(),
		ErpkEPs1: tx.ErpkEPs1(),
		ErpkEPs2: tx.ErpkEPs2(),
		ErpkEPs3: tx.ErpkEPs3(),
		ErpkEPt:  tx.ErpkEPt(),
		EspkEPs0: tx.EspkEPs0(),
		EspkEPs1: tx.EspkEPs1(),
		EspkEPs2: tx.EspkEPs2(),
		EspkEPs3: tx.EspkEPs3(),
		EspkEPt:  tx.EspkEPt(),
		EvSC1:    tx.EvSC1(),
		EvSC2:    tx.EvSC2(),
		EvRC1:    tx.EvRC1(),
		EvRC2:    tx.EvRC2(),
		CmS:      tx.CmS(),
		CmR:      tx.CmR(),
		CMsFPC:   tx.CMsFPC(),
		CMsFPZ1:  tx.CMsFPZ1(),
		CMsFPZ2:  tx.CMsFPZ2(),
		CMrFPC:   tx.CMrFPC(),
		CMrFPZ1:  tx.CMrFPZ1(),
		CMrFPZ2:  tx.CMrFPZ2(),
		EvsBsC1:  tx.EvsBsC1(),
		EvsBsC2:  tx.EvsBsC2(),
		EvOC1:    tx.EvOC1(),
		EvOC2:    tx.EvOC2(),
		CmO:      tx.CmO(),
		EvOEPs0:  tx.EvOEPs0(),
		EvOEPs1:  tx.EvOEPs1(),
		EvOEPs2:  tx.EvOEPs2(),
		EvOEPs3:  tx.EvOEPs3(),
		EvOEPt:   tx.EvOEPt(),
		BPC:      tx.BPC(),
		BPRV:     tx.BPRV(),
		BPRR:     tx.BPRR(),
		BPSV:     tx.BPSV(),
		BPSR:     tx.BPSR(),
		BPSOr:    tx.BPSOr(),
		EpkrC1:   tx.EpkrC1(),
		EpkrC2:   tx.EpkrC2(),
		EpkpC1:   tx.EpkpC1(),
		EpkpC2:   tx.EpkpC2(),
		SigM:     tx.SigM(),
		SigMHash: tx.SigMHash(),
		SigR:     tx.SigR(),
		SigS:     tx.SigS(),
		CmV:      tx.CmV(),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}
	return result
}

// newRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func newRPCPendingTransaction(tx *types.Transaction) *RPCTransaction {
	return newRPCTransaction(tx, common.Hash{}, 0, 0)
}

// newRPCTransactionFromBlockIndex returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(b *types.Block, index uint64) *RPCTransaction {
	txs := b.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	return newRPCTransaction(txs[index], b.Hash(), b.NumberU64(), index)
}

// newRPCRawTransactionFromBlockIndex returns the bytes of a transaction given a block and a transaction index.
func newRPCRawTransactionFromBlockIndex(b *types.Block, index uint64) hexutil.Bytes {
	txs := b.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	blob, _ := rlp.EncodeToBytes(txs[index])
	return blob
}

// newRPCTransactionFromBlockHash returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockHash(b *types.Block, hash common.Hash) *RPCTransaction {
	for idx, tx := range b.Transactions() {
		if tx.Hash() == hash {
			return newRPCTransactionFromBlockIndex(b, uint64(idx))
		}
	}
	return nil
}

// PublicTransactionPoolAPI exposes methods for the RPC interface
type PublicTransactionPoolAPI struct {
	b         Backend
	nonceLock *AddrLocker
}

// NewPublicTransactionPoolAPI creates a new RPC service with methods specific for the transaction pool.
func NewPublicTransactionPoolAPI(b Backend, nonceLock *AddrLocker) *PublicTransactionPoolAPI {
	return &PublicTransactionPoolAPI{b, nonceLock}
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByNumber(ctx context.Context, blockNr rpc.BlockNumber) *hexutil.Uint {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n
	}
	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByHash(ctx context.Context, blockHash common.Hash) *hexutil.Uint {
	if block, _ := s.b.BlockByHash(ctx, blockHash); block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n
	}
	return nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.b.BlockByHash(ctx, blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetRawTransactionByBlockNumberAndIndex returns the bytes of the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetRawTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) hexutil.Bytes {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetRawTransactionByBlockHashAndIndex returns the bytes of the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetRawTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) hexutil.Bytes {
	if block, _ := s.b.BlockByHash(ctx, blockHash); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *PublicTransactionPoolAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Uint64, error) {
	// Ask transaction pool for the nonce which includes pending transactions
	if blockNr, ok := blockNrOrHash.Number(); ok && blockNr == rpc.PendingBlockNumber {
		nonce, err := s.b.GetPoolNonce(ctx, address)
		if err != nil {
			return nil, err
		}
		return (*hexutil.Uint64)(&nonce), nil
	}
	// Resolve block number and use its state to ask for the nonce
	state, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	nonce := state.GetNonce(address)
	return (*hexutil.Uint64)(&nonce), state.Error()
}

// GetTransactionByHash returns the transaction for the given hash
func (s *PublicTransactionPoolAPI) GetTransactionByHash(ctx context.Context, hash common.Hash) (*RPCTransaction, error) {
	// 先从区块中找交易
	// Try to return an already finalized transaction
	tx, blockHash, blockNumber, index, err := s.b.GetTransaction(ctx, hash)
	if err != nil {
		return nil, err
	}
	if tx != nil {
		return newRPCTransaction(tx, blockHash, blockNumber, index), nil
	}
	// 区块中没找到交易，去交易池里找
	// No finalized transaction, try to retrieve it from the pool
	if tx := s.b.GetPoolTransaction(hash); tx != nil {
		return newRPCPendingTransaction(tx), nil
	}

	// Transaction unknown, return as such
	return nil, nil
}

// GetRawTransactionByHash returns the bytes of the transaction for the given hash.
func (s *PublicTransactionPoolAPI) GetRawTransactionByHash(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	// Retrieve a finalized transaction, or a pooled otherwise
	tx, _, _, _, err := s.b.GetTransaction(ctx, hash)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		if tx = s.b.GetPoolTransaction(hash); tx == nil {
			// Transaction not found anywhere, abort
			return nil, nil
		}
	}
	// Serialize to RLP and return
	return rlp.EncodeToBytes(tx)
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *PublicTransactionPoolAPI) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(s.b.ChainDb(), hash)
	if tx == nil {
		return nil, nil
	}
	receipts, err := s.b.GetReceipts(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	if len(receipts) <= int(index) {
		return nil, nil
	}
	receipt := receipts[index]

	var signer types.Signer = types.FrontierSigner{}
	if tx.Protected() {
		signer = types.NewEIP155Signer(tx.ChainId())
	}
	from, _ := types.Sender(signer, tx)

	fields := map[string]interface{}{
		"blockHash":         blockHash,
		"blockNumber":       hexutil.Uint64(blockNumber),
		"transactionHash":   hash,
		"transactionIndex":  hexutil.Uint64(index),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		"logsBloom":         receipt.Bloom,
	}

	// Assign receipt status or post state.
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields, nil
}

// sign is a helper function that signs a transaction with the private key of the given address.
func (s *PublicTransactionPoolAPI) sign(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Request the wallet to sign the transaction
	return wallet.SignTx(account, tx, s.b.ChainConfig().ChainID)
}

// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
// 只接收从Postman发过来的参数，所以不包括零知识证明那些
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	ID       *hexutil.Uint64 `json:"id"`
	// We accept "data" and "input" for backwards-compatibility reasons. "input" is the
	// newer name and should be preferred by clients.
	Data     *hexutil.Bytes  `json:"data"`
	Input    *hexutil.Bytes  `json:"input"`
	CmO      *hexutil.Bytes  `json:"cmo"` // 总金额承诺
	Vr       *hexutil.Uint64 `json:"r"`   // 找零金额
	Vs       *hexutil.Uint64 `json:"s"`   // 花费金额
	VoR      *hexutil.Bytes  `json:"vor"`
	Spk      *string         `json:"spk"` // 发送方公钥
	Rpk      *string         `json:"rpk"` // 接收方公钥
	EpkrC1   *hexutil.Bytes  `json:"epkrc1"`
	EpkrC2   *hexutil.Bytes  `json:"epkrc2"`
	EpkpC1   *hexutil.Bytes  `json:"epkpc1"`
	EpkpC2   *hexutil.Bytes  `json:"epkpc2"`
	SigM     *hexutil.Bytes  `json:"sigm"`
	SigMHash *hexutil.Bytes  `json:"sigmhash"`
	SigR     *hexutil.Bytes  `json:"sigr"`
	SigS     *hexutil.Bytes  `json:"sigs"`
	CmV      *hexutil.Bytes  `json:"cmv"`
}

// setDefaults is a helper function that fills in default values for unspecified tx fields.
func (args *SendTxArgs) setDefaults(ctx context.Context, b Backend) error {
	if args.GasPrice == nil {
		price, err := b.SuggestPrice(ctx)
		if err != nil {
			return err
		}
		args.GasPrice = (*hexutil.Big)(price)
	}
	if args.Value == nil {
		args.Value = new(hexutil.Big)
	}
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.From)
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}
	if args.To == nil {
		// Contract creation
		var input []byte
		if args.Data != nil {
			input = *args.Data
		} else if args.Input != nil {
			input = *args.Input
		}
		if len(input) == 0 {
			return errors.New(`contract creation without any data provided`)
		}
	}
	// Estimate the gas usage if necessary.
	if args.Gas == nil {
		// For backwards-compatibility reason, we try both input and data
		// but input is preferred.
		input := args.Input
		if input == nil {
			input = args.Data
		}
		callArgs := CallArgs{
			From:     &args.From, // From shouldn't be nil
			To:       args.To,
			GasPrice: args.GasPrice,
			Value:    args.Value,
			Data:     input,
		}
		pendingBlockNr := rpc.BlockNumberOrHashWithNumber(rpc.PendingBlockNumber)
		estimated, err := DoEstimateGas(ctx, b, callArgs, pendingBlockNr, b.RPCGasCap())
		if err != nil {
			return err
		}
		args.Gas = &estimated
		log.Trace("Estimate gas usage automatically", "gas", args.Gas)
	}
	return nil
}
func paraPK(pk string) (zkp.PublicKey, error) {
	if len(pk) != 256 {
		//return errors.New(`public key length should be 256`)
	}
	P, _ := new(big.Int).SetString(pk[:64], 16)
	G1, _ := new(big.Int).SetString(pk[64:128], 16)
	G2, _ := new(big.Int).SetString(pk[128:192], 16)
	H, _ := new(big.Int).SetString(pk[192:], 16)
	pubK := zkp.PublicKey{
		P:  P,
		G1: G1,
		G2: G2,
		H:  H,
	}
	return pubK, nil
}
func Hash(str string) []byte {
	//使用sha256哈希函数
	h := sha256.New()
	h.Write([]byte(str))
	return h.Sum(nil)
}
func (args *SendTxArgs) chechParameter() error {
	lackofParameterError := errors.New(`lack of parameter`)
	if args.ID == nil {
		return errors.New(`transaction ID should be decleared`)
	}
	ID := uint64(*args.ID)
	if ID != 0 && ID != 1 {
		return errors.New(`wrong transaction ID`)
	}
	// ID == 0 转账交易 ID == 1 购币交易
	if ID == 0 {
		if args.Spk == nil || args.Rpk == nil || args.Vs == nil || args.Vr == nil || args.VoR == nil || args.CmO == nil {
			return lackofParameterError
		}
	} else if ID == 1 {
		if args.EpkrC1 == nil || args.EpkrC2 == nil || args.EpkpC1 == nil || args.EpkpC2 == nil || args.SigM == nil || args.SigMHash == nil || args.SigR == nil || args.SigS == nil || args.CmV == nil {
			return lackofParameterError
		}
	}

	return nil
}
func (args *SendTxArgs) toZeroTransaction(regulator types.Regulator) (*types.Transaction, error) {

	Vs := uint64(*args.Vs)
	Vr := uint64(*args.Vr)
	Rpk, _ := paraPK(*args.Rpk) // 接收方公钥，不写入交易
	//Spk, _ := paraPK(*args.Spk)                      // 发送方公钥，不写入交易
	VoR := []byte(*args.VoR)                       // 被花费承诺的随机数，不写入交易
	CmO := []byte(*args.CmO)                       // 被花费承诺
	regulatorPubk := zkp.PublicKey(regulator.PubK) // 监管者公钥，不写入交易
	addrpkt := new(big.Int).SetBytes(Hash(*args.Rpk))
	addrpk := addrpkt.Mod(addrpkt, regulatorPubk.P).Bytes() //接收方地址公钥，不写入交易
	addspkt := new(big.Int).SetBytes(Hash(*args.Spk))
	addspk := addspkt.Mod(addspkt, regulatorPubk.P).Bytes() //发送方地址公钥，不写入交易
	// 加密并承诺双方地址公钥
	Erpk, _CMrpk, _ := zkp.EncryptAddress(regulatorPubk, addrpk)
	Espk, _CMspk, _ := zkp.EncryptAddress(regulatorPubk, addspk)
	_, CMrpk, _ := zkp.EncryptAddress(regulatorPubk, addrpk)
	_, CMspk, _ := zkp.EncryptAddress(regulatorPubk, addspk)
	// 双方地址公钥相等证明
	ErpkEP := zkp.GenerateAddressEqualityProof(regulatorPubk, regulatorPubk, CMrpk, _CMrpk, addrpk)
	EspkEP := zkp.GenerateAddressEqualityProof(regulatorPubk, regulatorPubk, CMspk, _CMspk, addspk)
	// 花费额承诺，格式正确证明
	EvS, CmS, _ := zkp.EncryptValue(regulatorPubk, Vs)
	CMsFP := zkp.GenerateFormatProof(regulatorPubk, Vs, CmS.R)
	Evs, _, _ := zkp.EncryptValue(Rpk, Vs) // 接收方公钥加密发送金额
	// 找零承诺，格式正确证明
	EvR, CmR, _ := zkp.EncryptValue(regulatorPubk, Vr)
	CMrFP := zkp.GenerateFormatProof(regulatorPubk, Vr, CmR.R)
	// 总花费额，由找零和发出相加求得
	EvO, CMo, _ := zkp.EncryptValue(regulatorPubk, Vr+Vs)
	// 总额度相等证明
	EvoEP := zkp.GenerateEqualityProof(regulatorPubk, regulatorPubk, CMo, zkp.Commitment{
		Commitment: CmO,
		R:          VoR,
	}, uint(Vr+Vs))
	// 会计平衡证明
	BP := zkp.GenerateBalanceProof(regulatorPubk, Vr, Vs, 0, CmR.R, CmS.R, VoR)
	// 将需要编码进入交易的量转换成*big.Int或*hexutil.Uint64或*hexutil.Bytes
	// CmO是 *hexutil.Bytes，不需编码
	ErpkC1, ErpkC2 := hexutil.Bytes(Erpk.C1), hexutil.Bytes(Erpk.C2)
	EspkC1, EspkC2 := hexutil.Bytes(Espk.C1), hexutil.Bytes(Espk.C2)
	CMRpk, CMSpk := hexutil.Bytes(CMrpk.Commitment), hexutil.Bytes(CMspk.Commitment)
	ErpkEPs0, ErpkEPs1, ErpkEPs2, ErpkEPs3, ErpkEPt := hexutil.Bytes(ErpkEP.LinearEquationProof.S[0]), hexutil.Bytes(ErpkEP.LinearEquationProof.S[1]), hexutil.Bytes(ErpkEP.LinearEquationProof.S[2]), hexutil.Bytes(ErpkEP.LinearEquationProof.S[3]), hexutil.Bytes(ErpkEP.LinearEquationProof.T)
	EspkEPs0, EspkEPs1, EspkEPs2, EspkEPs3, EspkEPt := hexutil.Bytes(EspkEP.LinearEquationProof.S[0]), hexutil.Bytes(EspkEP.LinearEquationProof.S[1]), hexutil.Bytes(EspkEP.LinearEquationProof.S[2]), hexutil.Bytes(EspkEP.LinearEquationProof.S[3]), hexutil.Bytes(EspkEP.LinearEquationProof.T)
	EvSC1, EvSC2 := hexutil.Bytes(EvS.C1), hexutil.Bytes(EvS.C2)
	EvRC1, EvRC2 := hexutil.Bytes(EvR.C1), hexutil.Bytes(EvR.C2)
	_CmS, _CmR := hexutil.Bytes(CmS.Commitment), hexutil.Bytes(CmR.Commitment)
	CMsFPC, CMsFPZ1, CMsFPZ2 := hexutil.Bytes(CMsFP.C), hexutil.Bytes(CMsFP.Z1), hexutil.Bytes(CMsFP.Z2)
	CMrFPC, CMrFPZ1, CMrFPZ2 := hexutil.Bytes(CMrFP.C), hexutil.Bytes(CMrFP.Z1), hexutil.Bytes(CMrFP.Z2)
	EvsBsC1, EvsBsC2 := hexutil.Bytes(Evs.C1), hexutil.Bytes(Evs.C2)
	EvOC1, EvOC2 := hexutil.Bytes(EvO.C1), hexutil.Bytes(EvO.C2)
	_CmO := hexutil.Bytes(CmO)
	EvOEPs0, EvOEPs1, EvOEPs2, EvOEPs3, EvOEPt := hexutil.Bytes(EvoEP.LinearEquationProof.S[0]), hexutil.Bytes(EvoEP.LinearEquationProof.S[1]), hexutil.Bytes(EvoEP.LinearEquationProof.S[2]), hexutil.Bytes(EvoEP.LinearEquationProof.S[3]), hexutil.Bytes(EvoEP.LinearEquationProof.T)
	BPC, BPRV, BPRR, BPSV, BPSR, BPSOr := hexutil.Bytes(BP.C), hexutil.Bytes(BP.R_v), hexutil.Bytes(BP.R_r), hexutil.Bytes(BP.S_v), hexutil.Bytes(BP.S_r), hexutil.Bytes(BP.S_or)
	// TODO:产生签名Sig
	// fmt.Println(ErpkC1, ErpkC2, EspkC1, EspkC2, CMRpk, CMSpk, ErpkEPs0, ErpkEPs1, ErpkEPs2, ErpkEPs3, ErpkEPt, EspkEPs0, EspkEPs1, EspkEPs2, EspkEPs3, EspkEPt, EvSC1, EvSC2, EvRC1, EvRC2, _CmS, _CmR, CMsFPC, CMsFPZ1, CMsFPZ2, CMrFPC, CMrFPZ1, CMrFPZ2, EvsBsC1, EvsBsC2, EvOC1, EvOC2, _CmO, EvOEPs0, EvOEPs1, EvOEPs2, EvOEPs3, EvOEPt, BPC, BPRV, BPRR, BPSV, BPSR, BPSOr)
	// 以上

	//fmt.Println(Erpk.C1, Erpk.C2, Espk.C1, Espk.C2)            // 4个字节数组
	//fmt.Println(ErpkEP, EspkEP)                                //
	//fmt.Println(addrpk, addspk)                                //
	//fmt.Println(Rpk, Spk)                                      // 不需要编码
	//fmt.Println(VoR)                                           // 字节数组
	//fmt.Println(EvS.C1, EvS.C2, CMsFP.C, CMsFP.Z1, CMsFP.Z2)   // 5个字节数组
	//fmt.Println(Evs.C1, Evs.C2)                                // 2个字节数组
	//fmt.Println(EvR.C1, EvR.C2, CMrFP.C, CMrFP.Z1, CMrFP.Z2)   // 5个字节数组
	//fmt.Println(BP.C, BP.R_r, BP.R_v, BP.S_or, BP.S_r, BP.S_v) // 6个字节数组
	//fmt.Println(EvoEP)                                         // 总额度相等证明
	// 验证
	//verify := zkp.VerifyFormatProof(EvS, regulatorPubk, CMsFP)
	//fmt.Println("花费额承诺，格式正确证明:", verify,"EVS: ",EvS,"pub: ",regulatorPubk,"CMSFP:",CMsFP)
	//verify = zkp.VerifyFormatProof(EvR, regulatorPubk, CMrFP)
	//fmt.Println("找零承诺，格式正确证明:", verify)
	//verify := zkp.VerifyBalanceProof(CmR.Commitment, CmS.Commitment, CmO, regulatorPubk, BP)
	//fmt.Println(CmR.Commitment, CmS.Commitment, CmO, regulatorPubk, BP)
	//fmt.Println("会计平衡证明:", verify)
	//verify := zkp.VerifyEqualityProof(regulatorPubk, regulatorPubk, EvO, zkp.CypherText{C1: nil, C2: CmO}, EvoEP) //EvO和CmO里面的金额相等
	//fmt.Println(regulatorPubk, regulatorPubk, EvO, zkp.CypherText{C1: nil, C2: CmO}, EvoEP)
	//fmt.Println("lenS: ",len(EvoEP.S),"lenT: ",len(EvoEP.T))
	//fmt.Println("总额度相等证明:", verify)
	//s1 := make([][]byte,4)
	//t1 := make([]byte,32)
	//l := zkp.LinearEquationProof{s1,t1}
	//e := zkp.EqualityProof{l}
	//
	//e.LinearEquationProof.S[0] = EvOEPs0.Btob()
	//e.LinearEquationProof.S[1] = EvOEPs1.Btob()
	//e.LinearEquationProof.S[2] = EvOEPs2.Btob()
	//e.LinearEquationProof.S[3] = EvOEPs3.Btob()
	//e.LinearEquationProof.T= EvOEPt.Btob()
	//fmt.Println("ten")
	//verify = zkp.VerifyEqualityProof(regulatorPubk, regulatorPubk, EvO, zkp.CypherText{C1: nil, C2: CmO}, e)
	//fmt.Println("总额度相等证明2:", verify)
	//verify = zkp.VerifyEqualityProof(regulatorPubk, regulatorPubk, Erpk, zkp.CypherText{C1: nil, C2: CMrpk.Commitment}, ErpkEP) //EvO和CmO里面的金额相等
	//fmt.Println("接收方公钥相等证明:", verify)
	//verify = zkp.VerifyEqualityProof(regulatorPubk, regulatorPubk, Espk, zkp.CypherText{C1: nil, C2: CMspk.Commitment}, EspkEP) //EvO和CmO里面的金额相等
	//fmt.Println("发送方公钥相等证明:", verify)

	var input []byte
	if args.Input != nil {
		input = *args.Input
	} else if args.Data != nil {
		input = *args.Data
	}
	if args.To == nil {
		return types.NewContractCreation(uint64(*args.Nonce), (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input, 0, &ErpkC1, &ErpkC2, &EspkC1, &EspkC2, &CMRpk, &CMSpk, &ErpkEPs0, &ErpkEPs1, &ErpkEPs2, &ErpkEPs3, &ErpkEPt, &EspkEPs0, &EspkEPs1, &EspkEPs2, &EspkEPs3, &EspkEPt, &EvSC1, &EvSC2, &EvRC1, &EvRC2, &_CmS, &_CmR, &CMsFPC, &CMsFPZ1, &CMsFPZ2, &CMrFPC, &CMrFPZ1, &CMrFPZ2, &EvsBsC1, &EvsBsC2, &EvOC1, &EvOC2, &_CmO, &EvOEPs0, &EvOEPs1, &EvOEPs2, &EvOEPs3, &EvOEPt, &BPC, &BPRV, &BPRR, &BPSV, &BPSR, &BPSOr, args.EpkrC1, args.EpkrC2, args.EpkpC1, args.EpkpC2, args.SigM, args.SigMHash, args.SigR, args.SigS, args.CmV), nil
	}
	return types.NewTransaction(uint64(*args.Nonce), *args.To, (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input, 0, &ErpkC1, &ErpkC2, &EspkC1, &EspkC2, &CMRpk, &CMSpk, &ErpkEPs0, &ErpkEPs1, &ErpkEPs2, &ErpkEPs3, &ErpkEPt, &EspkEPs0, &EspkEPs1, &EspkEPs2, &EspkEPs3, &EspkEPt, &EvSC1, &EvSC2, &EvRC1, &EvRC2, &_CmS, &_CmR, &CMsFPC, &CMsFPZ1, &CMsFPZ2, &CMrFPC, &CMrFPZ1, &CMrFPZ2, &EvsBsC1, &EvsBsC2, &EvOC1, &EvOC2, &_CmO, &EvOEPs0, &EvOEPs1, &EvOEPs2, &EvOEPs3, &EvOEPt, &BPC, &BPRV, &BPRR, &BPSV, &BPSR, &BPSOr, args.EpkrC1, args.EpkrC2, args.EpkpC1, args.EpkpC2, args.SigM, args.SigMHash, args.SigR, args.SigS, args.CmV), nil
}

func (args *SendTxArgs) toExTransaction(regulator types.Regulator) (*types.Transaction, error) {
	var input []byte
	if args.Input != nil {
		input = *args.Input
	} else if args.Data != nil {
		input = *args.Data
	}
	ErpkC1 := hexutil.Bytes(nil)
	ErpkC2 := hexutil.Bytes(nil)
	EspkC1 := hexutil.Bytes(nil)
	EspkC2 := hexutil.Bytes(nil)
	CMRpk := hexutil.Bytes(nil)
	CMSpk := hexutil.Bytes(nil)
	ErpkEPs0 := hexutil.Bytes(nil)
	ErpkEPs1 := hexutil.Bytes(nil)
	ErpkEPs2 := hexutil.Bytes(nil)
	ErpkEPs3 := hexutil.Bytes(nil)
	ErpkEPt := hexutil.Bytes(nil)
	EspkEPs0 := hexutil.Bytes(nil)
	EspkEPs1 := hexutil.Bytes(nil)
	EspkEPs2 := hexutil.Bytes(nil)
	EspkEPs3 := hexutil.Bytes(nil)
	EspkEPt := hexutil.Bytes(nil)
	EvSC1 := hexutil.Bytes(nil)
	EvSC2 := hexutil.Bytes(nil)
	EvRC1 := hexutil.Bytes(nil)
	EvRC2 := hexutil.Bytes(nil)
	_CmS := hexutil.Bytes(nil)
	_CmR := hexutil.Bytes(nil)
	CMsFPC := hexutil.Bytes(nil)
	CMsFPZ1 := hexutil.Bytes(nil)
	CMsFPZ2 := hexutil.Bytes(nil)
	CMrFPC := hexutil.Bytes(nil)
	CMrFPZ1 := hexutil.Bytes(nil)
	CMrFPZ2 := hexutil.Bytes(nil)
	EvsBsC1 := hexutil.Bytes(nil)
	EvsBsC2 := hexutil.Bytes(nil)
	EvOC1 := hexutil.Bytes(nil)
	EvOC2 := hexutil.Bytes(nil)
	_CmO := hexutil.Bytes(nil)
	EvOEPs0 := hexutil.Bytes(nil)
	EvOEPs1 := hexutil.Bytes(nil)
	EvOEPs2 := hexutil.Bytes(nil)
	EvOEPs3 := hexutil.Bytes(nil)
	EvOEPt := hexutil.Bytes(nil)
	BPC := hexutil.Bytes(nil)
	BPRV := hexutil.Bytes(nil)
	BPRR := hexutil.Bytes(nil)
	BPSV := hexutil.Bytes(nil)
	BPSR := hexutil.Bytes(nil)
	BPSOr := hexutil.Bytes(nil)
	if args.To == nil {
		return types.NewContractCreation(uint64(*args.Nonce), (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input, 1, &ErpkC1, &ErpkC2, &EspkC1, &EspkC2, &CMRpk, &CMSpk, &ErpkEPs0, &ErpkEPs1, &ErpkEPs2, &ErpkEPs3, &ErpkEPt, &EspkEPs0, &EspkEPs1, &EspkEPs2, &EspkEPs3, &EspkEPt, &EvSC1, &EvSC2, &EvRC1, &EvRC2, &_CmS, &_CmR, &CMsFPC, &CMsFPZ1, &CMsFPZ2, &CMrFPC, &CMrFPZ1, &CMrFPZ2, &EvsBsC1, &EvsBsC2, &EvOC1, &EvOC2, &_CmO, &EvOEPs0, &EvOEPs1, &EvOEPs2, &EvOEPs3, &EvOEPt, &BPC, &BPRV, &BPRR, &BPSV, &BPSR, &BPSOr, args.EpkrC1, args.EpkrC2, args.EpkpC1, args.EpkpC2, args.SigM, args.SigMHash, args.SigR, args.SigS, args.CmV), nil
	}
	return types.NewTransaction(uint64(*args.Nonce), *args.To, (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input, 1, &ErpkC1, &ErpkC2, &EspkC1, &EspkC2, &CMRpk, &CMSpk, &ErpkEPs0, &ErpkEPs1, &ErpkEPs2, &ErpkEPs3, &ErpkEPt, &EspkEPs0, &EspkEPs1, &EspkEPs2, &EspkEPs3, &EspkEPt, &EvSC1, &EvSC2, &EvRC1, &EvRC2, &_CmS, &_CmR, &CMsFPC, &CMsFPZ1, &CMsFPZ2, &CMrFPC, &CMrFPZ1, &CMrFPZ2, &EvsBsC1, &EvsBsC2, &EvOC1, &EvOC2, &_CmO, &EvOEPs0, &EvOEPs1, &EvOEPs2, &EvOEPs3, &EvOEPt, &BPC, &BPRV, &BPRR, &BPSV, &BPSR, &BPSOr, args.EpkrC1, args.EpkrC2, args.EpkpC1, args.EpkpC2, args.SigM, args.SigMHash, args.SigR, args.SigS, args.CmV), nil
}

func (args *SendTxArgs) toTransaction() (*types.Transaction, error) {
	/*rpk, err := paraPK(*args.Rpk)
	if err != nil {
		return nil, err
	}
	spk, err := paraPK(*args.Spk)
	if err != nil {
		return nil, err
	}
	rpk := []byte(*args.Rpk)
	spk := []byte(*args.Spk)

	fmt.Println(rpk)
	fmt.Println(spk)*/
	/*var input []byte
	if args.Input != nil {
		input = *args.Input
	} else if args.Data != nil {
		input = *args.Data
	}
	spk := []byte(*args.Spk)
	rpk := []byte(*args.Rpk)
	if args.To == nil {
		return types.NewContractCreation(uint64(*args.Nonce), (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input, uint64(*args.SnO), uint64(*args.Rr1), uint64(*args.CmSpk), uint64(*args.CmRpk), uint64(*args.CmO), uint64(*args.CmS), uint64(*args.CmR), uint64(*args.EvR), uint64(*args.EvR0), uint64(*args.EvR_), uint64(*args.EvR_0), uint64(*args.PI), uint64(*args.ID), *args.Sig, uint64(*args.CmV), uint64(*args.EpkV))
	}
	return types.NewTransaction(uint64(*args.Nonce), *args.To, (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input, uint64(*args.SnO), uint64(*args.Rr1), uint64(*args.CmSpk), uint64(*args.CmRpk), uint64(*args.CmO), uint64(*args.CmS), uint64(*args.CmR), uint64(*args.EvR), uint64(*args.EvR0), uint64(*args.EvR_), uint64(*args.EvR_0), uint64(*args.PI), uint64(*args.ID), *args.Sig, uint64(*args.CmV), uint64(*args.EpkV))
	*/
	return nil, nil
}

// SubmitTransaction is a helper function that submits tx to txPool and logs a message.
func SubmitTransaction(ctx context.Context, b Backend, tx *types.Transaction) (common.Hash, error) {
	if err := b.SendTx(ctx, tx); err != nil {
		return common.Hash{}, err
	}
	if tx.To() == nil {
		signer := types.MakeSigner(b.ChainConfig(), b.CurrentBlock().Number())
		from, err := types.Sender(signer, tx)
		if err != nil {
			return common.Hash{}, err
		}
		addr := crypto.CreateAddress(from, tx.Nonce())
		log.Info("Submitted contract creation", "fullhash", tx.Hash().Hex(), "contract", addr.Hex())
	} else {
		log.Info("Submitted transaction", "fullhash", tx.Hash().Hex(), "recipient", tx.To())
	}
	return tx.Hash(), nil
}

// SendTransaction creates a transaction for the given argument, sign it and submit it to the
// transaction pool.
// 从Postman发出的交易从此处开始
func (s *PublicTransactionPoolAPI) SendTransaction(ctx context.Context, args SendTxArgs) (common.Hash, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.From}
	// 拿到发出账户的钱包，即使账户未解锁也能拿到
	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return common.Hash{}, err
	}
	if args.Nonce == nil {
		// 如果没有提前声明nonce，则在签名前保持此地址的互斥锁，防止同一nonce同时分配给了多个账户
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.From)
		defer s.nonceLock.UnlockAddr(args.From)
	}

	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b); err != nil {
		return common.Hash{}, err
	}
	// 检查发来的值够不够
	err = args.chechParameter()
	if err != nil {
		return common.Hash{}, err
	}
	// Assemble the transaction and sign with the wallet
	if *args.ID == 0x0 {
		tx, err := args.toZeroTransaction(s.b.RegulatorKey())
		if err != nil {
			return common.Hash{}, err
		}
		signed, err := wallet.SignTx(account, tx, s.b.ChainConfig().ChainID)
		if err != nil {
			return common.Hash{}, err
		}
		return SubmitTransaction(ctx, s.b, signed)
	} else if *args.ID == 0x1 {
		tx, err := args.toExTransaction(s.b.RegulatorKey())
		if err != nil {
			return common.Hash{}, err
		}
		signed, err := wallet.SignTx(account, tx, s.b.ChainConfig().ChainID)
		if err != nil {
			return common.Hash{}, err
		}
		return SubmitTransaction(ctx, s.b, signed)
	} else {
		return common.Hash{}, err
	}
}

// FillTransaction fills the defaults (nonce, gas, gasPrice) on a given unsigned transaction,
// and returns it to the caller for further processing (signing + broadcast)
func (s *PublicTransactionPoolAPI) FillTransaction(ctx context.Context, args SendTxArgs) (*SignTransactionResult, error) {
	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b); err != nil {
		return nil, err
	}
	// Assemble the transaction and obtain rlp
	tx, err := args.toTransaction()
	if err != nil {
		return nil, err
	}
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}
	return &SignTransactionResult{data, tx}, nil
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicTransactionPoolAPI) SendRawTransaction(ctx context.Context, encodedTx hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return common.Hash{}, err
	}
	return SubmitTransaction(ctx, s.b, tx)
}

// Sign calculates an ECDSA signature for:
// keccack256("\x19Ethereum Signed Message:\n" + len(message) + message).
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The account associated with addr must be unlocked.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_sign
func (s *PublicTransactionPoolAPI) Sign(addr common.Address, data hexutil.Bytes) (hexutil.Bytes, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Sign the requested hash with the wallet
	signature, err := wallet.SignText(account, data)
	if err == nil {
		signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}
	return signature, err
}

// SignTransactionResult represents a RLP encoded signed transaction.
type SignTransactionResult struct {
	Raw hexutil.Bytes      `json:"raw"`
	Tx  *types.Transaction `json:"tx"`
}

// SignTransaction will sign the given transaction with the from account.
// The node needs to have the private key of the account corresponding with
// the given from address and it needs to be unlocked.
func (s *PublicTransactionPoolAPI) SignTransaction(ctx context.Context, args SendTxArgs) (*SignTransactionResult, error) {
	if args.Gas == nil {
		return nil, fmt.Errorf("gas not specified")
	}
	if args.GasPrice == nil {
		return nil, fmt.Errorf("gasPrice not specified")
	}
	if args.Nonce == nil {
		return nil, fmt.Errorf("nonce not specified")
	}
	if err := args.setDefaults(ctx, s.b); err != nil {
		return nil, err
	}
	formedTx, err := args.toTransaction()
	if err != nil {
		return nil, err
	}
	tx, err := s.sign(args.From, formedTx)
	if err != nil {
		return nil, err
	}
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}
	return &SignTransactionResult{data, tx}, nil
}

// PendingTransactions returns the transactions that are in the transaction pool
// and have a from address that is one of the accounts this node manages.
func (s *PublicTransactionPoolAPI) PendingTransactions() ([]*RPCTransaction, error) {
	pending, err := s.b.GetPoolTransactions()
	if err != nil {
		return nil, err
	}
	accounts := make(map[common.Address]struct{})
	for _, wallet := range s.b.AccountManager().Wallets() {
		for _, account := range wallet.Accounts() {
			accounts[account.Address] = struct{}{}
		}
	}
	transactions := make([]*RPCTransaction, 0, len(pending))
	for _, tx := range pending {
		var signer types.Signer = types.HomesteadSigner{}
		if tx.Protected() {
			signer = types.NewEIP155Signer(tx.ChainId())
		}
		from, _ := types.Sender(signer, tx)
		if _, exists := accounts[from]; exists {
			transactions = append(transactions, newRPCPendingTransaction(tx))
		}
	}
	return transactions, nil
}

// Resend accepts an existing transaction and a new gas price and limit. It will remove
// the given transaction from the pool and reinsert it with the new gas price and limit.
func (s *PublicTransactionPoolAPI) Resend(ctx context.Context, sendArgs SendTxArgs, gasPrice *hexutil.Big, gasLimit *hexutil.Uint64) (common.Hash, error) {
	if sendArgs.Nonce == nil {
		return common.Hash{}, fmt.Errorf("missing transaction nonce in transaction spec")
	}
	if err := sendArgs.setDefaults(ctx, s.b); err != nil {
		return common.Hash{}, err
	}
	matchTx, err := sendArgs.toTransaction()
	if err != nil {
		return common.Hash{}, err
	}
	pending, err := s.b.GetPoolTransactions()
	if err != nil {
		return common.Hash{}, err
	}

	for _, p := range pending {
		var signer types.Signer = types.HomesteadSigner{}
		if p.Protected() {
			signer = types.NewEIP155Signer(p.ChainId())
		}
		wantSigHash := signer.Hash(matchTx)

		if pFrom, err := types.Sender(signer, p); err == nil && pFrom == sendArgs.From && signer.Hash(p) == wantSigHash {
			// Match. Re-sign and send the transaction.
			if gasPrice != nil && (*big.Int)(gasPrice).Sign() != 0 {
				sendArgs.GasPrice = gasPrice
			}
			if gasLimit != nil && *gasLimit != 0 {
				sendArgs.Gas = gasLimit
			}
			formedTx, err := sendArgs.toTransaction()
			if err != nil {
				return common.Hash{}, err
			}
			signedTx, err := s.sign(sendArgs.From, formedTx)
			if err != nil {
				return common.Hash{}, err
			}
			if err = s.b.SendTx(ctx, signedTx); err != nil {
				return common.Hash{}, err
			}
			return signedTx.Hash(), nil
		}
	}

	return common.Hash{}, fmt.Errorf("transaction %#x not found", matchTx.Hash())
}

// PublicDebugAPI is the collection of Ethereum APIs exposed over the public
// debugging endpoint.
type PublicDebugAPI struct {
	b Backend
}

// NewPublicDebugAPI creates a new API definition for the public debug methods
// of the Ethereum service.
func NewPublicDebugAPI(b Backend) *PublicDebugAPI {
	return &PublicDebugAPI{b: b}
}

// GetBlockRlp retrieves the RLP encoded for of a single block.
func (api *PublicDebugAPI) GetBlockRlp(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", encoded), nil
}

// TestSignCliqueBlock fetches the given block number, and attempts to sign it as a clique header with the
// given address, returning the address of the recovered signature
//
// This is a temporary method to debug the externalsigner integration,
// TODO: Remove this method when the integration is mature
func (api *PublicDebugAPI) TestSignCliqueBlock(ctx context.Context, address common.Address, number uint64) (common.Address, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return common.Address{}, fmt.Errorf("block #%d not found", number)
	}
	header := block.Header()
	header.Extra = make([]byte, 32+65)
	encoded := clique.CliqueRLP(header)

	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: address}
	wallet, err := api.b.AccountManager().Find(account)
	if err != nil {
		return common.Address{}, err
	}

	signature, err := wallet.SignData(account, accounts.MimetypeClique, encoded)
	if err != nil {
		return common.Address{}, err
	}
	sealHash := clique.SealHash(header).Bytes()
	log.Info("test signing of clique block",
		"Sealhash", fmt.Sprintf("%x", sealHash),
		"signature", fmt.Sprintf("%x", signature))
	pubkey, err := crypto.Ecrecover(sealHash, signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	return signer, nil
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (api *PublicDebugAPI) PrintBlock(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return spew.Sdump(block), nil
}

// SeedHash retrieves the seed hash of a block.
func (api *PublicDebugAPI) SeedHash(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return fmt.Sprintf("0x%x", ethash.SeedHash(number)), nil
}

// PrivateDebugAPI is the collection of Ethereum APIs exposed over the private
// debugging endpoint.
type PrivateDebugAPI struct {
	b Backend
}

// NewPrivateDebugAPI creates a new API definition for the private debug methods
// of the Ethereum service.
func NewPrivateDebugAPI(b Backend) *PrivateDebugAPI {
	return &PrivateDebugAPI{b: b}
}

// ChaindbProperty returns leveldb properties of the key-value database.
func (api *PrivateDebugAPI) ChaindbProperty(property string) (string, error) {
	if property == "" {
		property = "leveldb.stats"
	} else if !strings.HasPrefix(property, "leveldb.") {
		property = "leveldb." + property
	}
	return api.b.ChainDb().Stat(property)
}

// ChaindbCompact flattens the entire key-value database into a single level,
// removing all unused slots and merging all keys.
func (api *PrivateDebugAPI) ChaindbCompact() error {
	for b := byte(0); b < 255; b++ {
		log.Info("Compacting chain database", "range", fmt.Sprintf("0x%0.2X-0x%0.2X", b, b+1))
		if err := api.b.ChainDb().Compact([]byte{b}, []byte{b + 1}); err != nil {
			log.Error("Database compaction failed", "err", err)
			return err
		}
	}
	return nil
}

// SetHead rewinds the head of the blockchain to a previous block.
func (api *PrivateDebugAPI) SetHead(number hexutil.Uint64) {
	api.b.SetHead(uint64(number))
}

// PublicNetAPI offers network related RPC methods
type PublicNetAPI struct {
	net            *p2p.Server
	networkVersion uint64
}

// NewPublicNetAPI creates a new net API instance.
func NewPublicNetAPI(net *p2p.Server, networkVersion uint64) *PublicNetAPI {
	return &PublicNetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *PublicNetAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers
func (s *PublicNetAPI) PeerCount() hexutil.Uint {
	return hexutil.Uint(s.net.PeerCount())
}

// Version returns the current ethereum protocol version.
func (s *PublicNetAPI) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}
