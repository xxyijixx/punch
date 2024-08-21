package internal

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"yiji.one/punch/iface"
	"yiji.one/punch/util"
)

var defaultInterfaceBlacklist = []string{
	iface.WgInterfaceDefault, "wt", "utun", "tun0", "zt", "ZeroTier", "wg", "ts",
	"Tailscale", "tailscale", "docker", "veth", "br-", "lo",
}

// ConfigInput carries configuration changes to the client
// ConfigInput 结构体定义了一个配置输入的结构
type ConfigInput struct {
	// ManagementURL 管理URL
	ManagementURL string
	// AdminURL 管理员URL
	AdminURL string
	// ConfigPath 配置路径
	ConfigPath string
	// NATExternalIPs NAT外部IP
	NATExternalIPs []string
	// CustomDNSAddress 自定义DNS地址
	CustomDNSAddress []byte
	// InterfaceName 接口名称
	InterfaceName *string
	// WireguardPort Wireguard端口
	WireguardPort *int
	// ExtraIFaceBlackList 额外接口黑名单
	ExtraIFaceBlackList []string
}

// SimpleConfig
type SimpleConfig struct {
	PrivateKey string
}

// Config Configuration type
type Config struct {
	// Wireguard private key of local peer
	PrivateKey     string
	PreSharedKey   string
	WgIface        string
	WgPort         int
	NetworkMonitor *bool
	IFaceBlackList []string
	// SSHKey is a private SSH key in a PEM format
	SSHKey string

	// ExternalIP mappings, if different from the host interface IP
	//
	//   External IP must not be behind a CGNAT and port-forwarding for incoming UDP packets from WgPort on ExternalIP
	//   to WgPort on host interface IP must be present. This can take form of single port-forwarding rule, 1:1 DNAT
	//   mapping ExternalIP to host interface IP, or a NAT DMZ to host interface IP.
	//
	//   A single mapping will take the form of: external[/internal]
	//    external (required): either the external IP address or "stun" to use STUN to determine the external IP address
	//    internal (optional): either the internal/interface IP address or an interface name
	//
	//   examples:
	//      "12.34.56.78"          => all interfaces IPs will be mapped to external IP of 12.34.56.78
	//      "12.34.56.78/eth0"     => IPv4 assigned to interface eth0 will be mapped to external IP of 12.34.56.78
	//      "12.34.56.78/10.1.2.3" => interface IP 10.1.2.3 will be mapped to external IP of 12.34.56.78

	NATExternalIPs []string
	// CustomDNSAddress sets the DNS resolver listening address in format ip:port
	CustomDNSAddress string

	// DisableAutoConnect determines whether the client should not start with the service
	// it's set to false by default due to backwards compatibility
	DisableAutoConnect bool

	// DNSRouteInterval is the interval in which the DNS routes are updated
	DNSRouteInterval time.Duration
}

// ReadConfig read config file and return with Config. If it is not exists create a new with default values
func ReadConfig(configPath string) (*Config, error) {
	// 判断配置文件是否存在
	if configFileIsExists(configPath) {
		config := &Config{}
		// 读取配置文件
		if _, err := util.ReadJson(configPath, config); err != nil {
			return nil, err
		}
		// 初始化配置，不进行任何更改
		if changed, err := config.apply(ConfigInput{}); err != nil {
			return nil, err
		} else if changed {
			// 如果配置有更改，则写入配置文件
			if err = WriteOutConfig(configPath, config); err != nil {
				return nil, err
			}
		}

		return config, nil
	}

	// 如果配置文件不存在，则创建一个新的配置文件
	cfg, err := createNewConfig(ConfigInput{ConfigPath: configPath})
	if err != nil {
		return nil, err
	}

	// 将新的配置文件写入文件
	err = WriteOutConfig(configPath, cfg)
	return cfg, err
}

// UpdateConfig update existing configuration according to input configuration and return with the configuration
func UpdateConfig(input ConfigInput) (*Config, error) {
	if !configFileIsExists(input.ConfigPath) {
		return nil, status.Errorf(codes.NotFound, "config file doesn't exist")
	}

	return update(input)
}

// UpdateOrCreateConfig reads existing config or generates a new one
func UpdateOrCreateConfig(input ConfigInput) (*Config, error) {
	if !configFileIsExists(input.ConfigPath) {
		log.Infof("generating new config %s", input.ConfigPath)
		cfg, err := createNewConfig(input)
		if err != nil {
			return nil, err
		}
		err = WriteOutConfig(input.ConfigPath, cfg)
		return cfg, err
	}

	return update(input)
}

// CreateInMemoryConfig generate a new config but do not write out it to the store
func CreateInMemoryConfig(input ConfigInput) (*Config, error) {
	return createNewConfig(input)
}

