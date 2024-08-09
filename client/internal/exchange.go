package internal

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"time"
)

type ClientInfo struct {
	ClientId string `json:"clientId"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	PubKey   string `json:"pubKey"`
}

type ClientReq struct {
	ClientID string `json:"clientId"`
	TargetID string `json:"targetId"`
	Key      string `json:"key"`
}

const (
	DEFAULT_PORT = 51833
)

var (
	clientId string
	targetId string
)

func init() {
	flag.StringVar(&clientId, "client", "A", "client id")
	flag.StringVar(&targetId, "target", "B", "target id")
}

func ExchangePeer(port int, key string) (ClientInfo, error) {
	// 指定目标IP和端口
	targetIP := "47.91.20.205"
	targetPort := 51833

	netAddr := &net.UDPAddr{Port: port}

	var response ClientInfo

	// 创建一个UDP连接
	conn, err := net.DialUDP("udp", netAddr, &net.UDPAddr{
		IP:   net.ParseIP(targetIP),
		Port: targetPort,
	})
	if err != nil {
		return response, fmt.Errorf("error connecting: %v", err)
	}
	defer conn.Close()

	// 创建一个消息
	clientReq := ClientReq{
		ClientID: clientId,
		TargetID: targetId,
		Key:      key,
	}

	// 将消息序列化为JSON
	jsonData, err := json.Marshal(clientReq)
	if err != nil {
		return response, fmt.Errorf("error marshaling JSON: %v", err)
	}

	// 发送JSON数据
	_, err = conn.Write(jsonData)
	if err != nil {
		return response, fmt.Errorf("error sending data: %v", err)
	}

	// 读取响应
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // 设置读取超时时间
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println("Error reading data:", err.Error())
		return response, err
	}

	// 将响应反序列化为消息
	err = json.Unmarshal(buffer[:n], &response)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err.Error())
		return response, err
	}

	return response, nil
}
