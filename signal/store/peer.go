package store

import (
	"sync"

	ypeer "yiji.one/punch/signal/store/peer"
)

var mu = sync.RWMutex{}

func Register(peer ypeer.Peer) {
	mu.Lock()
	defer mu.Unlock()
	peers := []ypeer.Peer{}
	DB.Where("token = ?", peer.Token).Find(&peers)
	for _, p := range peers {
		if p.ClientID == peer.ClientID {
			// 更新peer信息
			DB.Model(&ypeer.Peer{}).Where("token = ?", peer.Token).Where("client_id = ?", peer.ClientID).Updates(ypeer.Peer{PubKey: peer.PubKey, IP: peer.IP, Port: peer.Port})
			return
		}
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
