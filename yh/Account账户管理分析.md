# Account账户管理分析

eth.sendTransaction({
from: "0xf8a4909ce93a9d876b8f787e4771d87d6899d879", 
to: "0x72b92aebbd254f808cef0afbf5c96e7ae681cfda", value: web3.toWei(100, "ether"
)})这是web3j的转账方式。只要有一个合法的以太坊的地址就能够在以太坊区块链上进行交易，地址是交易最基本的单位。地址太长，且无规律，易丢失或忘记。所以账户Account就是管理它的一个基本数据结构。一个人可能会有多个账户，所以需要钱包Wallet来管理（类似于真实的世界，你多个银行户头号码，对应着多个银行卡，然后银行卡放到钱包中管理，为了管理多个钱包，又需要有一个皮包Manager。该模块还提供了一个后台钱包provider，用来动态提供一批账号。



### personal.newAccount()方法实现

当你在控制台输入personal.newAccount(),创建一个新的账户命令的流程如下：

执行Internal/ethapi/api.go中的NewAccount方法，该方法会返回一个地址

```go
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
```

api.go文件中的NewAccount方法调用fetchKeystore方法从账户管理器(manager)检索加密的密钥存储库获取keystore

fetchKeystore同样在ImportRawKey,LockAccount,UnlockAccount中有使用

```go
func fetchKeystore(am *accounts.Manager) *keystore.KeyStore {
	return am.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
}
```

api.go中的NewAccount方法获取到keystore后通过keystore调用keystore.go中的NewAccount方法获取account,并将这个账户添加到keystore中，返回account

```go
// NewAccount generates a new key and stores it into the key directory,
// encrypting it with the passphrase.
func (ks *KeyStore) NewAccount(passphrase string) (accounts.Account, error) {
	_, account, err := storeNewKey(ks.storage, crand.Reader, passphrase)
	if err != nil {
		return accounts.Account{}, err
	}
	// Add the account to the cache immediately rather
	// than waiting for file system notifications to pick it up.
	ks.cache.add(account)
	ks.refreshWallets()
	return account, nil
}
```

调用storeNewKey方法创建一个新的账户，生成一对公私钥，通过私钥以及地址构建一个账户

```go
func storeNewKey(ks keyStore, rand io.Reader, auth string) (*Key, accounts.Account, error) {
	key, err := newKey(rand)
	if err != nil {
		return nil, accounts.Account{}, err
	}
	a := accounts.Account{
		Address: key.Address,
		URL:     accounts.URL{Scheme: KeyStoreScheme, Path: ks.JoinPath(keyFileName(key.Address))},
	}
	if err := ks.StoreKey(a.URL.Path, key, auth); err != nil {
		zeroKey(key.PrivateKey)
		return nil, a, err
	}
	return key, a, err
}

```

Key的生成函数，通过椭圆曲线加密生成私钥，生成Key

```go
func newKey(rand io.Reader) (*Key, error) {
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand)
	if err != nil {
		return nil, err
	}
	return newKeyFromECDSA(privateKeyECDSA), nil
}
```

生成公钥和私钥对,`ecdsa.GenerateKey(crypto.S256(), rand)` 以太坊采用了椭圆曲线数字签名算法（ECDSA）生成一对公私钥，并选择的是secp256k1曲线

```go
// GenerateKey generates a public and private key pair.
func GenerateKey(c elliptic.Curve, rand io.Reader) (*PrivateKey, error) {
	k, err := randFieldElement(c, rand)
	if err != nil {
		return nil, err
	}

	priv := new(PrivateKey)
	priv.PublicKey.Curve = c
	priv.D = k
	priv.PublicKey.X, priv.PublicKey.Y = c.ScalarBaseMult(k.Bytes())
	return priv, nil
}
func randFieldElement(c elliptic.Curve, rand io.Reader) (k *big.Int, err error) {
	params := c.Params()
	b := make([]byte, params.BitSize/8+8)
	_, err = io.ReadFull(rand, b)
	if err != nil {
		return
	}

	k = new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, one)
	k.Mod(k, n)
	k.Add(k, one)
	return
}
```

以太坊使用私钥通过 ECDSA算法推导出公钥，继而经过 Keccak-256 单向散列函数推导出地址

```go
func newKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *Key {
	id := uuid.NewRandom()
	key := &Key{
		Id:         id,
		Address:    crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey: privateKeyECDSA,
	}
	return key
}
```

整个过程为：

