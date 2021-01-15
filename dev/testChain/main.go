package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testChain/accounts"
	"time"
)

const url = "http://127.0.0.1"
const regulatorURL = "http://39.106.173.191:1423/"
const exchangeURL = "http://127.0.0.1:1323/"
const nodeCount = 5

func main() {
	var rpcPorts = [nodeCount]int{8545, 8546, 8547, 8548, 8549}
	var nodeInfo = [nodeCount]string{adminNodeinfo(rpcPorts[0]), adminNodeinfo(rpcPorts[1]), adminNodeinfo(rpcPorts[2]), adminNodeinfo(rpcPorts[3]), adminNodeinfo(rpcPorts[4])}
	addPeer(rpcPorts, nodeInfo)
	fmt.Println("节点0账户为交易所账户")
	checkGethAccounts(rpcPorts, nodeInfo)
	Alice := accounts.GenerateAccount("日照香炉生紫烟", "A", "1", "")
	Bob := accounts.GenerateAccount("遥看瀑布挂前川", "B", "2", "")
	Calvin := accounts.GenerateAccount("飞流直下三千尺", "C", "3", "")
	David := accounts.GenerateAccount("疑是银河落九天", "D", "4", "")
	register(Alice)
	register(Bob)
	register(Calvin)
	register(David)
	coinReceipt := buyCoin(Alice, 100)
	coin := decryptCoinReceipt(coinReceipt, Alice.Priv)
	mineTx(8545, rpcPorts, coin.Hash)
	fmt.Println("账户A花费上述购币承诺，向账户B转5单位金额，找零为95金额")
	txHash := ethSendTransaction(8546, ethAccounts(rpcPorts[1])[0], ethAccounts(rpcPorts[2])[0], Alice, Bob, coin, 100, 5)
	mineTx(8546, rpcPorts, txHash)
	rpcTx := ethGetTransactionByHash(8546, txHash)
	tx := rpcTx.Result
	fmt.Println("接收方B用自己的私钥解密得到交易金额")
	sendAmount := decryptValue(tx.Evsbsc1, tx.Evsbsc2, Bob.Priv)
	fmt.Println("找零额承诺为 " + tx.Cmr + " 随机数为" + decrypt(tx.Cmrrc1, tx.Cmrrc2, Alice.Priv))
	sendCMr := decrypt(tx.Cmsrc1, tx.Cmsrc2, Bob.Priv)
	fmt.Println("发送额为 " + sendAmount + " 承诺为 " + tx.Cms + " 随机数为" + sendCMr)
	fmt.Println("B向C转账，花费5元承诺，转出2元，找零3元")
	coin = Coin{
		Cmv: tx.Cms,
		Vor: sendCMr,
	}
	txHash = ethSendTransaction(8547, ethAccounts(rpcPorts[2])[0], ethAccounts(rpcPorts[3])[0], Bob, Calvin, coin, 5, 2)
	mineTx(8547, rpcPorts, txHash)
	rpcTx = ethGetTransactionByHash(8547, txHash)
	tx = rpcTx.Result
	fmt.Println("接收方C用自己的私钥解密得到交易金额")
	sendAmount = decryptValue(tx.Evsbsc1, tx.Evsbsc2, Calvin.Priv)
	fmt.Println("找零额承诺为 " + tx.Cmr + " 随机数为" + decrypt(tx.Cmrrc1, tx.Cmrrc2, Bob.Priv))
	sendCMr = decrypt(tx.Cmsrc1, tx.Cmsrc2, Calvin.Priv)
	fmt.Println("发送额为 " + sendAmount + " 承诺为 " + tx.Cms + " 随机数为" + sendCMr)
	fmt.Println("测试完毕，测试通过")
}
func decrypt(hex0xStringC1 string, hex0xStringC2 string, priv accounts.PrivateKey) string {
	hexData1, _ := hex.DecodeString(hex0xStringC1[2:])
	hexData2, _ := hex.DecodeString(hex0xStringC2[2:])
	C := accounts.CypherText{
		C1: hexData1,
		C2: hexData2,
	}
	M := fmt.Sprintf("0x%x", accounts.Decrypt(priv, C))
	return M
}
func decryptValue(hex0xStringC1 string, hex0xStringC2 string, priv accounts.PrivateKey) string {
	hexData1, _ := hex.DecodeString(hex0xStringC1[2:])
	hexData2, _ := hex.DecodeString(hex0xStringC2[2:])
	C := accounts.CypherText{
		C1: hexData1,
		C2: hexData2,
	}
	M := fmt.Sprintf("0x%x", accounts.DecryptValue(priv, C))
	return M
}
func decryptCoinReceipt(recript Receipt, priv accounts.PrivateKey) Coin {
	return Coin{
		Cmv:  recript.Cmv,
		Vor:  decrypt(recript.Epkrc1, recript.Epkrc2, priv),
		Hash: recript.Hash,
	}
}
func addPeer(rpcPorts [nodeCount]int, nodeInfo [nodeCount]string) bool {
	for i := 0; i < nodeCount; i++ {
		for j := 0; j < nodeCount; j++ {
			if i < j {
				adminAddPeer(rpcPorts[i], nodeInfo[j])
			}
		}
	}
	time.Sleep(time.Duration(2) * time.Second) //等两秒
	for i := 0; i < nodeCount; i++ {
		peerCount := netPeerCount(rpcPorts[i])
		if !strings.EqualFold(peerCount, "0x4") {
			Fatalf("节点" + strconv.Itoa(i) + "添加节点失败，期待：0x4，拥有：" + peerCount)
		}
	}
	fmt.Println(strconv.Itoa(nodeCount) + "个节点两两添加成功")
	return true
}
func checkGethAccounts(rpcPorts [nodeCount]int, nodeInfo [nodeCount]string) bool {
	for i := 0; i < nodeCount; i++ {
		accounts := ethAccounts(rpcPorts[i])
		if len(accounts) == 0 {
			Fatalf("节点" + strconv.Itoa(i) + "账户缺失")
		}
		fmt.Println("节点"+strconv.Itoa(i)+"账户", accounts)
	}
	return true
}
func adminNodeinfo(rpcPort int) string {
	data := RPCbody{
		Jsonrpc: "2.0",
		Method:  "admin_nodeInfo",
		Params:  nil,
		ID:      67,
	}
	jsonStr, _ := json.Marshal(data)
	resp, err := http.Post(url+":"+strconv.Itoa(rpcPort),
		"application/json",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	if resp == nil {
		Fatalf("RPC端口为" + strconv.Itoa(rpcPort) + "的Geth节点未启动")
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var info NodeInfo
	json.Unmarshal(body, &info)
	enode := info.Result.Enode
	//将enode中的IP改为127.0.0.1
	enode = strings.Replace(enode, enode[strings.LastIndex(enode, "@")+1:strings.LastIndex(enode, ":")], "127.0.0.1", -1)
	return enode
}
func adminAddPeer(rpcPort int, peerUrl string) bool {
	data := RPCbody{
		Jsonrpc: "2.0",
		Method:  "admin_addPeer",
		Params:  []string{peerUrl},
		ID:      67,
	}
	jsonStr, _ := json.Marshal(data)
	resp, err := http.Post(url+":"+strconv.Itoa(rpcPort),
		"application/json",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result AddPeerResult
	json.Unmarshal(body, &result)
	return result.Result
}
func netPeerCount(rpcPort int) string {
	data := RPCbody{
		Jsonrpc: "2.0",
		Method:  "net_peerCount",
		Params:  nil,
		ID:      67,
	}
	jsonStr, _ := json.Marshal(data)
	resp, err := http.Post(url+":"+strconv.Itoa(rpcPort),
		"application/json",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var peerCount PeerCountResult
	json.Unmarshal(body, &peerCount)
	return peerCount.Result
}
func ethAccounts(rpcPort int) []string {
	data := RPCbody{
		Jsonrpc: "2.0",
		Method:  "eth_accounts",
		Params:  nil,
		ID:      67,
	}
	jsonStr, _ := json.Marshal(data)
	resp, err := http.Post(url+":"+strconv.Itoa(rpcPort),
		"application/json",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var accounts AccountsResult
	json.Unmarshal(body, &accounts)
	return accounts.Result
}
func register(account accounts.Account) string {
	data := account.Info
	jsonStr, _ := json.Marshal(data)
	resp, err := http.Post(regulatorURL+"register",
		"application/json",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	res := string(body)
	if res == "Successful!" {
		fmt.Println("账户" + account.Info.Name + "注册成功")
	} else if res == "Account registered!" {
		fmt.Println("账户" + account.Info.Name + "已注册")
	} else if res == "Fail!" {
		Fatalf("账户" + account.Info.Name + "注册失败")
	}
	return string(body)
}
func buyCoin(account accounts.Account, amount int) Receipt {
	fmt.Println("账户" + account.Info.Name + "购买" + strconv.Itoa(amount) + "金额的代币")
	key := account.Pub
	data := struct {
		G1     string `json:"g1"`
		G2     string `json:"g2"`
		P      string `json:"p"`
		H      string `json:"h"`
		Amount string `json:"amount"`
	}{
		G1:     key.G1.String(),
		G2:     key.G2.String(),
		P:      key.P.String(),
		H:      key.H.String(),
		Amount: strconv.Itoa(amount),
	}
	jsonStr, _ := json.Marshal(data)
	resp, err := http.Post(exchangeURL+"buy",
		"application/json",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var receipt Receipt
	json.Unmarshal(body, &receipt)
	if receipt.Cmv == "" || receipt.Epkrc1 == "" || receipt.Epkrc2 == "" || receipt.Hash == "" {
		Fatalf("账户" + account.Info.Name + "购买" + strconv.Itoa(amount) + "金额代币失败")
	}
	fmt.Println("账户"+account.Info.Name+"购买"+strconv.Itoa(amount)+"金额代币成功", "购币交易Hash:", receipt.Hash)
	return receipt
}
func minerStart(rpcPort int) bool {
	data := RPCbody{
		Jsonrpc: "2.0",
		Method:  "miner_start",
		Params:  nil,
		ID:      67,
	}
	jsonStr, _ := json.Marshal(data)
	resp, err := http.Post(url+":"+strconv.Itoa(rpcPort),
		"application/json",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result AddPeerResult
	json.Unmarshal(body, &result)
	return result.Result
}
func minerStop(rpcPort int) bool {
	data := RPCbody{
		Jsonrpc: "2.0",
		Method:  "miner_stop",
		Params:  nil,
		ID:      67,
	}
	jsonStr, _ := json.Marshal(data)
	resp, err := http.Post(url+":"+strconv.Itoa(rpcPort),
		"application/json",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result AddPeerResult
	json.Unmarshal(body, &result)
	return result.Result
}
func ethGetTransactionByHash(rpcPort int, txHash string) RPCtx {
	data := RPCbody{
		Jsonrpc: "2.0",
		Method:  "eth_getTransactionByHash",
		Params:  []string{txHash},
		ID:      67,
	}
	jsonStr, _ := json.Marshal(data)
	resp, err := http.Post(url+":"+strconv.Itoa(rpcPort),
		"application/json",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result RPCtx
	json.Unmarshal(body, &result)
	return result
}
func mineTx(rpcPort int, allRPCPort [nodeCount]int, TxHash string) bool {
	fmt.Println("打包共识使交易", TxHash, "生效")
	minerStart(rpcPort)
	for i := 60; i > 0; i-- {
		if res := ethGetTransactionByHash(rpcPort, TxHash); res.Result.BlockHash == "" {
			time.Sleep(time.Duration(1) * time.Second) //等一秒
		} else {
			fmt.Println("交易", TxHash, "已被打包")
			break
		}
	}
	time.Sleep(time.Duration(5) * time.Second) //多挖几个块，不然不好共识
	minerStop(rpcPort)
	consensus := 1
	//必须所有节点都在块中拿到此交易才算共识成功
	for j := 60; j > 0; j-- {
		for i := 0; i < nodeCount; i++ {
			if res := ethGetTransactionByHash(allRPCPort[i], TxHash); res.Result.BlockHash == "" {
				//fmt.Println("交易", TxHash, "未生效")
				time.Sleep(time.Duration(1) * time.Second) //等一秒
				break
			}
			consensus = i - nodeCount + 1
		}
		if consensus == 0 {
			break
		}
	}
	if consensus != 0 {
		Fatalf("交易 " + TxHash + " 共识失败")
	}
	fmt.Println("交易", TxHash, "已被共识")
	return true
}
func personalUnlockAccount(rpcPort int, account string, passphrase string) bool {
	data := RPCbody{
		Jsonrpc: "2.0",
		Method:  "personal_unlockAccount",
		Params:  []string{account, passphrase},
		ID:      67,
	}
	jsonStr, _ := json.Marshal(data)
	resp, err := http.Post(url+":"+strconv.Itoa(rpcPort),
		"application/json",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result AddPeerResult
	json.Unmarshal(body, &result)
	return result.Result
}
func perpareTX(senderGethAccount string, receiverGethAccount string, senderAccount accounts.Account, receiverAccount accounts.Account, coin Coin, total int, amount int) sendRPCTx {
	param := sendRPCTxParams{
		From:     senderGethAccount,
		To:       receiverGethAccount,
		Gas:      "0x76c0",
		GasPrice: "0x9184e72a000",
		Value:    "0x1",
		ID:       "0x0",
		Data:     "0x00",
		Spk:      fmt.Sprintf("%0*x%0*x%0*x%0*x", 64, senderAccount.Pub.P, 64, senderAccount.Pub.G1, 64, senderAccount.Pub.G2, 64, senderAccount.Pub.H),
		Rpk:      fmt.Sprintf("%0*x%0*x%0*x%0*x", 64, receiverAccount.Pub.P, 64, receiverAccount.Pub.G1, 64, receiverAccount.Pub.G2, 64, receiverAccount.Pub.H),
		S:        fmt.Sprintf("0x%x", amount),
		R:        fmt.Sprintf("0x%x", total-amount),
		Vor:      coin.Vor,
		Cmo:      coin.Cmv,
	}
	var params []sendRPCTxParams
	params = append(params, param)
	tx := sendRPCTx{
		Jsonrpc: "2.0",
		Method:  "eth_sendTransaction",
		Params:  params,
		ID:      67,
	}
	return tx
}
func ethSendTransaction(senderRPCPort int, senderGethAccount string, receiverGethAccount string, senderAccount accounts.Account, receiverAccount accounts.Account, coin Coin, total int, amount int) string {
	if !personalUnlockAccount(senderRPCPort, senderGethAccount, "123456") {
		Fatalf("发送方账户解锁失败")
	}
	txs := perpareTX(senderGethAccount, receiverGethAccount, senderAccount, receiverAccount, coin, total, amount)
	data := txs
	jsonStr, _ := json.Marshal(data)
	resp, err := http.Post(url+":"+strconv.Itoa(senderRPCPort),
		"application/json",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result PeerCountResult
	json.Unmarshal(body, &result)
	if result.Result != "" {
		fmt.Println("转账交易发送成功，待打包共识")
	} else {
		Fatalf("转账交易发送失败")
	}
	return result.Result
}
