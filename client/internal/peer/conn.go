package peer

import (
	"context"
	"flag"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/ice/v3"
	"github.com/pion/stun/v2"
	log "github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"yiji.one/punch/client/internal/stdnet"
	"yiji.one/punch/client/internal/wgproxy"
	"yiji.one/punch/iface"
	"yiji.one/punch/iface/bind"
)

const (
	iceKeepAliveDefault           = 4 * time.Second
	iceDisconnectedTimeoutDefault = 6 * time.Second
	// iceRelayAcceptanceMinWaitDefault is the same as in the Pion ICE package
	iceRelayAcceptanceMinWaitDefault = 2 * time.Second

	defaultWgKeepAlive = 2 * time.Second //25 * time.Second
)

var (
	defaultRemoteWGPort = 51822
	defaultRemoteAddr   = "47.91.20.205"
)

var (
	currentRemoteWgPort int
	currentRemoteAddr   string
)

func init() {
	flag.IntVar(&currentRemoteWgPort, "p", defaultRemoteWGPort, "Remote WireGuard port")
	flag.StringVar(&currentRemoteAddr, "addr", defaultRemoteAddr, "Remote WireGuard address")
}

type WgConfig struct {
	WgListenPort int
	RemoteKey    string
	WgInterface  *iface.WGIface
	AllowedIps   string
	PreSharedKey *wgtypes.Key
}

// ConnConfig is a peer Connection configuration
type ConnConfig struct {

	// Key is a public key of a remote peer
	Key string
	// LocalKey is a public key of a local peer
	LocalKey string

	// StunTurn is a list of STUN and TURN URLs
	StunTurn []*stun.URI

	// InterfaceBlackList is a list of machine interfaces that should be filtered out by ICE Candidate gathering
	// (e.g. if eth0 is in the list, host candidate of this interface won't be used)
	InterfaceBlackList   []string
	DisableIPv6Discovery bool

	Timeout time.Duration

	WgConfig WgConfig

	UDPMux      ice.UDPMux
	UDPMuxSrflx ice.UniversalUDPMux

	LocalWgPort int

	NATExternalIPs []string

	// RosenpassPubKey is this peer's Rosenpass public key
	RosenpassPubKey []byte
	// RosenpassPubKey is this peer's RosenpassAddr server address (IP:port)
	RosenpassAddr string
}

// OfferAnswer represents a session establishment offer or answer
type OfferAnswer struct {
	IceCredentials IceCredentials
	// WgListenPort is a remote WireGuard listen port.
	// This field is used when establishing a direct WireGuard connection without any proxy.
	// We can set the remote peer's endpoint with this port.
	WgListenPort int

	// Version of NetBird Agent
	Version string
	// RosenpassPubKey is the Rosenpass public key of the remote peer when receiving this message
	// This value is the local Rosenpass server public key when sending the message
	RosenpassPubKey []byte
	// RosenpassAddr is the Rosenpass server address (IP:port) of the remote peer when receiving this message
	// This value is the local Rosenpass server address when sending the message
	RosenpassAddr string
}

// IceCredentials ICE protocol credentials struct
type IceCredentials struct {
	UFrag string
	Pwd   string
}

type Conn struct {
	config ConnConfig
	mu     sync.Mutex

	closeCh            chan struct{}
	ctx                context.Context
	notifyDisconnected context.CancelFunc

	agent *ice.Agent

	wgProxyFactory *wgproxy.Factory
	wgProxy        wgproxy.Proxy

	adapter       iface.TunAdapter
	iFaceDiscover stdnet.ExternalIFaceDiscover
}

// WgConfig returns the WireGuard config
func (conn *Conn) WgConfig() WgConfig {
	return conn.config.WgConfig
}