从前控制台传入创建账户命令

首先创建随机私钥

通过私钥导出公钥

通过公私钥导出地址

### personal.listAccounts列出所有账户方法

控制台执行该命令时，会执行Internal/ethapi/api.go中的listAccount方法，该方法会从用户管理读取所有钱包信息，返回所有注册钱包下的所有地址信息

```go
该代码块位于Uiapi.go中
// List available accounts. As opposed to the external API definition, this method delivers
// the full Account object and not only Address.
// Example call
// {"jsonrpc":"2.0","method":"clef_listAccounts","params":[], "id":4}
func (s *UIServerAPI) ListAccounts(ctx context.Context) ([]accounts.Account, error) {
	var accs []accounts.Account
	for _, wallet := range s.am.Wallets() {
		accs = append(accs, wallet.Accounts()...)
	}
	return accs, nil
}
该代码位于api.go中
func (s *PrivateAccountAPI) ListAccounts() []common.Address {
	return s.am.Accounts()
}

```

### eth.sendTranscation方法实现

 调用api,go中的SendTransaction方法

```go
// SendTransaction creates a transaction for the given argument, sign it and submit it to the
// transaction pool.
func (s *PublicTransactionPoolAPI) SendTransaction(ctx context.Context, args SendTxArgs) (common.Hash, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.From}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return common.Hash{}, err
	}
  //Nonce防止双花攻击
	if args.Nonce == nil {
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.From)
		defer s.nonceLock.UnlockAddr(args.From)
	}

	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b); err != nil {
		return common.Hash{}, err
	}
	// Assemble the transaction and sign with the wallet
	tx := args.toTransaction()

	signed, err := wallet.SignTx(account, tx, s.b.ChainConfig().ChainID)
	if err != nil {
		return common.Hash{}, err
	}
	return SubmitTransaction(ctx, s.b, signed)
}
```

利用传入的参数from构造一个account，表示转出方。然后通过accountManager获得am.Find方法从账户管理系统中对钱包进行遍历，找到包含这个account的钱包

```go
// Find attempts to locate the wallet corresponding to a specific account. Since
// accounts can be dynamically added to and removed from wallets, this method has
// a linear runtime in the number of wallets.
func (am *Manager) Find(account Account) (Wallet, error) {
	am.lock.RLock()
	defer am.lock.RUnlock()

	for _, wallet := range am.wallets {
		if wallet.Contains(account) {
			return wallet, nil
		}
	}
	return nil, ErrUnknownAccount
}
```

调用setDefaults方法设置一些交易的默认值。如果没有设置Gas，GasPrice，Nonce等，那么它们将会被设置为默认值。

```go
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
```

参数设置好后，利用toTransaction方法创建一笔交易

```go
func (args *SendTxArgs) toTransaction() *types.Transaction {
	var input []byte
	if args.Input != nil {
		input = *args.Input
	} else if args.Data != nil {
		input = *args.Data
	}
	if args.To == nil {
		return types.NewContractCreation(uint64(*args.Nonce), (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input)
	}
	return types.NewTransaction(uint64(*args.Nonce), *args.To, (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input)
}
```

对传入的交易信息to参数进行判断。如果没有to值，这是一笔合约转账；有to值，就是发起的一笔转账。代码调用NewTransaction创建一笔交易信息。

```go
func newTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
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
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}

	return &Transaction{data: d}
}
```

填充交易结构体中的一些参数，来创建一个交易。到这，交易已经创建成功了。

对交易进行签名来确保交易的真实有效。

```go
func (ks *KeyStore) SignTx(a accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	// Look up the key to sign with and abort if it cannot be found
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	unlockedKey, found := ks.unlocked[a.Address]
	if !found {
		return nil, ErrLocked
	}
	// Depending on the presence of the chain ID, sign with EIP155 or homestead
	if chainID != nil {
		return types.SignTx(tx, types.NewEIP155Signer(chainID), unlockedKey.PrivateKey)
	}
	return types.SignTx(tx, types.HomesteadSigner{}, unlockedKey.PrivateKey)
}
```

首先验证账户是否已经解锁。若没有解锁，则直接异常退出。然后检查chainID，判断使用哪一种签名的方式，调用signTx进行签名。

```go
// SignTx signs the transaction using the given signer and private key
func SignTx(tx *Transaction, s Signer, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}
```

