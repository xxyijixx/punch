module yiji.one/punch

go 1.21.0

require (
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/google/uuid v1.6.0
	github.com/pion/ice/v3 v3.0.2
	github.com/sirupsen/logrus v1.9.3
	github.com/vishvananda/netlink v1.2.1-beta.2
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/sys v0.21.0
	golang.zx2c4.com/wireguard v0.0.0-20230704135630-469159ecf7d1
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20230429144221-925a1e7659e6
	golang.zx2c4.com/wireguard/windows v0.5.3
	google.golang.org/grpc v1.64.1
	google.golang.org/protobuf v1.34.1 // indirect
)

require (
	github.com/cilium/ebpf v0.15.0
	github.com/golang/mock v1.6.0
	github.com/google/gopacket v1.1.19
	github.com/hashicorp/go-multierror v1.1.1
	github.com/libp2p/go-netroute v0.2.1
	github.com/mdlayher/socket v0.4.1
	github.com/pion/logging v0.2.2
	github.com/pion/stun/v2 v2.0.0
	github.com/pion/transport/v3 v3.0.1
	github.com/stretchr/testify v1.9.0
	github.com/things-go/go-socks5 v0.0.4
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842
	golang.org/x/net v0.26.0
	golang.org/x/sync v0.7.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/josharian/native v1.1.0 // indirect
	github.com/mdlayher/genetlink v1.3.2 // indirect
	github.com/mdlayher/netlink v1.7.2 // indirect
	github.com/pion/dtls/v2 v2.2.10 // indirect
	github.com/pion/mdns v0.0.12 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/transport/v2 v2.2.4 // indirect
	github.com/pion/turn/v3 v3.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240515191416-fc5f0ca64291 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gvisor.dev/gvisor v0.0.0-20230927004350-cbd86285d259 // indirect
)

replace github.com/kardianos/service => github.com/netbirdio/service v0.0.0-20230215170314-b923b89432b0

replace github.com/getlantern/systray => github.com/netbirdio/systray v0.0.0-20231030152038-ef1ed2a27949

replace golang.zx2c4.com/wireguard => github.com/netbirdio/wireguard-go v0.0.0-20240105182236-6c340dd55aed

replace github.com/cloudflare/circl => github.com/cunicu/circl v0.0.0-20230801113412-fec58fc7b5f6

replace github.com/pion/ice/v3 => github.com/netbirdio/ice/v3 v3.0.0-20240315174635-e72a50fcb64e

replace github.com/libp2p/go-netroute => github.com/netbirdio/go-netroute v0.0.0-20240611143515-f59b0e1d3944
