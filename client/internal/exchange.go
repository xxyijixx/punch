package internal

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

type MessageType int

type Message struct {
	Type  MessageType `json:"type"`
	Token string      `json:"token"`
	// Header interface{} `json:"header"`
	Body interface{} `json:"body"`
}

const (
	Login     MessageType = 1
	KeepAlive MessageType = 2
	Query     MessageType = 3
)

type PeerLoginReq struct {
	ClientID  string `json:"clientId"`
	WgPubKey  string `json:"wgPubKey"`
	AllowedIP string `json:"allowedIp"`
}

type PeerQueryReq struct {
	ClientID string `json:"clientId"`
}

type PeerInfo struct {
	ClientId        string `json:"clientId"`
	IP              string `json:"ip"`
	Port            int    `json:"port"`
	WgPubKey        string `json:"wgPubKey"`
	AllowedIP       string `json:"allowedIp"`
	LastKeepAliveAt string `json:"lastKeepAliveAt"`
}

type PeerLoginRes struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

var (
	clientId        string
	signalIpAddress string
	token           string
)

func init() {
	flag.StringVar(&clientId, "client", "A", "client identity")
	flag.StringVar(&signalIpAddress, "signal", "47.91.20.205:51833", "signal server ip:port")
	flag.StringVar(&token, "token", "123456", "user token")
}

// GetSignalServer 获取信令端ip和端口
func GetSignalServer() (string, int) {
	signalHost, signalPort, err := net.SplitHostPort(signalIpAddress)
	if err != nil {
		panic(err)
	}
	sPort, err := strconv.Atoi(signalPort)
	if err != nil {
		panic(err)
	}
	return signalHost, sPort
}

func ClientRegister(port int, wgPubKey, localIP string) (PeerLoginRes, error) {
	log.Info("register ing...")
	netAddr := &net.UDPAddr{Port: port}
	// 使用随机端口
	// netAddr := &net.UDPAddr{}
	var response PeerLoginRes

	signalHost, signalPort := GetSignalServer()

	// 创建一个UDP连接
	conn, err := net.DialUDP("udp", netAddr, &net.UDPAddr{
		IP:   net.ParseIP(signalHost),
		Port: signalPort,
	})
	if err != nil {
		return response, fmt.Errorf("error connecting: %v", err)
	}
	defer conn.Close()

	ip, _, err := net.ParseCIDR(localIP)
	if err != nil {
		return response, err
	}

	// 创建一个消息
	message := Message{
		Type:  Login,
		Token: token,
		Body: PeerLoginReq{
			ClientID:  clientId,
			WgPubKey:  wgPubKey,
			AllowedIP: ip.String(),
		},
	}
	log.Infof("send message: %v\n", message)

	// 将消息序列化为JSON
	jsonData, err := json.Marshal(message)
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
	var result []byte
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return response, fmt.Errorf("error reading data: %v", err)
		}

		result = append(result, buffer[:n]...)

		// 检查是否读取完所有数据
		if n < len(buffer) {
			break
		}
	}
	// 将响应反序列化为消息
	err = json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err.Error())
		return response, err
	}

	return response, nil
}

func GetRemotePeers(clientId, token string) ([]PeerInfo, error) {
	// 指定目标IP和端口
	netAddr := &net.UDPAddr{}
	// netAddr := &net.UDPAddr{Port: 51822}

	var response []PeerInfo

	signalHost, signalPort := GetSignalServer()

	// 创建一个UDP连接
	conn, err := net.DialUDP("udp", netAddr, &net.UDPAddr{
		IP:   net.ParseIP(signalHost),
		Port: signalPort,
	})
	if err != nil {
		return response, fmt.Errorf("error connecting: %v", err)
	}
	defer conn.Close()

	// 创建一个消息
	message := Message{
		Type:  Query,
		Token: token,
		Body: PeerQueryReq{
			ClientID: clientId,
		},
	}

	// 将消息序列化为JSON
	jsonData, err := json.Marshal(message)
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
	conn.SetReadDeadline(time.Now().Add(10 * time.Second)) // 设置读取超时时间
	var result []byte
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return response, fmt.Errorf("error reading data: %v", err)
		}

		result = append(result, buffer[:n]...)

		// 检查是否读取完所有数据
		if n < len(buffer) {
			break
		}
	}
	// 将响应反序列化为消息
	err = json.Unmarshal(result, &response)
	if err != nil {
		return response, fmt.Errorf("error unmarshaling JSON: %v", err)
	}
	return response, nil
}