在签名时，首先获取交易的RLP哈希值，然后用传入的私钥进行椭圆加密。接着调用WithSignature方法进行初始化。进行到这里，我们交易的签名已经完成，并且封装成为一个带签名的交易。然后，我们就需要将这笔交易提交出去。调用SubmitTransaction方法提交交易。

```go
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
```

submitTransaction方法会将交易发送给backend进行处理，返回经过签名后的交易的hash值。

### geth account new 实现

运行命令：geth account new 

程序入口在cmd/geth/main.go

```go
func init() {
	// Initialize the CLI app and start Geth
	app.Action = geth
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2013-2020 The go-ethereum Authors"
	app.Commands = []cli.Command{
		// See chaincmd.go:
		initCommand,
		importCommand,
		exportCommand,
		importPreimagesCommand,
		exportPreimagesCommand,
		copydbCommand,
		removedbCommand,
		dumpCommand,
		inspectCommand,
		// See accountcmd.go:
		accountCommand,
		walletCommand,
        .......

}
```

账户相关的命令在accountcmd.go里，新建账户命令为new:

```go
var (
	walletCommand = cli.Command{
		Name:      "wallet",
		Usage:     "Manage Ethereum presale wallets",
		ArgsUsage: "",
		Category:  "ACCOUNT COMMANDS",
		Description: `
    geth wallet import /path/to/my/presale.wallet
will prompt for your password and imports your ether presale account.
It can be used non-interactively with the --password option taking a
passwordfile as argument containing the wallet password in plaintext.`,
		Subcommands: []cli.Command{
			{

				Name:      "import",
				Usage:     "Import Ethereum presale wallet",
				ArgsUsage: "<keyFile>",
				Action:    utils.MigrateFlags(importWallet),
				Category:  "ACCOUNT COMMANDS",
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
					utils.PasswordFileFlag,
					utils.LightKDFFlag,
				},
				Description: `
	geth wallet [options] /path/to/my/presale.wallet
will prompt for your password and imports your ether presale account.
It can be used non-interactively with the --password option taking a
passwordfile as argument containing the wallet password in plaintext.`,
			},
		},
	}

	accountCommand = cli.Command{
		Name:     "account",
		Usage:    "Manage accounts",
		Category: "ACCOUNT COMMANDS",
		Description: `
Manage accounts, list all existing accounts, import a private key into a new
account, create a new account or update an existing account.
It supports interactive mode, when you are prompted for password as well as
non-interactive mode where passwords are supplied via a given password file.
Non-interactive mode is only meant for scripted use on test networks or known
safe environments.
Make sure you remember the password you gave when creating a new account (with
either new or import). Without it you are not able to unlock your account.
Note that exporting your key in unencrypted format is NOT supported.
Keys are stored under <DATADIR>/keystore.
It is safe to transfer the entire directory or the individual keys therein
between ethereum nodes by simply copying.
Make sure you backup your keys regularly.`,
		Subcommands: []cli.Command{
			{
				Name:   "list",
				Usage:  "Print summary of existing accounts",
				Action: utils.MigrateFlags(accountList),
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
				},
				Description: `
Print a short summary of all accounts`,
			},
			{
				Name:   "new",
				Usage:  "Create a new account",
				Action: utils.MigrateFlags(accountCreate),
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
					utils.PasswordFileFlag,
					utils.LightKDFFlag,
				},
				Description: `
    geth account new
Creates a new account and prints the address.
The account is saved in encrypted format, you are prompted for a password.
You must remember this password to unlock your account in the future.
For non-interactive use the password can be specified with the --password flag:
Note, this is meant to be used for testing only, it is a bad idea to save your
password to file or expose in any other way.
`,
			},
			{
				Name:      "update",
				Usage:     "Update an existing account",
				Action:    utils.MigrateFlags(accountUpdate),
				ArgsUsage: "<address>",
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
					utils.LightKDFFlag,
				},
				Description: `
    geth account update <address>
Update an existing account.
The account is saved in the newest version in encrypted format, you are prompted
for a password to unlock the account and another to save the updated file.
This same command can therefore be used to migrate an account of a deprecated
format to the newest format or change the password for an account.
For non-interactive use the password can be specified with the --password flag:
    geth account update [options] <address>
Since only one password can be given, only format update can be performed,
changing your password is only possible interactively.
`,
			},
			{
				Name:   "import",
				Usage:  "Import a private key into a new account",
				Action: utils.MigrateFlags(accountImport),
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
					utils.PasswordFileFlag,
					utils.LightKDFFlag,
				},
				ArgsUsage: "<keyFile>",
				Description: `
    geth account import <keyfile>
