package internal

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/pion/stun/v2"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/rand"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"yiji.one/punch/client/internal/ebpf/manager"
	"yiji.one/punch/client/internal/peer"
	"yiji.one/punch/client/internal/wgproxy"
	"yiji.one/punch/iface"
	"yiji.one/punch/iface/bind"
)

// PeerConnectionTimeoutMax is a timeout of an initial connection attempt to a remote peer.
// E.g. this peer will wait PeerConnectionTimeoutMax for the remote peer to respond,
// if not successful then it will retry the connection attempt.
// Todo pass timeout at EnginConfig
const (
	PeerConnectionTimeoutMax = 45000 // ms
	PeerConnectionTimeoutMin = 30000 // ms
)

var ErrResetConnection = fmt.Errorf("reset connection")

// EngineConfig is a config for the Engine
type EngineConfig struct {
	WgPort      int
	WgIfaceName string

	// WgAddr is a Wireguard local address (Netbird Network IP)
	WgAddr string

	// WgPrivateKey is a Wireguard private key of our peer (it MUST never leave the machine)
	WgPrivateKey wgtypes.Key

	// IFaceBlackList is a list of network interfaces to ignore when discovering connection candidates (ICE related)
	IFaceBlackList []string

	// UDPMuxPort default value 0 - the system will pick an available port
	UDPMuxPort int

	// UDPMuxSrflxPort default value 0 - the system will pick an available port
	UDPMuxSrflxPort int
}

// Engine is a mechanism responsible for reacting on Signal and Management stream events and managing connections to the remote peers.
type Engine struct {
	config *EngineConfig

	// STUNs is a list of STUN servers used by ICE
	STUNs []*stun.URI
	// TURNs is a list of STUN servers used by ICE
	TURNs []*stun.URI

	clientCtx    context.Context
	clientCancel context.CancelFunc

	ctx    context.Context
	cancel context.CancelFunc

	wgInterface    *iface.WGIface
	wgProxyFactory *wgproxy.Factory

	udpMux *bind.UniversalUDPMuxDefault

	peerConns map[string]*peer.Conn

	firewall manager.Manager

	wgConnWorker sync.WaitGroup
}

// Peer is an instance of the Connection Peer
type Peer struct {
	WgPubKey     string
	WgAllowedIps string
}

// NewEngine creates a new Connection Engine
func NewEngine(
	clientCtx context.Context,
	clientCancel context.CancelFunc,
	config *EngineConfig,
) *Engine {
	return &Engine{
		config:       config,
		clientCtx:    clientCtx,
		peerConns:    make(map[string]*peer.Conn),
		clientCancel: clientCancel,
	}
}

// 创建Wireguard接口，并不会立刻创建Peer连接（原来需要接收Management建立请求才会创建）
func (e *Engine) Start() error {
	e.ctx, e.cancel = context.WithCancel(e.clientCtx)

	wgIface, err := e.newWgIface()
	if err != nil {
		log.Errorf("failed creating wireguard interface instance %s: [%s]", e.config.WgIfaceName, err)
		return fmt.Errorf("new wg interface: %w", err)
	}
	e.wgInterface = wgIface

	userspace := e.wgInterface.IsUserspaceBind()
	log.Info("userspace bind: ", userspace)
	e.wgProxyFactory = wgproxy.NewFactory(e.ctx, userspace, e.config.WgPort)

	err = e.wgInterfaceCreate()
	if err != nil {
		log.Errorf("failed creating tunnel interface %s: [%s]", e.config.WgIfaceName, err.Error())
		e.close()
		return fmt.Errorf("create wg interface: %w", err)
	}
	e.udpMux, err = e.wgInterface.Up()
	if err != nil {
		log.Errorf("failed to pull up wgInterface [%s]: %s", e.wgInterface.Name(), err.Error())
		e.close()
		return fmt.Errorf("up wg interface: %w", err)
	}

	return nil
}

func (e *Engine) Stop() (err error) {
	e.close()
	return nil
}

