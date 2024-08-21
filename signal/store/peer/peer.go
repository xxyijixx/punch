package peer

import "time"

type Peer struct {
	ID              int       `gorm:"primaryKey"`
	ClientID        string    `json:"clientId"`
	IP              string    `json:"ip"`
	Port            int       `json:"port"`
	WgPubKey        string    `json:"wgPubKey"`
	Token           string    `json:"token"`
	LastKeepAliveAt time.Time `json:"lastKeepAliveAt"`
}
