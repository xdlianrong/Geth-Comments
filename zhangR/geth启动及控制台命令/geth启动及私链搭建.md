# geth启动及私链搭建

1、在go-ethereum文件夹下打开终端，执行 make geth；在build/bin/下可找到geth文件，在该目录下执行geth -version，返回如下结果即为成功。（可将geth添加至环境变量中）

![geth_version](D:\Destop\geth启动及控制台命令\image\geth_version.png)

2、 新建一个文件夹，再新建一个genesis.json文件作为创世区块。

```javascript
{
  "config": {
    "chainId": 666,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "ethash": {}
  },
  "nonce": "0x0",
  "timestamp": "0x5ddf8f3e",
  "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": "0x47b760",
  "difficulty": "0x00002",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "alloc": { },
  "number": "0x0",
  "gasUsed": "0x0",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000"
}
```

​	可以在alloc中添加初始用户信息，格式如下

```javascript
"alloc":{
    "dbdbdb2cbd23b783741e8d7fcf51e459b497e4a6":
{
"balance":"100000000000000000000000000000"
}
}
```

3、 在该文件夹下打开终端，执行 geth --datadir mychain2 init genesis.json  初始化，初始化成功后，会在mychain2文件夹下生成两个文件夹 geth和keystore。 其中geth/chaindata中存放的是区块数据，keystore中存放的是账户数据。 

<img src="D:\Destop\geth启动及控制台命令\image\geth_init.png" alt="1584808551895"  />

4、启动节点或开发测试节点（-dev  自带有balance的开发者账户），执行 geth --datadir mychain2 --nodiscover --networkid 666 console 2>>output.log ，将输出日志保存至output.log文件中，避免干扰。

![1584809089834](D:\Destop\geth启动及控制台命令\image\geth_console.png)

同时可开启第二个终端，执行 tail -f output.log 同步查看日志信息。

![1584809327261](D:\Destop\geth启动及控制台命令\image\geth_output_log.png)

也可以添加启用 RPC 的 --rpc 选项

# geth控制台使用

1、进入console控制台，输入web3查看对象。