// WriteOutConfig write put the prepared config to the given path
func WriteOutConfig(path string, config *Config) error {
	return util.WriteJson(path, config)
}

// createNewConfig creates a new config generating a new Wireguard key and saving to file
func createNewConfig(input ConfigInput) (*Config, error) {
	config := &Config{
		// defaults to false only for new (post 0.26) configurations
	}
	if configFileIsExists(input.ConfigPath) {
		simpleConfig := &SimpleConfig{}
		if _, err := util.ReadJson(input.ConfigPath, simpleConfig); err != nil {
			return nil, err
		}
		config.PrivateKey = simpleConfig.PrivateKey

		if _, err := config.apply(input); err != nil {
			return nil, err
		}

		return config, nil
	}

	// 如果配置文件不存在，则创建一个新的配置
	if _, err := config.apply(input); err != nil {
		return nil, err
	}
	scfg := &SimpleConfig{
		PrivateKey: config.PrivateKey,
	}
	// 将新的配置文件写入文件
	err := util.WriteJson(input.ConfigPath, scfg)

	return config, err
}

func update(input ConfigInput) (*Config, error) {
	config := &Config{}

	if _, err := util.ReadJson(input.ConfigPath, config); err != nil {
		return nil, err
	}

	updated, err := config.apply(input)
	if err != nil {
		return nil, err
	}

	if updated {
		if err := util.WriteJson(input.ConfigPath, config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func (config *Config) apply(input ConfigInput) (updated bool, err error) {
	// WACXtBaxOtoObaoxScBst2OA/OjDS4XERXv4SlXqxUQ=
	if config.PrivateKey == "" {
		log.Infof("generated new Wireguard key")
		config.PrivateKey = generateKey()
		updated = true
	}
	pubKey, _ := wgtypes.ParseKey(config.PrivateKey)

	log.Infof("PrivateKey: [%s], PublicKey: [%s]", config.PrivateKey, pubKey.PublicKey().String())
	if input.WireguardPort != nil && *input.WireguardPort != config.WgPort {
		log.Infof("updating Wireguard port %d (old value %d)",
			*input.WireguardPort, config.WgPort)
		config.WgPort = *input.WireguardPort
		updated = true
	} else if config.WgPort == 0 {
		config.WgPort = iface.DefaultWgPort
		log.Infof("using default Wireguard port %d", config.WgPort)
		updated = true
	}

	if input.InterfaceName != nil && *input.InterfaceName != config.WgIface {
		log.Infof("updating Wireguard interface %#v (old value %#v)",
			*input.InterfaceName, config.WgIface)
		config.WgIface = *input.InterfaceName
		updated = true
	} else if config.WgIface == "" {
		config.WgIface = iface.WgInterfaceDefault
		log.Infof("using default Wireguard interface %s", config.WgIface)
		updated = true
	}

	if len(config.IFaceBlackList) == 0 {
		log.Infof("filling in interface blacklist with defaults: [ %s ]",
			strings.Join(defaultInterfaceBlacklist, " "))
		config.IFaceBlackList = append(config.IFaceBlackList, defaultInterfaceBlacklist...)
		updated = true
	}

	if len(input.ExtraIFaceBlackList) > 0 {
		for _, iFace := range util.SliceDiff(input.ExtraIFaceBlackList, config.IFaceBlackList) {
			log.Infof("adding new entry to interface blacklist: %s", iFace)
			config.IFaceBlackList = append(config.IFaceBlackList, iFace)
			updated = true
		}
	}

	return updated, nil
}

// parseURL parses and validates a service URL
func parseURL(serviceName, serviceURL string) (*url.URL, error) {
	parsedMgmtURL, err := url.ParseRequestURI(serviceURL)
	if err != nil {
		log.Errorf("failed parsing %s URL %s: [%s]", serviceName, serviceURL, err.Error())
		return nil, err
	}

	if parsedMgmtURL.Scheme != "https" && parsedMgmtURL.Scheme != "http" {
		return nil, fmt.Errorf(
			"invalid %s URL provided %s. Supported format [http|https]://[host]:[port]",
			serviceName, serviceURL)
	}

	if parsedMgmtURL.Port() == "" {
		switch parsedMgmtURL.Scheme {
		case "https":
			parsedMgmtURL.Host += ":443"
		case "http":
			parsedMgmtURL.Host += ":80"
		default:
			log.Infof("unable to determine a default port for schema %s in URL %s", parsedMgmtURL.Scheme, serviceURL)
		}
	}

	return parsedMgmtURL, err
}

// generateKey generates a new Wireguard private key
func generateKey() string {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		panic(err)
	}
	return key.String()
}

func configFileIsExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
