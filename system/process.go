package system

import (
	"context"

	"github.com/shirou/gopsutil/v4/process"

	"github.com/krameff/goss/util"
)

type Process interface {
	Executable() string
	Exists() (bool, error)
	Running() (bool, error)
	Pids() ([]int, error)
}

type DefProcess struct {
	executable string
	procMap    map[string][]*process.Process
	err        error
}

func NewDefProcess(_ context.Context, executable string, system *System, config util.Config) Process {
	pmap, err := system.ProcMap()
	return &DefProcess{
		executable: executable,
		procMap:    pmap,
		err:        err,
	}
}

func (p *DefProcess) Executable() string {
	return p.executable
}

func (p *DefProcess) Exists() (bool, error) { return p.Running() }

func (p *DefProcess) Pids() ([]int, error) {
	var pids []int
	if p.err != nil {
		return pids, p.err
	}
	for _, proc := range p.procMap[p.executable] {
		pids = append(pids, int(proc.Pid))
	}
	return pids, nil
}

func (p *DefProcess) Running() (bool, error) {
	if p.err != nil {
		return false, p.err
	}
	if _, ok := p.procMap[p.executable]; ok {
		return true, nil
	}
	return false, nil
}

func GetProcs() (map[string][]*process.Process, error) {
	pmap := make(map[string][]*process.Process)
	processes, err := process.Processes()
	if err != nil {
		return pmap, err
	}
	for _, p := range processes {
		// A process can legitimately disappear between listing and reading
		// its name (it exited), or its /proc entry may be transiently
		// unreadable. Skip it rather than failing the whole snapshot,
		// mirroring go-ps's behavior of silently ignoring per-process
		// read errors.
		name, err := p.Name()
		if err != nil {
			continue
		}
		pmap[name] = append(pmap[name], p)
	}

	return pmap, nil
}
