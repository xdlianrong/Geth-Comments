package main

type NodeInfo struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Enode string `json:"enode"`
		Enr   string `json:"enr"`
		IP    string `json:"ip"`
		Ports struct {
			Discovery int `json:"discovery"`
			Listener  int `json:"listener"`
		} `json:"ports"`
		ListenAddr string `json:"listenAddr"`
		Protocols  struct {
			Eth struct {
				Network    int    `json:"network"`
				Difficulty int    `json:"difficulty"`
				Genesis    string `json:"genesis"`
				Config     struct {
					ChainID             int      `json:"chainId"`
					HomesteadBlock      int      `json:"homesteadBlock"`
					Eip150Block         int      `json:"eip150Block"`
					Eip150Hash          string   `json:"eip150Hash"`
					Eip155Block         int      `json:"eip155Block"`
					Eip158Block         int      `json:"eip158Block"`
					ByzantiumBlock      int      `json:"byzantiumBlock"`
					ConstantinopleBlock int      `json:"constantinopleBlock"`
					PetersburgBlock     int      `json:"petersburgBlock"`
					IstanbulBlock       int      `json:"istanbulBlock"`
					Ethash              struct{} `json:"ethash"`
					CryptoType          int      `json:"cryptoType"`
				} `json:"config"`
				Head string `json:"head"`
			} `json:"eth"`
		} `json:"protocols"`
	} `json:"result"`
}

type RPCbody struct {
	Jsonrpc string   `json:"jsonrpc"`
	Method  string   `json:"method"`
	Params  []string `json:"params"`
	ID      int      `json:"id"`
}

type AddPeerResult struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  bool   `json:"result"`
}

type PeersResult struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  []struct {
		Enode   string   `json:"enode"`
		ID      string   `json:"id"`
		Name    string   `json:"name"`
		Caps    []string `json:"caps"`
		Network struct {
			LocalAddress  string `json:"localAddress"`
			RemoteAddress string `json:"remoteAddress"`
			Inbound       bool   `json:"inbound"`
			Trusted       bool   `json:"trusted"`
			Static        bool   `json:"static"`
		} `json:"network"`
		Protocols struct {
			Eth struct {
				Version    int    `json:"version"`
				Difficulty int    `json:"difficulty"`
				Head       string `json:"head"`
			} `json:"eth"`
		} `json:"protocols"`
	} `json:"result"`
}

type PeerCountResult struct {
	ID      int    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  string `json:"result"`
}
type AccountsResult struct {
	ID      int      `json:"id"`
	Jsonrpc string   `json:"jsonrpc"`
	Result  []string `json:"result"`
}
type Receipt struct {
	Cmv    string `json:"cmv"`
	Epkrc1 string `json:"epkrc1"`
	Epkrc2 string `json:"epkrc2"`
	Hash   string `json:"hash"` //此次购币交易的交易哈希
}
type Coin struct {
	Cmv  string `json:"cmv"`
	Vor  string `json:"vor"`
	Hash string `json:"hash"`
}
type RPCtx struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		BlockHash        string `json:"blockHash"`
		BlockNumber      string `json:"blockNumber"`
		From             string `json:"from"`
		Gas              string `json:"gas"`
		GasPrice         string `json:"gasPrice"`
		Hash             string `json:"hash"`
		Input            string `json:"input"`
		Nonce            string `json:"nonce"`
		To               string `json:"to"`
		TransactionIndex string `json:"transactionIndex"`
		Value            string `json:"value"`
		V                string `json:"v"`
		R                string `json:"r"`
		S                string `json:"s"`
		ID               string `json:"ID"`
		Erpkc1           string `json:"erpkc1"`
		Erpkc2           string `json:"erpkc2"`
		Espkc1           string `json:"espkc1"`
		Espkc2           string `json:"espkc2"`
		Cmrpk            string `json:"cmrpk"`
		Cmspk            string `json:"cmspk"`
		Erpkeps0         string `json:"erpkeps0"`
		Erpkeps1         string `json:"erpkeps1"`
		Erpkeps2         string `json:"erpkeps2"`
		Erpkeps3         string `json:"erpkeps3"`
		Erpkept          string `json:"erpkept"`
		Espkeps0         string `json:"espkeps0"`
		Espkeps1         string `json:"espkeps1"`
		Espkeps2         string `json:"espkeps2"`
		Espkeps3         string `json:"espkeps3"`
		Espkept          string `json:"espkept"`
		Evsc1            string `json:"evsc1"`
		Evsc2            string `json:"evsc2"`
		Evrc1            string `json:"evrc1"`
		Evrc2            string `json:"evrc2"`
		Cms              string `json:"cms"`
		Cmr              string `json:"cmr"`
		Cmsfpc           string `json:"cmsfpc"`
		Cmsfpz1          string `json:"cmsfpz1"`
		Cmsfpz2          string `json:"cmsfpz2"`
		Cmrfpc           string `json:"cmrfpc"`
		Cmrfpz1          string `json:"cmrfpz1"`
		Cmrfpz2          string `json:"cmrfpz2"`
		Evsbsc1          string `json:"evsbsc1"`
		Evsbsc2          string `json:"evsbsc2"`
		Evoc1            string `json:"evoc1"`
		Evoc2            string `json:"evoc2"`
		Cmo              string `json:"cmo"`
		Evoeps0          string `json:"evoeps0"`
		Evoeps1          string `json:"evoeps1"`
		Evoeps2          string `json:"evoeps2"`
		Evoeps3          string `json:"evoeps3"`
		Evoept           string `json:"evoept"`
		Bpc              string `json:"bpc"`
		Bprv             string `json:"bprv"`
		Bprr             string `json:"bprr"`
		Bpsv             string `json:"bpsv"`
		Bpsr             string `json:"bpsr"`
		Bpsor            string `json:"bpsor"`
		Epkrc1           string `json:"epkrc1"`
		Epkrc2           string `json:"epkrc2"`
		Epkpc1           string `json:"epkpc1"`
		Epkpc2           string `json:"epkpc2"`
		Sigm             string `json:"sigm"`
		Sigmhash         string `json:"sigmhash"`
		Sigr             string `json:"sigr"`
		Sigs             string `json:"sigs"`
		Cmv              string `json:"cmv"`
		Cmsrc1           string `json:"cmsrc1"`
		Cmsrc2           string `json:"cmsrc2"`
		Cmrrc1           string `json:"cmrrc1"`
		Cmrrc2           string `json:"cmrrc2"`
	} `json:"result"`
}

type sendRPCTx struct {
	Jsonrpc string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  []sendRPCTxParams `json:"params"`
	ID      int               `json:"id"`
}
type sendRPCTxParams struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Value    string `json:"value"`
	ID       string `json:"id"`
	Data     string `json:"data"`
	Spk      string `json:"spk"`
	Rpk      string `json:"rpk"`
	S        string `json:"s"`
	R        string `json:"r"`
	Vor      string `json:"vor"`
	Cmo      string `json:"cmo"`
}
