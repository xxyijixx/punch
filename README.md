## 简介

可以通过该项目实现 wireguard 节点的连接，在这里实现了一个简单的信令服务，用于交换双方的公钥、端点信息，从而实现节点之间的连接。

<div>
<img src="https://github.com/fluidicon.png" atl="github" style="width: 24px; height: 24px;"> <span>Github地址：https://github.com/Yijixx/punch</span>
</div>
## 目录结构

```plaintext
|____util              # 实用工具库，包含网络相关的通用功能。
| |____net             # 与网络操作相关的工具和功能，可能包括跨平台的网络抽象层。
|____iface             # 网络接口层，负责管理不同操作系统的网络接口配置。
| |____freebsd         # FreeBSD 平台特定的网络接口实现。
| |____netstack        # 基于网络栈的接口实现，可能用于模拟网络设备或堆栈。
| |____bind            # 网络绑定相关的实现。
|____sharedsock        # 共享sock的实现，支持在多个进程或线程间共享网络连接。
|____client            # 客户端模块。
| |____internal        # 客户端的内部实现细节。
| | |____stdnet        # 标准网络功能，可能包含标准网络接口的实现或抽象。
| | |____peer          # 处理客户端之间的直接通信。
| | |____wgproxy       # WireGuard 代理功能的实现。
| | |____ebpf          # eBPF 相关功能，用于在Linux平台上进行高效的数据包处理或监控。
| | | |____manager
| | | |____ebpf
| | | | |____src       # 存放C语言编写的eBPF程序源文件。
|____signal            # 信令模块，负责客户端之间的信令通信，用于建立连接。
```

## 使用方法

1. 启动信令服务（需要有公网 IP）

将`signal/main.go`这个文件放置在服务器上直接运行，或者通过构建之后上传至服务器运行

2. 启动客户端

> 代码中 AllowIps 为`172.16.0.0/24`，wireguard 地址需要在这个网段之中，或者自行修改

```bash
cd client

go build -o client

./client -client <客户端标识> -signal <信令服务器IP:PORT> -local-network <Wireguard的地址>
```

## 运行效果

![](https://vip.helloimg.com/i/2024/08/19/66c2c1b5ee82c.png)
