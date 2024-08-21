package store

import (
	"sync"
	"time"

	ypeer "yiji.one/punch/signal/store/peer"
)

type PeerLogin struct {
	IP       string
	Port     int
	ClientID string
	WgPubKey string
	Token    string
}

type PeerConfig struct {
	IP              string `json:"ip"`
	Port            int    `json:"port"`
	ClientID        string `json:"clientId"`
	WgPubKey        string `json:"wgPubKey"`
	Token           string `json:"token"`
	LastKeepAliveAt string `json:"lastKeepAliveAt"`
}

var mu = sync.RWMutex{}

func Register(peerLogin PeerLogin) {
	mu.Lock()
	defer mu.Unlock()
	peers := []ypeer.Peer{}
	DB.Where("token = ?", peerLogin.Token).Find(&peers)
	for _, peer := range peers {
		if peer.ClientID == peerLogin.ClientID {
			// 更新peer信息
			DB.Model(&ypeer.Peer{}).Where("token = ?", peerLogin.Token).Where("client_id = ?", peerLogin.ClientID).Updates(ypeer.Peer{WgPubKey: peerLogin.WgPubKey, IP: peerLogin.IP, Port: peerLogin.Port})
			return
		}
	}
	peer := ypeer.Peer{
		IP:              peerLogin.IP,
		Port:            peerLogin.Port,
		ClientID:        peerLogin.ClientID,
		WgPubKey:        peerLogin.WgPubKey,
		Token:           peerLogin.Token,
		LastKeepAliveAt: time.Now(),
	}
	DB.Create(&peer)
}

func GetClients(clientId, token string) []PeerConfig {
	mu.RLock()
	defer mu.RUnlock()

	peers := []ypeer.Peer{}
	peerConfig := make([]PeerConfig, 0)
	DB.Where("token = ?", token).Find(&peers)
	for _, peer := range peers {
		if peer.ClientID != clientId {
			peerConfig = append(peerConfig, PeerConfig{
				IP:       peer.IP,
				Port:     peer.Port,
				ClientID: peer.ClientID,
				WgPubKey: peer.WgPubKey,
				Token:    peer.Token,
			})
		}
	}
	return peerConfig
}
