#geth搭建私链

### 搭建环境
Ubuntu 16.04

### 1.安装Geth

通过以下命令直接安装Geth
```udo apt-get install software-properties-common
sudo add-apt-repository -y ppa:ethereum/ethereum
sudo apt-get update
sudo apt-get install ethereum
```
### 2.配置创世块
```
mkdir Priviate-Geth
cd Private-Geth
gedit genesis.json (也可以用Vim，我比较习惯图形化的)
```

创世块json文件
```{
  "config": {
     "chainId": 10,
     "homesteadBlock": 0,
     "eip155Block": 0,
     "eip158Block": 0
  },
  "coinbase"   : "0x0000000000000000000000000000000000000000",
  "difficulty" : "0x2000",
  "extraData"  : "",
  "gasLimit"   : "0xffffffff",
  "nonce"      : "0x0000000000000042",
  "mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
  "parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
  "timestamp"  : "0x00",
  "alloc": {
     "08a58f09194e403d02a1928a7bf78646cfc260b0": {
         "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
     },
     "87366ef81db496edd0ea2055ca605e8686eec1e6": {
         "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
     }
  }
}
```
json文件各个说明
| 参数名 |  作用 |
| :-----| ----: | 
| chainId | 指定了独立的区块链网络 ID。网络 ID 在连接到其他节点的时候会用到，以太坊公网的网络 ID 是 1，为了不与公有链网络冲突，运行私有链节点的时候要指定自己的网络 ID。不同 ID 网络的节点无法相互连接 | 
| nonce | nonce就是一个64位随机数，用于挖矿 | 
| difficulty | 设置设置当前区块的难度，越大挖矿就越难 | 
| mixhash | 与nonce配合用于挖矿，由上一个区块的一部分生成的hash | 
| alloc | 用来预置账号以及账号的以太币数量，因为私有链挖矿比较容易，所以我们也可以不需要预置有币的账号，需要的时候自己创建即可以 | 
| gasLimit | 该值设置对GAS的消耗总量限制，用来限制区块能包含的交易信息总和。我们创建的是私有链，可以填最大 | 
| parentHash|上一个区块的hash，创世区块的该项参数就为0 |
| coinbase| 矿工账号|

以上搜了一些我觉得比较重要的一些参数，了解这些参数，对创世块，区块有个更好的了解

### 初始化创世块
 ```
 cd Priviate-Geth
 mkdir data   ##data用于存放账户信息和区块数据
 geth --datadir ./data init genesis.json
```
执行完最后一句后，会在data目录产生两个子文件
Geth：保存所建私有链的区块数据
keystore：用于保存用户的账户数据，私钥之类的东西

### 启动私有链

```
## --datadir 表示当前区块链网络数据存放的位置
## --nodiscover 表示该链禁止被其他节点发现
打开控制台
geth --datadir ./data --nodiscover console 2
如果在创建创世区块之前已经启动了私有链，在创建了创世区块配置文件后，可以通过以下命令重新进入geth控制台，并使用配置文件更新区块：
geth --datadir data --networkid 10 console
networkid为配置文件中的chainid
```
### 私链上的一系列操作
```
personal.newAccount("") 创建新账户，里面输入密码，创建后会返回账户的地址
eth.account 查看私链账户信息,返回的为一个列表类型，存储账户地址
eth.getBalance()  里面为账户地址
miner.start() 开始挖矿
miner.stop()  停止挖矿
personal.unlockAccount(，‘’) 账户解锁，参数为账户地址和密码
需先解锁，再转账
amount = web3.toWei(2,'ether')
 eth.sendTransaction({from:eth.accounts[0], to:eth.accounts[1], value:amount})   账户转账
 eth.getTransaction() 查看交易信息
 eth.getBlock() 查看区块信息
```
上面为一些常用的区块操作，还有更多的区块操作，可见<https://github.com/ethereum/go-ethereum/wiki/Management-APIs>

### 关于搭链
搭链的操作其实很简单，网络上也有很多的教程，可能出现一些问题，但是大多数的问题都能够搜索到，我觉得最关键的还是在于体会这整个过程，对区块链，像创世块之类的有个清楚的了解，对链上的一些操作有更好的认知，比如说在执行转账命令后，账户的余额并不会发生改变，这笔交易此时在交易池中，等待着矿工进行验证，此时我没有进行挖矿操作，故账户余额还未发生改变，等到我开启挖矿后，便发生了变化。因此搭建私人链还是对于认识区块链和以太坊有个更深入的作用，像挖矿之类的以前只停留在概念上，接触后体会更加深刻，所以还是值得搭建私链。
