package server

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"

	"yiji.one/punch/signal/store"
	ypeer "yiji.one/punch/signal/store/peer"
)

type RegisterReq struct {
	ClientID string
	TargetID string
	Key      string
	Type     int
	Token    string
}

type RegisterResponse struct {
	Ip   string
	Port int
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
	var clientReq RegisterReq
	err := json.Unmarshal(buffer, &clientReq)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err.Error())
		return
	}

	ip := remoteAddr.IP.String()
	port := remoteAddr.Port

	peer := ypeer.Peer{
		IP:       ip,
		Port:     port,
		ClientID: clientReq.ClientID,
		PubKey:   clientReq.Key,
		Token:    clientReq.Token,
	}

	fmt.Printf("Received: %#v type: %d\n", peer, clientReq.Type)
	if clientReq.Type == 1 {
		// 注册客户端
		store.Register(peer)
	}
	// 打印消息
	responseData, _ := json.Marshal(&RegisterResponse{
		Ip:   ip,
		Port: port,
	})
	_, err = conn.WriteToUDP(responseData, remoteAddr)
	if err != nil {
		fmt.Println("Error sending data:", err.Error())
	}

}
