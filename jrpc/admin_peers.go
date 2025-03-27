package jrpc

type AdminPeers []AdminPeers_Peer

type AdminPeers_Peer struct {
	Enode        string   `json:"enode"`
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Capabilities []string `json:"caps"`

	Network struct {
		LocalAddress  string `json:"localAddress"`
		RemoteAddress string `json:"remoteAddress"`
	} `json:"network"`
}