```javascript
> web3
{
  admin: {
    datadir: "/home/jassy/project/Geth-Comments/ZhangR/mychain2",
    nodeInfo: {
      enode: "enode://835c2ea146cae6a313ab78615425f4fca825fe8dcdcb6741de19c8a422bfd1abf5154859161f469f4ee50973e80265d092af84cb883fa5105ec58b117bfa6460@127.0.0.1:30303?discport=0",
      enr: "enr:-JC4QAINbybAwMAQUHnD_O_wlqAWwXDg0cOru-qvy_v6O27TX37vJ8ViEOeIu51YG-JPtYkSCaVJif1W9AqCndnbZJ0Bg2V0aMfGhJPR68WAgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQKDXC6hRsrmoxOreGFUJfT8qCX-jc3LZ0HeGcikIr_Rq4N0Y3CCdl8",
      id: "7ccbdc45b827387fc7e3f4c58da954cbe423eeb76753bab05425ba7cfe0026d7",
      ip: "127.0.0.1",
      listenAddr: "[::]:30303",
      name: "Geth/v1.9.11-stable/linux-amd64/go1.14",
      ports: {
        discovery: 0,
        listener: 30303
      },
      protocols: {
        eth: {...}
      }
    },
    peers: [],
    addPeer: function(),
    addTrustedPeer: function(),
    clearHistory: function(),
    exportChain: function(),
    getDatadir: function(callback),
    getNodeInfo: function(callback),
    getPeers: function(callback),
    importChain: function(),
    removePeer: function(),
    removeTrustedPeer: function(),
    sleep: function(),
    sleepBlocks: function(),
    startRPC: function(),
    startWS: function(),
    stopRPC: function(),
    stopWS: function()
  },
  bzz: {
    hive: undefined,
    info: undefined,
    blockNetworkRead: function(),
    download: function(),
    get: function(),
    getHive: function(callback),
    getInfo: function(callback),
    modify: function(),
    put: function(),
    retrieve: function(),
    store: function(),
    swapEnabled: function(),
    syncEnabled: function(),
    upload: function()
  },
  currentProvider: {
    send: function(),
    sendAsync: function()
  },
  db: {
    getHex: function(),
    getString: function(),
    putHex: function(),
    putString: function()
  },
  debug: {
    accountRange: function(),
    backtraceAt: function(),
    blockProfile: function(),
    chaindbCompact: function(),
    chaindbProperty: function(),
    cpuProfile: function(),
    dumpBlock: function(),
    freeOSMemory: function(),
    freezeClient: function(),
    gcStats: function(),
    getBadBlocks: function(),
    getBlockRlp: function(),
    getModifiedAccountsByHash: function(),
    getModifiedAccountsByNumber: function(),
    goTrace: function(),
    memStats: function(),
    mutexProfile: function(),
    preimage: function(),
    printBlock: function(),
    seedHash: function(),
    setBlockProfileRate: function(),
    setGCPercent: function(),
    setHead: function(),
    setMutexProfileFraction: function(),
    stacks: function(),
    standardTraceBadBlockToFile: function(),
    standardTraceBlockToFile: function(),
    startCPUProfile: function(),
    startGoTrace: function(),
    stopCPUProfile: function(),
    stopGoTrace: function(),
    storageRangeAt: function(),
    testSignCliqueBlock: function(),
    traceBadBlock: function(),
    traceBlock: function(),
    traceBlockByHash: function(),
    traceBlockByNumber: function(),
    traceBlockFromFile: function(),
    traceTransaction: function(),
    verbosity: function(),
    vmodule: function(),
    writeBlockProfile: function(),
    writeMemProfile: function(),
    writeMutexProfile: function()
  },
  eth: {
    accounts: [],
    blockNumber: 0,
    coinbase: undefined,
    compile: {
      lll: function(),
      serpent: function(),
      solidity: function()
    },
    defaultAccount: undefined,
    defaultBlock: "latest",
    gasPrice: 1000000000,
    hashrate: 0,
    mining: false,
    pendingTransactions: [],
    protocolVersion: "0x41",
    syncing: false,
    call: function(),
    chainId: function(),
    contract: function(abi),
    estimateGas: function(),
    fillTransaction: function(),
    filter: function(options, callback, filterCreationErrorCallback),
    getAccounts: function(callback),
    getBalance: function(),
    getBlock: function(),
    getBlockByHash: function(),
    getBlockByNumber: function(),
    getBlockNumber: function(callback),
    getBlockTransactionCount: function(),
    getBlockUncleCount: function(),
    getCode: function(),
    getCoinbase: function(callback),
    getCompilers: function(),
    getGasPrice: function(callback),
    getHashrate: function(callback),
    getHeaderByHash: function(),
    getHeaderByNumber: function(),
    getMining: function(callback),
    getPendingTransactions: function(callback),
    getProof: function(),
    getProtocolVersion: function(callback),
    getRawTransaction: function(),
    getRawTransactionFromBlock: function(),
    getStorageAt: function(),
    getSyncing: function(callback),
    getTransaction: function(),
    getTransactionCount: function(),
    getTransactionFromBlock: function(),
    getTransactionReceipt: function(),
    getUncle: function(),
    getWork: function(),
    iban: function(iban),
    icapNamereg: function(),
    isSyncing: function(callback),
    namereg: function(),
    resend: function(),
    sendIBANTransaction: function(),
    sendRawTransaction: function(),
    sendTransaction: function(),
    sign: function(),
    signTransaction: function(),
    submitTransaction: function(),
    submitWork: function()
  },
  ethash: {
    getHashrate: function(),
    getWork: function(),
    submitHashRate: function(),
    submitWork: function()
  },
  isIBAN: undefined,
  miner: {
    getHashrate: function(),
    setEtherbase: function(),
    setExtra: function(),
    setGasPrice: function(),
    setRecommitInterval: function(),
    start: function(),
    stop: function()
  },
  net: {
    listening: true,
    peerCount: 0,
    version: "666",
    getListening: function(callback),
    getPeerCount: function(callback),
    getVersion: function(callback)
  },
  personal: {
    listAccounts: [],
    listWallets: [],
    deriveAccount: function(),
    ecRecover: function(),
    getListAccounts: function(callback),
    getListWallets: function(callback),
    importRawKey: function(),
    initializeWallet: function(),
    lockAccount: function(),
    newAccount: function(),
    openWallet: function(),
    sendTransaction: function(),
    sign: function(),
    signTransaction: function(),
    unlockAccount: function(),
    unpair: function()
  },
  providers: {
    HttpProvider: function(host, timeout, user, password),
    IpcProvider: function(path, net)
  },
  rpc: {
    modules: {
      admin: "1.0",
      debug: "1.0",
      eth: "1.0",
      ethash: "1.0",
      miner: "1.0",
      net: "1.0",
      personal: "1.0",
      rpc: "1.0",
      txpool: "1.0",
      web3: "1.0"
    },
    getModules: function(callback)
  },
  settings: {
    defaultAccount: undefined,
    defaultBlock: "latest"
  },
  shh: {
    addPrivateKey: function(),
    addSymKey: function(),
    deleteKeyPair: function(),
    deleteSymKey: function(),
    generateSymKeyFromPassword: function(),
    getPrivateKey: function(),
    getPublicKey: function(),
    getSymKey: function(),
    hasKeyPair: function(),
    hasSymKey: function(),
    info: function(),
    markTrustedPeer: function(),
    newKeyPair: function(),
    newMessageFilter: function(options, callback, filterCreationErrorCallback),
    newSymKey: function(),
    post: function(),
    setMaxMessageSize: function(),
    setMinPoW: function(),
    version: function()
  },
  txpool: {
    content: {
      pending: {},
      queued: {}
    },
    inspect: {
      pending: {},
      queued: {}
    },
    status: {
      pending: 0,
      queued: 0
    },
    getContent: function(callback),
    getInspect: function(callback),
    getStatus: function(callback)
  },
  version: {
    api: "0.20.1",
    ethereum: "0x41",
    network: "666",
    node: "Geth/v1.9.11-stable/linux-amd64/go1.14",
    whisper: undefined,
    getEthereum: function(callback),
    getNetwork: function(callback),
    getNode: function(callback),
    getWhisper: function(callback)
  },
  BigNumber: function a(e,n),
  createBatch: function(),
  fromAscii: function(str),
  fromDecimal: function(value),
  fromICAP: function(icap),
  fromUtf8: function(str),
  fromWei: function(number, unit),
  isAddress: function(address),
  isChecksumAddress: function(address),
  isConnected: function(),
  padLeft: function(string, chars, sign),
  padRight: function(string, chars, sign),
  reset: function(keepIsSyncing),
  setProvider: function(provider),
  sha3: function(string, options),
  toAscii: function(hex),
  toBigNumber: function(number),
  toChecksumAddress: function(address),
  toDecimal: function(value),
  toHex: function(val),
  toUtf8: function(hex),
  toWei: function(number, unit)
}

```

