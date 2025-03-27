package jrpc

type TxpoolContent struct {
	Pending map[string]map[string]*TxpoolContent_Tx `json:"pending"`
	Queued  map[string]map[string]*TxpoolContent_Tx `json:"queued"`
}

type TxpoolContent_Tx struct {
	From  string `json:"from"`
	Nonce string `json:"nonce"`
	Hash  string `json:"hash"`
}
