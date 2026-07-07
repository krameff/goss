//go:build windows
// +build windows

package system

import "errors"

var errNotImplemented = errors.New("not implemented")

func getUsage(mountpoint string) (int, error) {
	return 0, errNotImplemented
}
