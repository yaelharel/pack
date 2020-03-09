package fakes

import "github.com/buildpacks/pack/internal/build"

type FakePhaseManager struct {
	NewCallCount      int
	ReturnForNew      build.RunnerCleaner
	NewCalledWithName string
	NewCalledWithOps  []build.PhaseOperation

	WithArgsCallCount int
	WithArgsReceived  []string

	WithNetworkCallCount int
	WithNetworkReceived  string

	WithDaemonAccessCallCount int

	WithBindsCallCount int
	WithBindsReceived  []string

	WithRegistryAccessCallCount int
	WithRegistryAccessReceived  []string

	WithRootCallCount int
}

func NewFakePhaseManager(ops ...func(*FakePhaseManager)) *FakePhaseManager {
	fakePhaseManager := &FakePhaseManager{
		ReturnForNew: &FakePhase{},
	}

	for _, op := range ops {
		op(fakePhaseManager)
	}

	return fakePhaseManager
}

func WhichReturnsForNew(phase build.RunnerCleaner) func(*FakePhaseManager) {
	return func(manager *FakePhaseManager) {
		manager.ReturnForNew = phase
	}
}

func (m *FakePhaseManager) New(name string, ops ...build.PhaseOperation) (build.RunnerCleaner, error) {
	m.NewCallCount = m.NewCallCount + 1
	m.NewCalledWithName = name
	m.NewCalledWithOps = ops

	return m.ReturnForNew, nil
}

func (m *FakePhaseManager) WithArgs(args ...string) build.PhaseOperation {
	m.WithArgsCallCount = m.WithArgsCallCount + 1
	m.WithArgsReceived = args

	return func(p *build.Phase) (phase *build.Phase, e error) {
		return nil, nil
	}
}

func (m *FakePhaseManager) WithNetwork(arg string) build.PhaseOperation {
	m.WithNetworkCallCount = m.WithNetworkCallCount + 1
	m.WithNetworkReceived = arg

	return func(p *build.Phase) (phase *build.Phase, e error) {
		return nil, nil
	}
}

func (m *FakePhaseManager) WithDaemonAccess() build.PhaseOperation {
	m.WithDaemonAccessCallCount = m.WithDaemonAccessCallCount + 1

	return func(p *build.Phase) (phase *build.Phase, e error) {
		return nil, nil
	}
}

func (m *FakePhaseManager) WithRoot() build.PhaseOperation {
	m.WithRootCallCount = m.WithRootCallCount + 1

	return func(p *build.Phase) (phase *build.Phase, e error) {
		return nil, nil
	}
}

func (m *FakePhaseManager) WithBinds(args ...string) build.PhaseOperation {
	m.WithBindsCallCount = m.WithBindsCallCount + 1
	m.WithBindsReceived = args

	return func(p *build.Phase) (phase *build.Phase, e error) {
		return nil, nil
	}
}

func (m *FakePhaseManager) WithRegistryAccess(args ...string) build.PhaseOperation {
	m.WithRegistryAccessCallCount = m.WithRegistryAccessCallCount + 1
	m.WithRegistryAccessReceived = args

	return func(p *build.Phase) (phase *build.Phase, e error) {
		return nil, nil
	}
}
