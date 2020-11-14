说明：

已经创建好私链，搭好两个账户，并挖了一会矿，私链所在文件夹为”你的私链的文件夹目录“，这个目录如果使用./默认是在go-ethereum下，也可以使用绝对路径

私链初始化参数：

--datadir "/home/test/音乐/privchain" --regulatorip 39.99.227.43 --exchangeurl "127.0.0.1:1323/pubpub"  init "/home/test/音乐/genesis.json"

私链启动参数：--regulatorip 39.99.227.43 --exchangeurl "127.0.0.1:1323/pubpub"  --identity "666" --rpc  --rpccorsdomain '*' --rpcport "8545" --rpcapi "eth,net,web3,personal,admin,txpool,debug,miner" --datadir "/home/test/音乐/privchain" --port "3303" --nodiscover --allow-insecure-unlock console   



其具体意义请查阅：

http://www.360doc.com/content/13/0814/10/9171956_307028720.shtml

具体有哪些api接口请查阅：

https://eth.wiki/en/json-rpc/API

