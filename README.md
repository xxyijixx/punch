本仓库为简单实现

通过`ex-server/exchange.go`实现一个简单的IP端口交换，该服务需要运行在一台有公网IP的服务器上


客户端参数如下：

```
Usage of ./punch-cli:
  -client string
        client id (default "A")
  -ex-ip string
        exchange server ip (default "47.91.20.205")
  -ex-port int
        exchange server port (default 51833)
  -local-network string
        Local WireGuard address (default "172.16.0.1/24")
```


使用方法:
```
go run main.go -client A -ex-ip <IP端口交换服务器的IP> -ex-port <IP端口交换服务器的端口> -local-netword <Wireguard的本机网络地址> 
```