Imports an unencrypted private key from <keyfile> and creates a new account.
Prints the address.
The keyfile is assumed to contain an unencrypted private key in hexadecimal format.
The account is saved in encrypted format, you are prompted for a password.
You must remember this password to unlock your account in the future.
For non-interactive use the password can be specified with the -password flag:
    geth account import [options] <keyfile>
Note:
As you can directly copy your encrypted accounts to another ethereum instance,
this import mechanism is not needed when you transfer an account between
nodes.
`,
			},
		},
	}
)
```

new一个新账户的时候，会调用accountCreate：

```go
// accountCreate creates a new account into the keystore defined by the CLI flags.
func accountCreate(ctx *cli.Context) error {
	cfg := gethConfig{Node: defaultNodeConfig()}
	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}
	utils.SetNodeConfig(ctx, &cfg.Node)
	scryptN, scryptP, keydir, err := cfg.Node.AccountConfig()

	if err != nil {
		utils.Fatalf("Failed to read configuration: %v", err)
	}

	password := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, utils.MakePasswordList(ctx))

	account, err := keystore.StoreKey(keydir, password, scryptN, scryptP)

	if err != nil {
		utils.Fatalf("Failed to create account: %v", err)
	}
	fmt.Printf("\nYour new key was generated\n\n")
	fmt.Printf("Public address of the key:   %s\n", account.Address.Hex())
	fmt.Printf("Path of the secret key file: %s\n\n", account.URL.Path)
	fmt.Printf("- You can share your public address with anyone. Others need it to interact with you.\n")
	fmt.Printf("- You must NEVER share the secret key with anyone! The key controls access to your funds!\n")
	fmt.Printf("- You must BACKUP your key file! Without the key, it's impossible to access account funds!\n")
	fmt.Printf("- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!\n\n")
	return nil
}
```

**accountCreate**分为三个步骤，**其中最关键的为第三步**

1. 获取配置
2. 解析用户密码
3. 生成地址

第三步生成地址调用的keystore.StoreKey,程序在keystore_passpharse.go

```go
func StoreKey(dir, auth string, scryptN, scryptP int) (accounts.Account, error) {
	_, a, err := storeNewKey(&keyStorePassphrase{dir, scryptN, scryptP, false}, rand.Reader, auth)
	return a, err
}
```

调用了key.go里面的storeNewKey创建新账户

```
func storeNewKey(ks keyStore, rand io.Reader, auth string) (*Key, accounts.Account, error) {
	key, err := newKey(rand)
	if err != nil {
		return nil, accounts.Account{}, err
	}
	a := accounts.Account{
		Address: key.Address,
		URL:     accounts.URL{Scheme: KeyStoreScheme, Path: ks.JoinPath(keyFileName(key.Address))},
	}
	if err := ks.StoreKey(a.URL.Path, key, auth); err != nil {
		zeroKey(key.PrivateKey)
		return nil, a, err
	}
	return key, a, err
}
```

newKey创建新账户时，

1. 由secp256k1曲线生成私钥，是由32字节随机数组成

2. 采用椭圆曲线数字签名算法（ECDSA）将私钥映射成公钥，一个私钥只能映射出一个公钥。

3. 然后由公钥算出地址并构建一个自定义的Key

   通过公钥算出地址并构建一个自定义的Key

   ```go
   func newKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *Key {
   	id := uuid.NewRandom()
   	key := &Key{
   		Id:         id,
           //由公钥推出地址
   		Address:    crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
   		PrivateKey: privateKeyECDSA,
   	}
   	return key
   }
   ```

   由公钥算出地址是由crypto.PubkeytoAddress完成

   ```go
   func PubkeyToAddress(p ecdsa.PublicKey) common.Address {
       // (1) 将pubkey转换为字节序列
   	pubBytes := FromECDSAPub(&p)
       // (2) pubBytes为04 开头的65字节公钥,去掉04后剩下64字节进行Keccak256运算
   	// (3) 经过Keccak256运算后变成32字节，最终取这32字节的后20字节作为真正的地址
   	return common.BytesToAddress(Keccak256(pubBytes[1:])[12:])
   }
   
   // Keccak256 calculates and returns the Keccak256 hash of the input data.
   func Keccak256(data ...[]byte) []byte {
   	d := sha3.NewLegacyKeccak256()
   	for _, b := range data {
   		d.Write(b)
   	}
   	return d.Sum(nil)
   }
   ```

   公钥（64字节）经过Keccak-256单向散列函数变成了32字节，然后取后20字节作为地址。本质上是从32字节的私钥映射到20字节的公共地址。这意味着一个账户可以有不止一个私钥。

