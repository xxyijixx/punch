//go:build !android

package peer

import (
	"yiji.one/punch/client/internal/stdnet"
)

func (conn *Conn) newStdNet() (*stdnet.Net, error) {
	return stdnet.NewNet(conn.config.InterfaceBlackList)
}