func (e *Engine) close() {
	if e.wgProxyFactory != nil {
		if err := e.wgProxyFactory.Free(); err != nil {
			log.Errorf("failed closing ebpf proxy: %s", err)
		}
	}
	log.Debugf("removing Netbird interface %s", e.config.WgIfaceName)
	if e.wgInterface != nil {
		if err := e.wgInterface.Close(); err != nil {
			log.Errorf("failed closing Netbird interface %s %v", e.config.WgIfaceName, err)
		}
	}
}

func (e *Engine) newWgIface() (*iface.WGIface, error) {
	transportNet, err := e.newStdNet()
	if err != nil {
		log.Errorf("failed to create pion's stdnet: %s", err)
	}

	return iface.NewWGIFace(e.config.WgIfaceName, e.config.WgAddr, e.config.WgPort, e.config.WgPrivateKey.String(), iface.DefaultMTU, transportNet, nil, e.addrViaRoutes)
}

func (e *Engine) wgInterfaceCreate() (err error) {

	err = e.wgInterface.Create()

	return err
}

func findIPFromInterfaceName(ifaceName string) (net.IP, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, err
	}
	return findIPFromInterface(iface)
}

func findIPFromInterface(iface *net.Interface) (net.IP, error) {
	ifaceAddrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range ifaceAddrs {
		if ipv4Addr := addr.(*net.IPNet).IP.To4(); ipv4Addr != nil {
			return ipv4Addr, nil
		}
	}
	return nil, fmt.Errorf("interface %s don't have an ipv4 address", iface.Name)
}

func (e *Engine) addrViaRoutes(addr netip.Addr) (bool, netip.Prefix, error) {
	return false, netip.Prefix{}, nil
}

// IsWGIfaceUp checks if the WireGuard interface is up.
func (e *Engine) IsWGIfaceUp() bool {
	// If the Engine is nil or the WireGuard interface is nil, return false.
	if e == nil || e.wgInterface == nil {
		return false
	}
	// Get the interface by name.
	iface, err := net.InterfaceByName(e.wgInterface.Name())
	// If there is an error, log it and return false.
	if err != nil {
		log.Debugf("failed to get interface by name %s: %v", e.wgInterface.Name(), err)
		return false
	}

	// If the interface is up, return true.
	if iface.Flags&net.FlagUp != 0 {
		return true
	}

	// If the interface is not up, return false.
	return false
}

func (e *Engine) createPeerConn(pubKey string, allowedIPs string) (*peer.Conn, error) {
	log.Debugf("creating peer connection %s", pubKey)
	var stunTurn []*stun.URI
	stunTurn = append(stunTurn, e.STUNs...)
	stunTurn = append(stunTurn, e.TURNs...)

	wgConfig := peer.WgConfig{
		RemoteKey:    pubKey,
		WgListenPort: e.config.WgPort,
		WgInterface:  e.wgInterface,
		AllowedIps:   allowedIPs,
	}

	// randomize connection timeout
	timeout := time.Duration(rand.Intn(PeerConnectionTimeoutMax-PeerConnectionTimeoutMin)+PeerConnectionTimeoutMin) * time.Millisecond
	config := peer.ConnConfig{
		Key:                  pubKey,
		LocalKey:             e.config.WgPrivateKey.PublicKey().String(),
		StunTurn:             stunTurn,
		InterfaceBlackList:   e.config.IFaceBlackList,
		DisableIPv6Discovery: true,
		Timeout:              timeout,
		UDPMux:               e.udpMux.UDPMuxDefault,
		UDPMuxSrflx:          e.udpMux,
		WgConfig:             wgConfig,
		LocalWgPort:          e.config.WgPort,
	}

	peerConn, err := peer.NewConn(config, e.wgProxyFactory)
	if err != nil {
		return nil, err
	}
	return peerConn, nil
}

type RemotePeerConfig struct {
	WgPubKey   string
	AllowedIps []string
}
