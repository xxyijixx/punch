//go:build !android

package internal

import (
	"yiji.one/punch/client/internal/stdnet"
)

func (e *Engine) newStdNet() (*stdnet.Net, error) {
	return stdnet.NewNet(e.config.IFaceBlackList)
}