// NewConn creates a new not opened Conn to the remote peer.
// To establish a connection run Conn.Open
func NewConn(config ConnConfig, wgProxyFactory *wgproxy.Factory) (*Conn, error) {
	return &Conn{
		config:         config,
		mu:             sync.Mutex{},
		closeCh:        make(chan struct{}),
		wgProxyFactory: wgProxyFactory,
	}, nil
}

func (conn *Conn) reCreateAgent() error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	failedTimeout := 6 * time.Second

	var err error
	transportNet, err := conn.newStdNet()
	if err != nil {
		log.Errorf("failed to create pion's stdnet: %s", err)
	}

	iceKeepAlive := iceKeepAlive()
	iceDisconnectedTimeout := iceDisconnectedTimeout()
	iceRelayAcceptanceMinWait := iceRelayAcceptanceMinWait()

	agentConfig := &ice.AgentConfig{
		MulticastDNSMode:       ice.MulticastDNSModeDisabled,
		NetworkTypes:           []ice.NetworkType{ice.NetworkTypeUDP4, ice.NetworkTypeUDP6},
		Urls:                   conn.config.StunTurn,
		FailedTimeout:          &failedTimeout,
		InterfaceFilter:        stdnet.InterfaceFilter(conn.config.InterfaceBlackList),
		UDPMux:                 conn.config.UDPMux,
		UDPMuxSrflx:            conn.config.UDPMuxSrflx,
		NAT1To1IPs:             conn.config.NATExternalIPs,
		Net:                    transportNet,
		DisconnectedTimeout:    &iceDisconnectedTimeout,
		KeepaliveInterval:      &iceKeepAlive,
		RelayAcceptanceMinWait: &iceRelayAcceptanceMinWait,
	}

	if conn.config.DisableIPv6Discovery {
		agentConfig.NetworkTypes = []ice.NetworkType{ice.NetworkTypeUDP4}
	}

	conn.agent, err = ice.NewAgent(agentConfig)
	if err != nil {
		return err
	}

	return nil
}

// Open opens connection to the remote peer starting ICE candidate gathering process.
// Blocks until connection has been closed or connection timeout.
// ConnStatus will be set accordingly
func (conn *Conn) Open(ctx context.Context, remoteAddr string, remoteWgPort int) error {
	log.Debugf("trying to connect to peer %s", conn.config.Key)
	var err error

	// err = conn.reCreateAgent()
	// if err != nil {
	// 	return err
	// }
	conn.mu.Lock()
	conn.ctx, conn.notifyDisconnected = context.WithCancel(ctx)
	defer conn.notifyDisconnected()
	conn.mu.Unlock()

	log.Infof("connection offer sent to peer %s, waiting for the confirmation", conn.config.Key)
	// dynamically set remote WireGuard port if other side specified a different one from the default one
	// remoteWgPort := currentRemoteWgPort
	// remoteAddr := currentRemoteAddr
	// the ice connection has been established successfully so we are ready to start the proxy
	endpoint, err := conn.configureConnection(remoteAddr, remoteWgPort)
	if err != nil {
		log.Info("configre error", err)
		return err
	}

	log.Infof("connected to peer [%s], endpoint address: %s", conn.config.Key, endpoint.String())

	// return nil
	// wait until connection disconnected or has been closed externally (upper layer, e.g. engine)
	select {
	case <-conn.closeCh:
		log.Info("Conn close ", conn.config.Key)
		// closed externally
		return NewConnectionClosedError(conn.config.Key)
	case <-conn.ctx.Done():
		log.Info("Conn ctx done ", conn.config.Key)
		// disconnected from the remote peer
		return NewConnectionDisconnectedError(conn.config.Key)
	}
}