2、新建账户   

```javascript
> personal.newAccount()
Passphrase: 
Repeat passphrase: 
"0x0de94aee8cb4cfb282ef56ab7735e794528a1c55"
> personal.newAccount("123456")
"0x2d985eb845347da804a7e0861dbd73594045dd09"

```

3、查看账户信息

```javascript
> eth.accounts
["0x0de94aee8cb4cfb282ef56ab7735e794528a1c55", "0x2d985eb845347da804a7e0861dbd73594045dd09"]
> eth.getBalance(eth.accounts[0])
0
> eth.getBalance(eth.accounts[1])
0
```

4、设置一个账户为etherbase（开发者模式中自带账户即为etherbase）

```javascript
> miner.setEtherbase(eth.accounts[0])
true
```

5、开始挖矿，可以看到Generating DAG的进度，完成之后挖矿就会开始，可以在日志中看到小锤子图标。

```javascript
> miner.start()
null
> miner.stop()
null
```

![1584810797009](D:\Destop\geth启动及控制台命令\image\miner.png)

6、查看账户现在的余额和区块数。

```javascript
> eth.getBalance(eth.accounts[0])
6000000000000000000
> eth.blockNumber
3
```

7、从第一个账户发送1 ether到第二个账户

首先要先解锁第一个账户

```javascript
> personal.unlockAccount(eth.accounts[0])
Unlock account 0x0de94aee8cb4cfb282ef56ab7735e794528a1c55
Passphrase: 
true
```

发送交易

