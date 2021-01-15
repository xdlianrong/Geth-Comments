#首次运行需要修改以下3个变量
gethDir="/home/jassy/GolandProjects/Geth-Comments" #Geth的GitHub项目所在文件夹
testDataDir="/home/jassy/chain/testChain" #一个空文件夹，用于存储节点数据
SM="1" #和genesis.json中cryptoType值相等，若genesis.json中无此值，请令SM="0"
#首次运行需要修改以上3个变量
genesisPath=""$gethDir"/dev/testChain/genesis.json"
gethCodeDir=""$gethDir"/go-ethereum-release-1.9"
gethBinDir=""$gethCodeDir"/build/bin/geth"
regulatorIP="39.106.173.191"
exchangeIP="127.0.0.1"
rpcAPI="eth,net,web3,personal,admin,txpool,debug,miner"
identity="666"
pid=$(ps x | grep geth | grep -v grep | awk '{print $1}')
for i in $pid
do
  kill -9 $i
done
screen -S exchange -X quit
cd $gethCodeDir \
&& make geth \
&& cd $gethCodeDir \
&& cd ../exchange \
&& go build \
&& if [ "$SM" -eq "0" ];then
    screen -S exchange -d -m ./exchange -ea 0x47c9a59fe5d28ff862f8eaf5924dbc90af00b0ce -ek 123456
  elif [ "$SM" -eq "1" ];then
    screen -S exchange -d -m ./exchange -ea 0x352ccb3bc9a998e09f8872a25296d3f33b65b5e1 -ek 123456
  fi \
&& cd $testDataDir \
&& rm -rf ./* \
&& mkdir node0 node1 node2 node3 node4\
&& $gethBinDir --datadir node0 --regulatorip $regulatorIP --exchangeip $exchangeIP init $genesisPath \
&& $gethBinDir --datadir node1 --regulatorip $regulatorIP --exchangeip $exchangeIP init $genesisPath \
&& $gethBinDir --datadir node2 --regulatorip $regulatorIP --exchangeip $exchangeIP init $genesisPath \
&& $gethBinDir --datadir node3 --regulatorip $regulatorIP --exchangeip $exchangeIP init $genesisPath \
&& $gethBinDir --datadir node4 --regulatorip $regulatorIP --exchangeip $exchangeIP init $genesisPath \
&& cd $gethCodeDir \
&& if [ "$SM" -eq "0" ];then
        cd ../dev/testChain/keys \
        && cp UTC--2021-01-14T03-06-22.416540000Z--47c9a59fe5d28ff862f8eaf5924dbc90af00b0ce ""$testDataDir"/node0/keystore"\
        && cp UTC--2021-01-14T03-07-11.612118000Z--19dbca3be6358f474caea47a0f177a33afa5a1d2 ""$testDataDir"/node1/keystore"\
        && cp UTC--2021-01-14T03-07-16.776036000Z--f17d78b504e271ea28523ac4eb39c4ecd1a86349 ""$testDataDir"/node2/keystore"\
        && cp UTC--2021-01-14T03-07-19.991185000Z--2c03bc23e7cc213cae42f5a6591c8a4789784ca8 ""$testDataDir"/node3/keystore"\
        && cp UTC--2021-01-14T09-23-35.900984000Z--ec349a9e1661b96e83fd2e761c22ddacf47141b3 ""$testDataDir"/node4/keystore"
  elif [ "$SM" -eq "1" ];then
        cd ../dev/testChain/SMkeys \
        && cp UTC--2021-01-15T06-39-28.382669000Z--352ccb3bc9a998e09f8872a25296d3f33b65b5e1 ""$testDataDir"/node0/keystore"\
        && cp UTC--2021-01-15T06-40-30.866123000Z--0bd8dbe302ae1e41b5b4a268ff2d6e07851f16ce ""$testDataDir"/node1/keystore"\
        && cp UTC--2021-01-15T06-40-33.522033000Z--089400c7a8100555fc354c8607a020fbfe9dbc5a ""$testDataDir"/node2/keystore"\
        && cp UTC--2021-01-15T06-40-35.994789000Z--fd41cca27f355bc134e4f7c1bd6f752c9b837bf5 ""$testDataDir"/node3/keystore"\
        && cp UTC--2021-01-15T06-40-38.096210000Z--20bb6449fdb0685696a6f48566e9899d95b3684d ""$testDataDir"/node4/keystore"
fi \
&& cd $gethCodeDir
./build/bin/geth --identity $identity --rpc --rpcport "8545" --rpccorsdomain "*" --rpcapi $rpcAPI --datadir "$testDataDir"/node0 --ipcpath "$testDataDir"/node0/geth.ipc --port "30303" --ipcpath "node0.rpc" --nodiscover --allow-insecure-unlock --regulatorip $regulatorIP --exchangeip $exchangeIP 2>>"$testDataDir"/node0/log.log&
./build/bin/geth --identity $identity --rpc --rpcport "8546" --rpccorsdomain "*" --rpcapi $rpcAPI --datadir "$testDataDir"/node1 --ipcpath "$testDataDir"/node1/geth.ipc --port "30304" --ipcpath "node1.rpc" --nodiscover --allow-insecure-unlock --regulatorip $regulatorIP --exchangeip $exchangeIP 2>>"$testDataDir"/node1/log.log&
./build/bin/geth --identity $identity --rpc --rpcport "8547" --rpccorsdomain "*" --rpcapi $rpcAPI --datadir "$testDataDir"/node2 --ipcpath "$testDataDir"/node2/geth.ipc --port "30305" --ipcpath "node2.rpc" --nodiscover --allow-insecure-unlock --regulatorip $regulatorIP --exchangeip $exchangeIP 2>>"$testDataDir"/node2/log.log&
./build/bin/geth --identity $identity --rpc --rpcport "8548" --rpccorsdomain "*" --rpcapi $rpcAPI --datadir "$testDataDir"/node3 --ipcpath "$testDataDir"/node3/geth.ipc --port "30306" --ipcpath "node3.rpc" --nodiscover --allow-insecure-unlock --regulatorip $regulatorIP --exchangeip $exchangeIP 2>>"$testDataDir"/node3/log.log&
./build/bin/geth --identity $identity --rpc --rpcport "8549" --rpccorsdomain "*" --rpcapi $rpcAPI --datadir "$testDataDir"/node4 --ipcpath "$testDataDir"/node4/geth.ipc --port "30307" --ipcpath "node4.rpc" --nodiscover --allow-insecure-unlock --regulatorip $regulatorIP --exchangeip $exchangeIP 2>>"$testDataDir"/node4/log.log&
