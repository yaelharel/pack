package build

import (
	"fmt"
	"github.com/buildpacks/lifecycle/auth"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"
)

type ConcretePhaseManager struct {
	lifecycle *Lifecycle
}

type PhaseOperation func(*Phase) (*Phase, error)

func NewConcretePhaseManager(lifecycle *Lifecycle) *ConcretePhaseManager {
	return &ConcretePhaseManager{lifecycle: lifecycle}
}

func (m *ConcretePhaseManager) New(name string, ops ...PhaseOperation) (RunnerCleaner, error) {
	ctrConf := &dcontainer.Config{
		Image:  m.lifecycle.builder.Name(),
		Labels: map[string]string{"author": "pack"},
	}
	hostConf := &dcontainer.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", m.lifecycle.LayersVolume, layersDir),
			fmt.Sprintf("%s:%s", m.lifecycle.AppVolume, appDir),
		},
	}
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

	if m.lifecycle.httpProxy != "" {
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

	var err error
	for _, op := range ops {
		phase, err = op(phase)
		if err != nil {
			return nil, errors.Wrapf(err, "create %s phase", name)
		}
	}
	return phase, nil
}

func (*ConcretePhaseManager) WithArgs(args ...string) PhaseOperation {
	return func(phase *Phase) (*Phase, error) {
		phase.ctrConf.Cmd = append(phase.ctrConf.Cmd, args...)
		return phase, nil
	}
}

func (*ConcretePhaseManager) WithDaemonAccess() PhaseOperation {
	return func(phase *Phase) (*Phase, error) {
		phase.ctrConf.User = "root"
		phase.hostConf.Binds = append(phase.hostConf.Binds, "/var/run/docker.sock:/var/run/docker.sock")
		return phase, nil
	}
}

func (*ConcretePhaseManager) WithRoot() PhaseOperation {
	return func(phase *Phase) (*Phase, error) {
		phase.ctrConf.User = "root"
		return phase, nil
	}
}

func (*ConcretePhaseManager) WithBinds(binds ...string) PhaseOperation {
	return func(phase *Phase) (*Phase, error) {
		phase.hostConf.Binds = append(phase.hostConf.Binds, binds...)
		return phase, nil
	}
}

func (*ConcretePhaseManager) WithRegistryAccess(repos ...string) PhaseOperation {
	return func(phase *Phase) (*Phase, error) {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, repos...)
		if err != nil {
			return nil, err
		}
		phase.ctrConf.Env = append(phase.ctrConf.Env, fmt.Sprintf(`CNB_REGISTRY_AUTH=%s`, authConfig))
		phase.hostConf.NetworkMode = "host"
		return phase, nil
	}
}

func (*ConcretePhaseManager) WithNetwork(networkMode string) PhaseOperation {
	return func(phase *Phase) (*Phase, error) {
		phase.hostConf.NetworkMode = dcontainer.NetworkMode(networkMode)
		return phase, nil
	}
}