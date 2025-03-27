package types

import "github.com/flashbots/bmonitor/jrpc"

type BuilderStatus struct {
	Peers  *jrpc.AdminPeers
	Txpool *jrpc.TxpoolContent
	Err    error
}
