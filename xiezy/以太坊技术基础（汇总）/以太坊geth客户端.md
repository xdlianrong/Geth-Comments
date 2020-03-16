# 以太坊geth客户端

## 1 geth简介

以太坊的最初源码实现，以太坊开发团队基于go语言开发的以太坊平台

## 2 启动geth

推荐linux ubuntu系统

前提是安装好了golang语言，并且配置好了环境。现在版本的geth需要golang至少1.13以上，不要用apt安装golang，下载的golang版本一般比较低，可以先在别的地方下好软件包，然后解压至/usr/local 

编译源码

```
git clone https://github.com/ethereum/go-ethereum.git
cd go-ethereum
make geth
```

这里的make geth可能会报错，原因是windows下的换行是/r/n，linux是/n，所以需要把env.sh的格式从dos变为unix，因此执行

```
cd build
vi env.sh
:set ff = unix
```

可以在build/bin目录下看到make后的信息，通过以下命令查看geth的版本信息

```
./build/bin/geth version
```

通过以下语句，启动脚本（这里设置在./build/bin/geth/data目录启动），这里面可以利用三种模式在以太坊主网设置节点，推荐还可以先尝试连接以太坊的测试网络（在这里可以完成智能合约检查）

```
./build/bin/geth --testnet
## 连接以太坊测试网络
./build/bin/geth --datadir ./data --syncmode full 
## 设置为全节点，不推荐个人机使用，耗费大量存储和内存
./build/bin/geth --datadir ./data --syncmode fast
## 设置为全节点，但是不完成交易验证，节省CPU耗费，推荐使用
./build/bin/geth --datadir ./data --syncmode light
## 设置为轻节点，推荐个人机使用
```

这里面，p2p端口的默认端口号是30303，利用下面命令可以查看一些代码日志并且完成版本切换

```
git log                 ## 切换查看当前geth的历史更新信息
git tag                 ## 查看历史geth版本
git checkout v1.8.17    ## （例子）切换geth版本至v1.8.17
```

## 3 利用以太坊架构搭建一条私有链

首先需要自己新建并且编写一个json文件，命名为genesis.json，完成一些创世块和私有链信息配置，建议不要把该json文件放在主目录，可以新建一个文件夹放在里面

```go
{
  "config": {
    "chainId": 666,
    // 链id，主网chainId为1，testnet的chainId为3
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
   // 矿工生成的nonce值
  "timestamp": "0x5ddf8f3e",
   // 创世块时间戳
  "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000",
   // 附加信息，随便写
  "gasLimit": "0x47b760",
   // 区块总汽油费限制
  "difficulty": "0x00002",
   // 挖矿难度值
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
   // 上一个区块一部分的哈希值，用来与nonce值一起配合挖矿
  "coinbase": "0x0000000000000000000000000000000000000000",
   // 矿工账号
  "alloc": {
    "0x1e82968C4624880FD1E8e818421841E6DB8D1Fa4" : {"balance" : "30000000000000000000"}
  },
   // 在创世块预定义一些账户，可以分配一些资金在上面，作为预售eth
  "number": "0x0",
  "gasUsed": "0x0",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000"
   // 创世块的父区块，默认值为0
}
```

然后初始化私有链，首先要切换到主目录，启动刚才配置的json文件

```
./build/bin/geth --datadir ../mychain/ init ../mychain/genesis.json
```

最后启动私有链,这里的networkid要与json文件配置的chainId一样

```
./build/bin/geth --datadir ../mychain --networkid 666
```

## 4 geth控制台

运行下条指令，与geth控制台交互，整个控制台可以认为是一个web3对象，web3里面定义了许多属性，提供各种方法

```
./build/bin/geth --datadir ../mychain --networkid 666 console
```