// configureConnection starts proxying traffic from/to local Wireguard and sets connection status to StatusConnected
func (conn *Conn) configureConnection(remoteAddr string, remoteWgPort int) (net.Addr, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.wgProxy = conn.wgProxyFactory.GetProxy(conn.ctx)
	log.Infof("ConfigureConnection %s, %d, %v", remoteAddr, remoteWgPort, conn.wgProxy)
	go conn.punchRemoteWGPort(remoteAddr, remoteWgPort)
	var endpoint = &net.UDPAddr{
		IP:   net.ParseIP(remoteAddr),
		Port: remoteWgPort,
	}

	endpointUdpAddr, _ := net.ResolveUDPAddr(endpoint.Network(), endpoint.String())
	log.Debugf("Conn resolved IP for %s: %s", endpoint, endpointUdpAddr.IP)

	err := conn.config.WgConfig.WgInterface.UpdatePeer(conn.config.WgConfig.RemoteKey, conn.config.WgConfig.AllowedIps, defaultWgKeepAlive, endpointUdpAddr, conn.config.WgConfig.PreSharedKey)
	if err != nil {
		log.Info("Error conn")
	}
	// if err != nil {
	// 	if conn.wgProxy != nil {
	// 		if err := conn.wgProxy.CloseConn(); err != nil {
	// 			log.Warnf("Failed to close turn connection: %v", err)
	// 		}
	// 	}
	// 	return nil, fmt.Errorf("update peer: %w", err)
	// }

	// if err != nil {
	// 	log.Warnf("unable to save peer's state, got error: %v", err)
	// }

	// if runtime.GOOS == "ios" {
	// 	runtime.GC()
	// }

	return endpoint, nil
}

func (conn *Conn) punchRemoteWGPort(remoteAddr string, remoteWgPort int) {

	// wait local endpoint configuration
	time.Sleep(time.Second)
	log.Infof("punchRemoteWGPort addr: %s port: %d", remoteAddr, remoteWgPort)
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", remoteAddr, remoteWgPort))
	if err != nil {
		log.Warnf("got an error while resolving the udp address, err: %s", err)
		return
	}

	mux, ok := conn.config.UDPMuxSrflx.(*bind.UniversalUDPMuxDefault)
	if !ok {
		log.Warn("invalid udp mux conversion")
		return
	}
	_, err = mux.GetSharedConn().WriteTo([]byte{0x6e, 0x62}, addr)
	if err != nil {
		log.Warnf("got an error while sending the punch packet, err: %s", err)
	}
	// mux.ReadFromConn(conn.ctx)
}

// cleanup closes all open resources and sets status to StatusDisconnected
func (conn *Conn) cleanup() error {
	log.Debugf("trying to cleanup %s", conn.config.Key)
	conn.mu.Lock()
	defer conn.mu.Unlock()

	var err1, err2 error
	if conn.agent != nil {
		err1 = conn.agent.Close()
		if err1 == nil {
			conn.agent = nil
		}
	}

	if conn.wgProxy != nil {
		err2 = conn.wgProxy.CloseConn()
		conn.wgProxy = nil
	}

	return err2
}

// Close closes this peer Conn issuing a close event to the Conn closeCh
func (conn *Conn) Close() error {
	conn.mu.Lock()
	conn.cleanup()
	defer conn.mu.Unlock()
	select {
	case conn.closeCh <- struct{}{}:
		return nil
	default:
		// probably could happen when peer has been added and removed right after not even starting to connect
		// todo further investigate
		// this really happens due to unordered messages coming from management
		// more importantly it causes inconsistency -> 2 Conn objects for the same peer
		// e.g. this flow:
		// update from management has peers: [1,2,3,4]
		// engine creates a Conn for peers:  [1,2,3,4] and schedules Open in ~1sec
		// before conn.Open() another update from management arrives with peers: [1,2,3]
		// engine removes peer 4 and calls conn.Close() which does nothing (this default clause)
		// before conn.Open() another update from management arrives with peers: [1,2,3,4,5]
		// engine adds a new Conn for 4 and 5
		// therefore peer 4 has 2 Conn objects
		log.Warnf("Connection has been already closed or attempted closing not started connection %s", conn.config.Key)
		return NewConnectionAlreadyClosed(conn.config.Key)
	}
}
