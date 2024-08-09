//go:build !android
// +build !android

package iface

import (
	"yiji.one/punch/iface/bind"
)

type wgTunDevice interface {
	Create() (wgConfigurer, error)
	Up() (*bind.UniversalUDPMuxDefault, error)
	UpdateAddr(address WGAddress) error
	WgAddress() WGAddress
	DeviceName() string
	Close() error
	Wrapper() *DeviceWrapper // todo eliminate this function
}