以太坊地址的生成过程：

1. 由secp256k1曲线生成私钥，是由32字节的随机数生成
2. 采用椭圆曲线数字签名算法（ECDSA）将私钥（32字节）映射成公钥（65字节）。
3. 公钥（去掉04后剩下64字节）经过Keccak-256单向散列函数变成了32字节，然后取后20字节作为地址

### keyStorePassphrase{}

keystore接口的实现类，它实现了以Web3 Secret Storage加密方法为公钥密钥信息进行加密管理。

本地文件的存储都是JSON格式的，使用时方便许多

key的结构体：

```go
type Key struct {
	Id uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address common.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
}
```

最后一个成员变量，私钥，其中有一个PublicKey的成员变量和一个Address成员类型变量形成映射对。不需要来回经常切换。以太坊的钱包在创建密码时正是对应着passphrase。

personal.newAccount("123456")"0x00fe1b8a035b5c5e42249627ea62f75e5a071cb3"// 或personal.newAccount()Passphrase:Repeat passphrase:"0x6a787f16c2037826fbc112c337d7b571bb19c022"12345678910

![](C:\Users\43798\Desktop\20180707190939425.png)



### stateDB & stateObject

以太坊账户管理中，stateObject表示一个账户的动态变化，结构中的关键字段如下：

```go
type stateObject struct {
	address  common.Address
	addrHash common.Hash // hash of ethereum address of the account
	data     Account
	db       *StateDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access
	code Code // contract bytecode, which gets set when code is loaded

	originStorage  Storage // Storage cache of original entries to dedup rewrites, reset for every transaction
	pendingStorage Storage // Storage entries that need to be flushed to disk, at the end of an entire block
	dirtyStorage   Storage // Storage entries that have been modified in the current transaction execution
	fakeStorage    Storage // Fake storage which constructed by caller for debugging purpose.

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	dirtyCode bool // true if the code was updated
	suicided  bool
	deleted   bool
}

```

address为账户的160bits地址

data为账户的信息，即前面提到的Account结构

trie合约账户的存储空间的缓存，我们可以从由**data**的**Root**从底层数据库中读取这棵树

code合约代码的缓存，作用和trie类似 

stateDB表示所有账户的动态变化，即它管理的stateObject，结构中的关键字段如下:

```go
type StateDB struct {
	db   Database
	trie Trie

	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects        map[common.Address]*stateObject
	stateObjectsPending map[common.Address]struct{} // State objects finalized but not yet written to the trie
	stateObjectsDirty   map[common.Address]struct{} // State objects modified in the current execution

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// The refund counter, also used by state transitioning.
	refund uint64

	thash, bhash common.Hash
	txIndex      int
	logs         map[common.Hash][]*types.Log
	logSize      uint

	preimages map[common.Hash][]byte

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        *journal
	validRevisions []revision
	nextRevisionId int

	// Measurements gathered during execution for debugging purposes
	AccountReads   time.Duration
	AccountHashes  time.Duration
	AccountUpdates time.Duration
	AccountCommits time.Duration
	StorageReads   time.Duration
	StorageHashes  time.Duration
	StorageUpdates time.Duration
	StorageCommits time.Duration
}
```

db以太坊底层数据库结构，账户的信息都是从数据库中读取

trie所有账户组织而成的MPT树的实例，从它里面可以读取以太坊所有账户

stateObjects管理的所有需要修改的stateObject

**账户操作**：

1. 在执行区块中的交易时，我们可能需要修改某些账户的信息(比如增减余额，或者修改合约账户代码) ，这时我们按以下步骤进行操作

2. 从stateDB找到账户对应的stateObject，若不存在，则从trie树中，通过读取底层数据库构建新的stateObject，访问过的stateObject会缓存起来
   对stateObject账户进行操作，可能会涉及对余额的操作，如AddBalance()调用，也有可能对存储空间的操作，如SetState()，或者对合约代码的操作如 SetCode()
3. 在区块构建完成时，计算每个账户新的MPT树的各个节点Hash，并存入数据库，完成修改。
   

这一部分还没有详细的去看，只是进行了初步的了解