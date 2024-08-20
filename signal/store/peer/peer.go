package peer

type Peer struct {
	ID       int    `gorm:"primaryKey"`
	ClientID string `json:"clientId"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	PubKey   string `json:"pubKey"`
	Token    string `json:"token"`
}
