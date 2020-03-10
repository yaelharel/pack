package build

import (
	"fmt"

	dcontainer "github.com/docker/docker/api/types/container"
)

type PhaseConfigProvider interface {
	ContainerConfig(string) *dcontainer.Config // does this need to be aliased?
	HostConfig([]string) *dcontainer.HostConfig
}

type ConcretePhaseManager struct {
	lifecycle *Lifecycle
}

type PhaseOperation func(*DefaultPhaseConfigProvider) (*DefaultPhaseConfigProvider, error)

func NewConcretePhaseManager(lifecycle *Lifecycle) *ConcretePhaseManager {
	return &ConcretePhaseManager{lifecycle: lifecycle}
}

func (m *ConcretePhaseManager) New(name string, pcp PhaseConfigProvider) (RunnerCleaner, error) {
	ctrConf := pcp.ContainerConfig(m.lifecycle.builder.Name())
	hostConf := pcp.HostConfig([]string{
		fmt.Sprintf("%s:%s", m.lifecycle.LayersVolume, layersDir),
		fmt.Sprintf("%s:%s", m.lifecycle.AppVolume, appDir),
	})
	ctrConf.Cmd = []string{"/cnb/lifecycle/" + name}
	phase := &Phase{
		ctrConf:  ctrConf,
		hostConf: hostConf,
		name:     name,
		docker:   m.lifecycle.docker,
		logger:   m.lifecycle.logger,
		uid:      m.lifecycle.builder.UID,
		gid:      m.lifecycle.builder.GID,
		appPath:  m.lifecycle.appPath,
		appOnce:  m.lifecycle.appOnce,
	}

	if m.lifecycle.httpProxy != "" { // consider also passing this to the config provider
		phase.ctrConf.Env = append(phase.ctrConf.Env, "HTTP_PROXY="+m.lifecycle.httpProxy)
		phase.ctrConf.Env = append(phase.ctrConf.Env, "http_proxy="+m.lifecycle.httpProxy)
	}
	if m.lifecycle.httpsProxy != "" {
		phase.ctrConf.Env = append(phase.ctrConf.Env, "HTTPS_PROXY="+m.lifecycle.httpsProxy)
		phase.ctrConf.Env = append(phase.ctrConf.Env, "https_proxy="+m.lifecycle.httpsProxy)
	}
	if m.lifecycle.noProxy != "" {
		phase.ctrConf.Env = append(phase.ctrConf.Env, "NO_PROXY="+m.lifecycle.noProxy)
		phase.ctrConf.Env = append(phase.ctrConf.Env, "no_proxy="+m.lifecycle.noProxy)
	}

	//var err error
	//for _, op := range ops {
	//	phase, err = op(phase)
	//	if err != nil {
	//		return nil, errors.Wrapf(err, "create %s phase", name)
	//	}
	//}
	return phase, nil
}


