package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"sync"
	"time"
)

type ClientReq struct {
	ClientID string
	TargetID string
	Key      string
	Type     int
	Token    string
}

type ClientInfo struct {
	ClientID string `json:"clientId"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	PubKey   string `json:"pubKey"`
	Token    string `json:"token"`
}

type RegisterClients struct {
	mu      sync.RWMutex
	clients []ClientInfo
}

var registerClients = RegisterClients{
	mu:      sync.RWMutex{},
	clients: []ClientInfo{},
}

var (
	localPort int
)

func init() {
	flag.IntVar(&localPort, "port", 51833, "local port to listen on")
	flag.Parse()
}

func (r *RegisterClients) Register(clientInfo ClientInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	clientList := r.GetClients(clientInfo.ClientID, clientInfo.Token)
	if len(clientList) > 1 {
		fmt.Println("client already registered")
		return
	}
	// 检查是否已经存在相同的客户端信息
	for idx, client := range r.clients {
		if client.ClientID == clientInfo.ClientID && client.Token == clientInfo.Token {
			r.clients[idx] = clientInfo
			return
		}
	}
	r.clients = append(r.clients, clientInfo)
}

func (r *RegisterClients) GetClients(clientId, token string) []ClientInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clients := []ClientInfo{}

	for _, client := range r.clients {
		if client.ClientID != clientId && client.Token == token {
			clients = append(clients, client)
		}
	}
	return clients
}

func main() {
	// 创建一个UDP监听器
	addr := net.UDPAddr{
		Port: localPort,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer conn.Close()

	fmt.Println("Listening on :", localPort)

	// 接受连接
	buffer := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading data:", err.Error())
			continue
		}
		go handleConnection(conn, buffer[:n], remoteAddr) // 处理连接
	}
}

func handleConnection(conn *net.UDPConn, buffer []byte, remoteAddr *net.UDPAddr) {
	// 将JSON数据反序列化为消息
	var clientReq ClientReq
	err := json.Unmarshal(buffer, &clientReq)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err.Error())
		return
	}

	ip := remoteAddr.IP.String()
	port := remoteAddr.Port

	clientInfo := ClientInfo{
		IP:       ip,
		Port:     port,
		ClientID: clientReq.ClientID,
		PubKey:   clientReq.Key,
		Token:    clientReq.Token,
	}

	fmt.Printf("Received: %#v type: %d\n", clientInfo, clientReq.Type)
	if clientReq.Type == 1 {
		// 注册客户端
		registerClients.Register(clientInfo)
		// return
	}

	// 打印消息

	for {
		clients := registerClients.GetClients(clientInfo.ClientID, clientReq.Token)
		if len(clients) > 0 {
			responseData, _ := json.Marshal(clients)

			_, err = conn.WriteToUDP(responseData, remoteAddr)
			if err != nil {
				fmt.Println("Error sending data:", err.Error())
			}
		}
		// 如果目标客户端还没有注册，等待一段时间再检查
		time.Sleep(500 * time.Millisecond)
	}

}
