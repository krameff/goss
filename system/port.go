package system

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/goss-org/GOnetstat"
	"github.com/goss-org/goss/util"
)

type Port interface {
	Port() string
	Exists() (bool, error)
	Listening() (bool, error)
	IP() ([]string, error)
}

type DefPort struct {
	port     string
	sysPorts map[string][]GOnetstat.Process
	err      error
}

func NewDefPort(_ context.Context, port string, system *System, config util.Config) Port {
	p := normalizePort(port)
	sysPorts, err := system.Ports()
	return &DefPort{
		port:     p,
		sysPorts: sysPorts,
		err:      err,
	}
}

func splitPort(fullport string) (network, port string) {
	split := strings.SplitN(fullport, ":", 2)
	if len(split) == 2 {
		return split[0], split[1]
	}
	return "tcp", fullport

}

func normalizePort(fullport string) string {
	net, addr := splitPort(fullport)
	return net + ":" + addr
}

func (p *DefPort) Port() string {
	return p.port
}

func (p *DefPort) Exists() (bool, error) { return p.Listening() }

func (p *DefPort) Listening() (bool, error) {
	if p.err != nil {
		return false, p.err
	}
	if _, ok := p.sysPorts[p.port]; ok {
		return true, nil
	}
	return false, nil
}

func (p *DefPort) IP() ([]string, error) {
	if p.err != nil {
		return nil, p.err
	}
	var ips []string
	for _, entry := range p.sysPorts[p.port] {
		ips = append(ips, entry.Ip)
	}
	return ips, nil
}

// GetPorts returns the set of listening TCP/UDP ports known to the OS's
// netstat backend. Failures from individual protocol lookups are joined and
// returned rather than silently discarded: GOnetstat treats a missing/
// unreadable /proc/net/{tcp,udp}{,6} file as "no ports" with no error, but it
// does return an error if a file it *could* read contains a line it can't
// parse (e.g. an unexpected IP/port encoding), which would otherwise present
// as every port simply not listening.
func GetPorts(lookupPids bool) (map[string][]GOnetstat.Process, error) {
	ports := make(map[string][]GOnetstat.Process)
	var errs []error

	netstat, err := GOnetstat.Tcp(lookupPids)
	errs = append(errs, err)
	for _, entry := range netstat {
		if entry.State == "LISTEN" {
			port := strconv.FormatInt(entry.Port, 10)
			ports["tcp:"+port] = append(ports["tcp:"+port], entry)
		}
	}

	netstat, err = GOnetstat.Tcp6(lookupPids)
	errs = append(errs, err)
	for _, entry := range netstat {
		if entry.State == "LISTEN" {
			port := strconv.FormatInt(entry.Port, 10)
			ports["tcp6:"+port] = append(ports["tcp6:"+port], entry)
		}
	}

	netstat, err = GOnetstat.Udp(lookupPids)
	errs = append(errs, err)
	for _, entry := range netstat {
		port := strconv.FormatInt(entry.Port, 10)
		ports["udp:"+port] = append(ports["udp:"+port], entry)
	}

	netstat, err = GOnetstat.Udp6(lookupPids)
	errs = append(errs, err)
	for _, entry := range netstat {
		port := strconv.FormatInt(entry.Port, 10)
		ports["udp6:"+port] = append(ports["udp6:"+port], entry)
	}

	return ports, errors.Join(errs...)
}
