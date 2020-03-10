package build

import (
	"fmt"
	"github.com/buildpacks/lifecycle/auth"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/google/go-containerregistry/pkg/authn"
)

type DefaultPhaseConfigProvider struct {
	ctrConf *dcontainer.Config
	hostConf *dcontainer.HostConfig
}

func NewDefaultPhaseConfigProvider(ops ...PhaseOperation) *DefaultPhaseConfigProvider {
	pcp := new(DefaultPhaseConfigProvider)

	for _, op := range ops {
		op(pcp)
	}

	return pcp
}

func (d *DefaultPhaseConfigProvider) ContainerConfig(name string) *dcontainer.Config {
	return nil
}

func (d *DefaultPhaseConfigProvider) HostConfig(binds []string) *dcontainer.HostConfig {
	return nil
}

func WithArgs(args ...string) PhaseOperation {
	return func(phase *DefaultPhaseConfigProvider) (*DefaultPhaseConfigProvider, error) {
		phase.ctrConf.Cmd = append(phase.ctrConf.Cmd, args...)
		return phase, nil
	}
}

func WithDaemonAccess() PhaseOperation {
	return func(phase *DefaultPhaseConfigProvider) (*DefaultPhaseConfigProvider, error) {
		phase.ctrConf.User = "root"
		phase.hostConf.Binds = append(phase.hostConf.Binds, "/var/run/docker.sock:/var/run/docker.sock")
		return phase, nil
	}
}

func WithRoot() PhaseOperation {
	return func(phase *DefaultPhaseConfigProvider) (*DefaultPhaseConfigProvider, error) {
		phase.ctrConf.User = "root"
		return phase, nil
	}
}

func WithBinds(binds ...string) PhaseOperation {
	return func(phase *DefaultPhaseConfigProvider) (*DefaultPhaseConfigProvider, error) {
		phase.hostConf.Binds = append(phase.hostConf.Binds, binds...)
		return phase, nil
	}
}

func WithRegistryAccess(repos ...string) PhaseOperation {
	return func(phase *DefaultPhaseConfigProvider) (*DefaultPhaseConfigProvider, error) {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, repos...)
		if err != nil {
			return nil, err
		}
		phase.ctrConf.Env = append(phase.ctrConf.Env, fmt.Sprintf(`CNB_REGISTRY_AUTH=%s`, authConfig))
		phase.hostConf.NetworkMode = "host"
		return phase, nil
	}
}

func WithNetwork(networkMode string) PhaseOperation {
	return func(phase *DefaultPhaseConfigProvider) (*DefaultPhaseConfigProvider, error) {
		phase.hostConf.NetworkMode = dcontainer.NetworkMode(networkMode)
		return phase, nil
	}
}