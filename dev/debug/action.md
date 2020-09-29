说明：

已经创建好私链，搭好两个账户，并挖了一会矿，私链所在文件夹为/privchain

账户分别为：

```go
0x75e36ea49f49d6f6619eb23904e8a8cab3a3dda2
0xfec0b0311e40713f2d9f35a9c4d9f6f538be6a91
```

私链启动参数： --identity "666" --rpc --rpcport "8545" --rpcapi "eth,net,web3,personal,admin,txpool,debug,miner" --datadir ./privchain --port "3303" --nodiscover --allow-insecure-unlock console  

其具体意义请查阅：

http://www.360doc.com/content/13/0814/10/9171956_307028720.shtml

具体有哪些api接口请查阅：

https://eth.wiki/en/json-rpc/API

