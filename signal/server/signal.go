package server

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
	"yiji.one/punch/signal/store"
)

type RegisterReq struct {
	ClientID string
	WgPubKey string
	Type     int
	Token    string
}

type RegisterResponse struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

var (
	localPort int
)

func init() {
	flag.IntVar(&localPort, "port", 51833, "local port to listen on")
	flag.Parse()
}

func Run() {
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
	log.Info("Listening on :", localPort)
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
	var clientReq RegisterReq
	err := json.Unmarshal(buffer, &clientReq)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err.Error())
		return
	}

	ip := remoteAddr.IP.String()
	port := remoteAddr.Port

	peerLogin := store.PeerLogin{
		IP:       ip,
		Port:     port,
		ClientID: clientReq.ClientID,
		WgPubKey: clientReq.WgPubKey,
		Token:    clientReq.Token,
	}

	log.Infof("Received: %#v\n", peerLogin)
	var responseData []byte
	if clientReq.Type == 1 {
		// 注册客户端
		store.Register(peerLogin)
		responseData, _ = json.Marshal(&RegisterResponse{
			IP:   ip,
			Port: port,
		})
	} else if clientReq.Type == 0 {
		// 查询客户端
		peerLogin := store.GetClients(clientReq.ClientID, clientReq.Token)
		responseData, _ = json.Marshal(&peerLogin)
	}
	_, err = conn.WriteToUDP(responseData, remoteAddr)
	if err != nil {
		fmt.Println("Error sending data:", err.Error())
	}

}
