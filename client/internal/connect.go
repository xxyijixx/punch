package internal

import (
	"context"
	"errors"
	"flag"
	"net"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"yiji.one/punch/iface"
)

var (
	localWgAddr string
)

func init() {
	flag.StringVar(&localWgAddr, "local-network", "172.16.0.1/24", "Local WireGuard address")
}

type ConnectClient struct {
	ctx         context.Context
	config      *Config
	engine      *Engine
	engineMutex sync.Mutex
}

func NewConnectClient(
	ctx context.Context,
	config *Config,
) *ConnectClient {
	return &ConnectClient{
		ctx:         ctx,
		config:      config,
		engineMutex: sync.Mutex{},
	}
}

// Run with main logic.
func (c *ConnectClient) Run() error {
	return c.run()
}

func (c *ConnectClient) run() error {
	defer func() {
		if r := recover(); r != nil {
			log.Panicf("Panic occurred: %v, stack trace: %s", r, string(debug.Stack()))
		}
	}()

	log.Infof("starting NetBird client version %s on %s/%s", "NB", runtime.GOOS, runtime.GOARCH)

	backOff := &backoff.ExponentialBackOff{
		InitialInterval:     time.Second,
		RandomizationFactor: 1,
		Multiplier:          1.7,
		MaxInterval:         15 * time.Second,
		MaxElapsedTime:      3 * 30 * 24 * time.Hour, // 3 months
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}

	myPrivateKey, err := wgtypes.ParseKey(c.config.PrivateKey)
	if err != nil {
		log.Errorf("failed parsing Wireguard key %s: [%s]", c.config.PrivateKey, err.Error())
		return err
	}

	pubKey := myPrivateKey.PublicKey().String()

	operation := func() error {
		// if context cancelled we not start new backoff cycle
		select {
		case <-c.ctx.Done():
			return nil
		default:
		}
		// ctx := context.WithValue(context.TODO(), "k1", "v1")
		engineCtx, cancel := context.WithCancel(c.ctx)
		// engineCtx, cancel := context.WithCancel(ctx)
		defer func() {
			cancel()
		}()

		engineConfig, err := createEngineConfig(myPrivateKey, c.config)
		if err != nil {
			return err
		}
		// 注册客户端
		loginResponse, err := ClientRegister(engineConfig.WgPort, pubKey, engineConfig.WgAddr)
		if err != nil {
			return err
		}
		log.Info("登录", loginResponse)
		c.engineMutex.Lock()
		c.engine = NewEngine(engineCtx, cancel, engineConfig)
		c.engineMutex.Unlock()

		err = c.engine.Start()

		if err != nil {
			log.Errorf("error while starting Netbird Connection Engine: %s", err)
			return err
		}
		<-engineCtx.Done()
		backOff.Reset()

		err = c.engine.Stop()

		if err != nil {
			log.Errorf("failed stopping engine %v", err)
			return err
		}
		return nil
	}

	err = backoff.Retry(operation, backOff)
	if err != nil {
		log.Debugf("exiting client retry loop due to unrecoverable error: %s", err)

		return err
	}
	return nil
}

func createEngineConfig(key wgtypes.Key, config *Config) (*EngineConfig, error) {

	engineConfig := &EngineConfig{
		WgIfaceName:  config.WgIface,
		WgPort:       config.WgPort,
		WgAddr:       localWgAddr,
		WgPrivateKey: key,
	}

	port, err := freePort(config.WgPort)
	if err != nil {
		return nil, err
	}
	if port != config.WgPort {
		log.Infof("using %d as wireguard port: %d is in use", port, config.WgPort)
	}
	log.Infof("using %d as wireguard port", port)
	engineConfig.WgPort = port
	return engineConfig, nil
}

func freePort(start int) (int, error) {
	addr := net.UDPAddr{}
	if start == 0 {
		start = iface.DefaultWgPort
	}
	for x := start; x <= 65535; x++ {
		addr.Port = x
		conn, err := net.ListenUDP("udp", &addr)
		if err != nil {
			continue
		}
		log.Info("Current Listen UDP Port: ", x)
		conn.Close()
		return x, nil
	}
	return 0, errors.New("no free ports")
}
