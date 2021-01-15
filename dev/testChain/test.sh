#首次运行需要修改以下2个变量
gethDir="/Users/fuming/go/src/Geth-Comments" #Geth的GitHub项目所在文件夹
testDataDir="/Users/fuming/Downloads/testChain" #一个空文件夹，用于存储节点数据
#首次运行需要修改以上2个变量
genesisPath=""$gethDir"/dev/testChain/genesis.json"
gethCodeDir=""$gethDir"/go-ethereum-release-1.9"
gethBinDir=""$gethCodeDir"/build/bin/geth"
regulatorIP="39.106.173.191"
exchangeIP="127.0.0.1"
rpcAPI="eth,net,web3,personal,admin,txpool,debug,miner"
identity="666"
kill -9 $(lsof -i:8545 | awk '{print $2}')
kill -9 $(lsof -i:8546 | awk '{print $2}')
kill -9 $(lsof -i:8547 | awk '{print $2}')
kill -9 $(lsof -i:8548 | awk '{print $2}')
kill -9 $(lsof -i:8549 | awk '{print $2}')
kill -9 $(lsof -i:1323 | awk '{print $2}')
screen -S exchange -X quit
cd $gethCodeDir \
&& make geth \
&& cd $gethCodeDir \
&& cd ../exchange \
&& go build \
&& screen -S exchange -d -m ./exchange -ea 0x47c9a59fe5d28ff862f8eaf5924dbc90af00b0ce -ek 123456 \
&& cd $testDataDir \
&& rm -rf ./* \
&& mkdir node0 node1 node2 node3 node4\
&& $gethBinDir --datadir node0 --regulatorip $regulatorIP --exchangeip $exchangeIP init $genesisPath \
&& $gethBinDir --datadir node1 --regulatorip $regulatorIP --exchangeip $exchangeIP init $genesisPath \
&& $gethBinDir --datadir node2 --regulatorip $regulatorIP --exchangeip $exchangeIP init $genesisPath \
&& $gethBinDir --datadir node3 --regulatorip $regulatorIP --exchangeip $exchangeIP init $genesisPath \
&& $gethBinDir --datadir node4 --regulatorip $regulatorIP --exchangeip $exchangeIP init $genesisPath \
&& cd $gethCodeDir \
&& cd ../dev/testChain/keys \
&& cp UTC--2021-01-14T03-06-22.416540000Z--47c9a59fe5d28ff862f8eaf5924dbc90af00b0ce ""$testDataDir"/node0/keystore"\
&& cp UTC--2021-01-14T03-07-11.612118000Z--19dbca3be6358f474caea47a0f177a33afa5a1d2 ""$testDataDir"/node1/keystore"\
&& cp UTC--2021-01-14T03-07-16.776036000Z--f17d78b504e271ea28523ac4eb39c4ecd1a86349 ""$testDataDir"/node2/keystore"\
&& cp UTC--2021-01-14T03-07-19.991185000Z--2c03bc23e7cc213cae42f5a6591c8a4789784ca8 ""$testDataDir"/node3/keystore"\
&& cp UTC--2021-01-14T09-23-35.900984000Z--ec349a9e1661b96e83fd2e761c22ddacf47141b3 ""$testDataDir"/node4/keystore"\
&& cd $gethCodeDir
./build/bin/geth --identity $identity --rpc --rpcport "8545" --rpccorsdomain "*" --rpcapi $rpcAPI --datadir "$testDataDir"/node0 --ipcpath "$testDataDir"/node0/geth.ipc --port "30303" --ipcpath "node0.rpc" --nodiscover --allow-insecure-unlock --regulatorip $regulatorIP --exchangeip $exchangeIP 2>>"$testDataDir"/node0/log.log&
./build/bin/geth --identity $identity --rpc --rpcport "8546" --rpccorsdomain "*" --rpcapi $rpcAPI --datadir "$testDataDir"/node1 --ipcpath "$testDataDir"/node1/geth.ipc --port "30304" --ipcpath "node1.rpc" --nodiscover --allow-insecure-unlock --regulatorip $regulatorIP --exchangeip $exchangeIP 2>>"$testDataDir"/node1/log.log&
./build/bin/geth --identity $identity --rpc --rpcport "8547" --rpccorsdomain "*" --rpcapi $rpcAPI --datadir "$testDataDir"/node2 --ipcpath "$testDataDir"/node2/geth.ipc --port "30305" --ipcpath "node2.rpc" --nodiscover --allow-insecure-unlock --regulatorip $regulatorIP --exchangeip $exchangeIP 2>>"$testDataDir"/node2/log.log&
./build/bin/geth --identity $identity --rpc --rpcport "8548" --rpccorsdomain "*" --rpcapi $rpcAPI --datadir "$testDataDir"/node3 --ipcpath "$testDataDir"/node3/geth.ipc --port "30306" --ipcpath "node3.rpc" --nodiscover --allow-insecure-unlock --regulatorip $regulatorIP --exchangeip $exchangeIP 2>>"$testDataDir"/node3/log.log&
./build/bin/geth --identity $identity --rpc --rpcport "8549" --rpccorsdomain "*" --rpcapi $rpcAPI --datadir "$testDataDir"/node4 --ipcpath "$testDataDir"/node4/geth.ipc --port "30307" --ipcpath "node4.rpc" --nodiscover --allow-insecure-unlock --regulatorip $regulatorIP --exchangeip $exchangeIP 2>>"$testDataDir"/node4/log.log&