```javascript
>eth.sendTransaction({from:eth.accounts[0],to:eth.accounts[1],value:web3.toWei(1,"ether")})
"0x11fb5750bf8c62d47a504c11737bb4cd917babaa3ae84b7344c6ce32496e0595"
```

此时账户中的余额不会变化，我们还需要再次挖矿才能看到结果。

![1584811555021](D:\Destop\geth启动及控制台命令\image\miner2.png)

再次查看余额

```javascript
> eth.getBalance(eth.accounts[0])
11000000000000000000
> eth.getBalance(eth.accounts[1])
1000000000000000000
```

8、通过发送交易返回的hash再次查看该交易

```javascript
>eth.getTransaction("0x11fb5750bf8c62d47a504c11737bb4cd917babaa3ae84b7344c6ce32496e0595")
{
  blockHash: "0xfcf364577a42de928c63422483bda36351304dfb834e59e2e616417466a34f89",
  blockNumber: 4,
  from: "0x0de94aee8cb4cfb282ef56ab7735e794528a1c55",
  gas: 21000,
  gasPrice: 1000000000,
  hash: "0x11fb5750bf8c62d47a504c11737bb4cd917babaa3ae84b7344c6ce32496e0595",
  input: "0x",
  nonce: 0,
  r: "0x63bf9d82727008ac963475fb95210443034e269c99166f1b5768011c3237715c",
  s: "0x6050e997588acf976fee1400b32063e4ba0ed809673dd5432e6e9de647ccaeb9",
  to: "0x2d985eb845347da804a7e0861dbd73594045dd09",
  transactionIndex: 0,
  v: "0x557",
  value: 1000000000000000000
}

```

9、查看区块

```javascript
> eth.getBlock(1)
{
  difficulty: 131072,
  extraData: "0xd68301090b846765746886676f312e3134856c696e7578",
  gasLimit: 4704588,
  gasUsed: 0,
  hash: "0xf1e3c9fd7dd0fcfc3b2c2fd0c62b8f22b50c7d97f19f56b1ea141b24a190fa4a",
  logsBloom: "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  miner: "0x0de94aee8cb4cfb282ef56ab7735e794528a1c55",
  mixHash: "0x8c8ee96da98b2a94b7dcd6069ecfa2165546acb2f0567ee44f0d3e7ea356c281",
  nonce: "0x4bd281efd93c5263",
  number: 1,
  parentHash: "0xd3d6bb893a6e274cab241245d5df1274c58d664fbb1bfd6e59141c2e0bc5304a",
  receiptsRoot: "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
  sha3Uncles: "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
  size: 534,
  stateRoot: "0x7a3923186faaebf04137328b4a44f7b0145c437f3b57ad279b89acb26edd5048",
  timestamp: 1584810647,
  totalDifficulty: 131074,
  transactions: [],
  transactionsRoot: "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
  uncles: []
}


> eth.getBlock(4)
{
  difficulty: 131072,
  extraData: "0xd68301090b846765746886676f312e3134856c696e7578",
  gasLimit: 4718380,
  gasUsed: 21000,
  hash: "0xfcf364577a42de928c63422483bda36351304dfb834e59e2e616417466a34f89",
  logsBloom: "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  miner: "0x0de94aee8cb4cfb282ef56ab7735e794528a1c55",
  mixHash: "0x7f6be6479bcc11cbb1547a026a7c0cc5ad4b6e228f36558a6305d01ee6d43227",
  nonce: "0x73cc56e29fa6743f",
  number: 4,
  parentHash: "0x1b7ecc7e74ebfa864739c1251d518b0e3b6d1136c6ef0415ad8a85c2e3751bab",
  receiptsRoot: "0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2",
  sha3Uncles: "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
  size: 648,
  stateRoot: "0x3dbb528be75e465921c791a1517150dcbc1749e399dc98b340a6f69f21a43b55",
  timestamp: 1584811389,
  totalDifficulty: 524290,
  transactions: ["0x11fb5750bf8c62d47a504c11737bb4cd917babaa3ae84b7344c6ce32496e0595"],
  transactionsRoot: "0xd065817d56a7892e5ffc725af4d3a90bd0fee92845cfacede19ee046d81ff76f",
  uncles: []
}

```
