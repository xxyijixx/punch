package main

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

type ClientReq struct {
	ClientID string
	TargetID string
	Key      string
}

type ClientInfo struct {
	ClientID string `json:"clientId"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	PubKey   string `json:"pubKey"`
}

var clientsMap sync.Map

func main() {
	// 创建一个UDP监听器
	addr := net.UDPAddr{
		Port: 51833,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer conn.Close()

	fmt.Println("Listening on :51833")

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
	}

	clientsMap.Store(clientInfo.ClientID, clientInfo)

	// 打印消息
	fmt.Println("Received:", clientInfo)

	// 轮询等待目标客户端注册
	for {
		if targetInfo, ok := clientsMap.Load(clientReq.TargetID); ok {
			targetClientInfo := targetInfo.(ClientInfo)

			// 交换信息并返回给当前客户端
			responseData, _ := json.Marshal(targetClientInfo)

			// 发送JSON数据
			_, err = conn.WriteToUDP(responseData, remoteAddr)
			if err != nil {
				fmt.Println("Error sending data:", err.Error())
				return
			}
			return
		}

		// 如果目标客户端还没有注册，等待一段时间再检查
		time.Sleep(500 * time.Millisecond)
	}
}
