package internal

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"
)

type PeerInfo struct {
	ClientId        string `json:"clientId"`
	IP              string `json:"ip"`
	Port            int    `json:"port"`
	WgPubKey        string `json:"wgPubKey"`
	Token           string `json:"token"`
	LastKeepAliveAt string `json:"lastKeepAliveAt"`
}

type PeerLoginReq struct {
	ClientID string `json:"clientId"`
	TargetID string `json:"targetId"`
	WgPubKey string `json:"wgPubKey"`
	Type     int    `json:"type"`
	Token    string `json:"token"`
}

type PeerLoginRes struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

const (
	DEFAULT_PORT = 51833
)

var (
	clientId        string
	targetId        string
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

func ClientRegister(port int, wgPubKey string) (PeerLoginRes, error) {

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

	// 创建一个消息
	clientReq := PeerLoginReq{
		ClientID: clientId,
		TargetID: targetId,
		WgPubKey: wgPubKey,
		Type:     1,
		Token:    token,
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
	var result []byte
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return response, fmt.Errorf("Error reading data:", err.Error())
		}

		result = append(result, buffer[:n]...)

		// 检查是否读取完所有数据
		if n < len(buffer) {
			break
		}
	}
	// 将响应反序列化为消息
	err = json.Unmarshal(result, &response)
	fmt.Println("DD ", string(result))
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err.Error())
		return response, err
	}

	return response, nil
}

func GetRemotePeers(clientId, token string) ([]PeerInfo, error) {
	// 指定目标IP和端口
	netAddr := &net.UDPAddr{}

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
	clientReq := PeerLoginReq{
		ClientID: clientId,
		WgPubKey: "",
		Type:     0,
		Token:    token,
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
