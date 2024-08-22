package server

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
	"yiji.one/punch/signal/store"
)

type MessageType int

const (
	Login     MessageType = 1
	KeepAlive MessageType = 2
	Query     MessageType = 3
)

type Message struct {
	Type  MessageType `json:"type"`
	Token string      `json:"token"`
	Body  interface{} `json:"body"`
}
type PeerLoginReq struct {
	ClientID  string `json:"clientId"`
	WgPubKey  string `json:"wgPubKey"`
	AllowedIP string `json:"allowedIp"`
}

type PeerQueryReq struct {
	ClientID string `json:"clientId"`
}

type RegisterReq struct {
	ClientID  string `json:"clientId"`
	WgPubKey  string `json:"wgPubKey"`
	Type      int    `json:"type"`
	Token     string `json:"token"`
	AllowedIP string `json:"allowedIp"`
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
	// var clientReq RegisterReq
	var message Message
	err := json.Unmarshal(buffer, &message)
	log.Infof("Received: %#v\n", message)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err.Error())
		return
	}

	var responseData []byte
	if message.Type == Login {
		responseData = handleLoginRequest(message, remoteAddr)
	} else if message.Type == Query {
		responseData = handleQueryRequest(message)
	}
	// log.Infof("Received: %#v type: %v\n", peerLogin, clientReq.Type)
	_, err = conn.WriteToUDP(responseData, remoteAddr)
	if err != nil {
		fmt.Println("Error sending data:", err.Error())
	}

}

// handleLoginRequest 处理登录请求
func handleLoginRequest(message Message, remoteAddr *net.UDPAddr) (responseData []byte) {
	log.Info("handle login request")
	ip := remoteAddr.IP.String()
	port := remoteAddr.Port

	var peerLoginReq PeerLoginReq
	// peerLoginReq, ok := message.Body.(PeerLoginReq)
	jsonData, err := json.Marshal(message.Body)
	if err != nil {
		log.Error("Json序列化失败", err)
		return
	}
	err = json.Unmarshal(jsonData, &peerLoginReq)
	if err != nil {
		log.Error("类型转换", err)
		return
	}
	peerLogin := store.PeerLogin{
		IP:        ip,
		Port:      port,
		ClientID:  peerLoginReq.ClientID,
		WgPubKey:  peerLoginReq.WgPubKey,
		Token:     message.Token,
		AllowedIP: peerLoginReq.AllowedIP,
	}
	store.Register(peerLogin)
	responseData, _ = json.Marshal(&RegisterResponse{
		IP:   ip,
		Port: port,
	})

	return
}

// handleQueryRequest 处理查询请求
func handleQueryRequest(message Message) (responseData []byte) {
	log.Info("handle query request")
	var peerQueryReq PeerQueryReq
	jsonData, err := json.Marshal(message.Body)
	if err != nil {
		log.Error("Json序列化失败", err)
		return
	}
	err = json.Unmarshal(jsonData, &peerQueryReq)
	if err != nil {
		log.Error("类型转换", err)
		return
	}

	peerConfig := store.GetClients(peerQueryReq.ClientID, message.Token)
	responseData, _ = json.Marshal(peerConfig)

	return
}
