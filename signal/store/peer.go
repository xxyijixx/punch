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

var mu = sync.RWMutex{}

func Register(peerLogin PeerLogin) {
	mu.Lock()
	defer mu.Unlock()
	peers := []ypeer.Peer{}
	DB.Where("token = ?", peerLogin.Token).Find(&peers)
	for _, peer := range peers {
		if peer.ClientID == peerLogin.ClientID {
			// 更新peer信息
			DB.Model(&ypeer.Peer{}).Where("token = ?", peerLogin.Token).Where("client_id = ?", peerLogin.ClientID).Updates(ypeer.Peer{WgPubKey: peer.WgPubKey, IP: peer.IP, Port: peer.Port})
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

func GetClients(clientId, token string) []ypeer.Peer {
	mu.RLock()
	defer mu.RUnlock()

	peers := []ypeer.Peer{}
	DB.Where("token = ?", token).Find(&ypeer.Peer{})
	for i, peer := range peers {
		if peer.ClientID == clientId {
			peers = append(peers[:i], peers...)
			return peers
		}
	}
	return peers
}