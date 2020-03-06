package build_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/build"
	ilogging "github.com/buildpacks/pack/internal/logging"
	"github.com/buildpacks/pack/internal/stringset"
	h "github.com/buildpacks/pack/testhelpers"
)

type FakePhase struct {
	CleanupCallCount int
	RunCallCount     int
}

func (p *FakePhase) Cleanup() error {
	p.CleanupCallCount = p.CleanupCallCount + 1

	return nil
}

func (p *FakePhase) Run(ctx context.Context) error {
	p.RunCallCount = p.RunCallCount + 1

	return nil
}

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

func TestPhases(t *testing.T) {
	// TODO: shared with other test file; fix CreateFakeLifecycle
	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
	h.AssertNil(t, err)

	repoName = "phase.test.lc-" + h.RandString(10) // TODO: reponame is globally referenced in CreateFakeLifecycle

	wd, err := os.Getwd()
	h.AssertNil(t, err)

	h.CreateImageFromDir(t, dockerCli, repoName, filepath.Join(wd, "testdata", "fake-lifecycle"))
	defer h.DockerRmi(dockerCli, repoName)

	spec.Run(t, "phases", testPhases, spec.Report(report.Terminal{}), spec.Sequential())
}

func testPhases(t *testing.T, when spec.G, it spec.S) {
	when("#Detect", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t)
			fakePhase := &FakePhase{}
			fakePhaseManager := fakePhaseManager(whichReturnsForNew(fakePhase))

			err := lifecycle.Detect(context.Background(), "test", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakePhaseManager()

			err := lifecycle.Detect(context.Background(), "test", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.NewCalledWithName, "detector")
			h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
			assertIncludeAllExpectedArgPatterns(t,
				fakePhaseManager.WithArgsReceived,
				//[]string{"-log-level", "debug"}, // TODO: test verbose logging
				[]string{"-app", "/workspace"},
				[]string{"-platform", "/platform"},
			)
		})

		it("configures the phase with the expected network mode", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakePhaseManager()
			expectedNetworkMode := "some-network-mode"

			err := lifecycle.Detect(context.Background(), expectedNetworkMode, fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.WithNetworkCallCount, 1)
			h.AssertEq(t, fakePhaseManager.WithNetworkReceived, expectedNetworkMode)
		})
	})

	when("Restore", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t)
			fakePhase := &FakePhase{}
			fakePhaseManager := fakePhaseManager(whichReturnsForNew(fakePhase))

			err := lifecycle.Restore(context.Background(), "test", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with daemon access", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakePhaseManager()

			err := lifecycle.Restore(context.Background(), "test", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.WithDaemonAccessCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakePhaseManager()

			err := lifecycle.Restore(context.Background(), "test", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.NewCalledWithName, "restorer")
			h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
			assertIncludeAllExpectedArgPatterns(t,
				fakePhaseManager.WithArgsReceived,
				[]string{"-cache-dir", "/cache"},
				[]string{"-layers", "/layers"},
			)
		})

		it("configures the phase with binds", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakePhaseManager()
			expectedBinds := "some-cache-name:/cache"

			err := lifecycle.Restore(context.Background(), "some-cache-name", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.WithBindsCallCount, 1)
			h.AssertEq(t, fakePhaseManager.WithBindsReceived[0], expectedBinds)
		})
	})

	when("Analyze", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t)
			fakePhase := &FakePhase{}
			fakePhaseManager := fakePhaseManager(whichReturnsForNew(fakePhase))

			err := lifecycle.Analyze(context.Background(), "test", "test", false, false, fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		when("clear cache", func() {
			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakePhaseManager()
				expectedRepoName := "some-repo-name"

				err := lifecycle.Analyze(context.Background(), expectedRepoName, "test", false, true, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.NewCalledWithName, "analyzer")
				h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
				assertIncludeAllExpectedArgPatterns(t,
					fakePhaseManager.WithArgsReceived,
					[]string{"-skip-layers"},
				)
			})
		})

		when("clear cache is false", func() {
			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakePhaseManager()
				expectedRepoName := "some-repo-name"

				err := lifecycle.Analyze(context.Background(), expectedRepoName, "test", false, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.NewCalledWithName, "analyzer")
				h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
				assertIncludeAllExpectedArgPatterns(t,
					fakePhaseManager.WithArgsReceived,
					[]string{"-cache-dir", "/cache"},
				)
			})
		})

		when("publish", func() {
			it("configures the phase with registry access", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakePhaseManager()
				expectedRepoName := "some-repo-name"

				err := lifecycle.Analyze(context.Background(), expectedRepoName, "test", true, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithRegistryAccessCallCount, 1)
				h.AssertEq(t, fakePhaseManager.WithRegistryAccessReceived[0], expectedRepoName)
			})

			it("configures the phase with root", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakePhaseManager()

				err := lifecycle.Analyze(context.Background(), "test", "test", true, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithRootCallCount, 1)
			})

			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakePhaseManager()
				expectedRepoName := "some-repo-name"

				err := lifecycle.Analyze(context.Background(), expectedRepoName, "test", true, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.NewCalledWithName, "analyzer")
				h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
				assertIncludeAllExpectedArgPatterns(t,
					fakePhaseManager.WithArgsReceived,
					[]string{"-layers", "/layers"},
					[]string{expectedRepoName},
				)
			})

			it("configures the phase with binds", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakePhaseManager()
				expectedBinds := "some-cache-name:/cache"

				err := lifecycle.Analyze(context.Background(), "test", "some-cache-name", true, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithBindsCallCount, 1)
				h.AssertEq(t, fakePhaseManager.WithBindsReceived[0], expectedBinds)
			})
		})

		when("publish is false", func() {
			it("configures the phase with daemon access", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakePhaseManager()

				err := lifecycle.Analyze(context.Background(), "test", "test", false, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithDaemonAccessCallCount, 1)
			})

			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakePhaseManager()
				expectedRepoName := "some-repo-name"

				err := lifecycle.Analyze(context.Background(), expectedRepoName, "test", false, true, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.NewCalledWithName, "analyzer")
				h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
				assertIncludeAllExpectedArgPatterns(t,
					fakePhaseManager.WithArgsReceived,
					[]string{"-daemon"},
					[]string{"-layers", "/layers"},
					[]string{expectedRepoName},
				)
			})

			it("configures the phase with binds", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakePhaseManager()
				expectedBinds := "some-cache-name:/cache"

				err := lifecycle.Analyze(context.Background(), "test", "some-cache-name", false, true, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithBindsCallCount, 1)
				h.AssertEq(t, fakePhaseManager.WithBindsReceived[0], expectedBinds)
			})
		})
	})
}

func fakeLifecycle(t *testing.T) *build.Lifecycle {
	var outBuf bytes.Buffer
	logger := ilogging.NewLogWithWriters(&outBuf, &outBuf)

	// TODO: see if we can use a fake docker client
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
	h.AssertNil(t, err)

	lifecycle, err := CreateFakeLifecycle(filepath.Join("testdata", "fake-app"), docker, logger)
	h.AssertNil(t, err)

	return lifecycle
}

func fakePhaseManager(ops ...func(*FakePhaseManager)) *FakePhaseManager {
	fakePhaseManager := &FakePhaseManager{
		ReturnForNew: &FakePhase{},
	}

	for _, op := range ops {
		op(fakePhaseManager)
	}

	return fakePhaseManager
}

func whichReturnsForNew(phase build.RunnerCleaner) func(*FakePhaseManager) {
	return func(manager *FakePhaseManager) {
		manager.ReturnForNew = phase
	}
}

func assertIncludeAllExpectedArgPatterns(t *testing.T, receivedArgs []string, expectedPatterns ...[]string) {
	missingPatterns := [][]string{}

	for _, expectedPattern := range expectedPatterns {
		if !patternExists(expectedPattern, receivedArgs) {
			missingPatterns = append(missingPatterns, expectedPattern)
		}
	}

	assertSliceEmpty(t,
		missingPatterns,
		"Expected the patterns %s to exist in [%s]",
		missingPatterns,
		strings.Join(receivedArgs, " "),
	)
}

func patternExists(expectedPattern []string, receivedArgs []string) bool {
	_, missing, _ := stringset.Compare(receivedArgs, expectedPattern)
	if len(missing) > 0 {
		return false
	}

	if len(expectedPattern) == 1 {
		return true
	}

	for _, loc := range matchLocations(expectedPattern[0], receivedArgs) {
		finalElementLoc := loc + len(expectedPattern)

		receivedSubSlice := receivedArgs[loc:finalElementLoc]

		if reflect.DeepEqual(receivedSubSlice, expectedPattern) {
			return true
		}
	}

	return false
}

func matchLocations(expectedArg string, receivedArgs []string) []int {
	indices := []int{}

	for i, receivedArg := range receivedArgs {
		if receivedArg == expectedArg {
			indices = append(indices, i)
		}
	}

	return indices
}

func assertSliceEmpty(t *testing.T, actual interface{}, msg string, msgArgs ...interface{}) {
	empty, err := sliceEmpty(actual)

	if err != nil {
		t.Fatalf("assertSliceNotEmpty error: %s", err.Error())
	}

	if !empty {
		t.Fatalf(msg, msgArgs...)
	}
}

func sliceEmpty(slice interface{}) (bool, error) {
	switch reflect.TypeOf(slice).Kind() {
	case reflect.Slice:
		return reflect.ValueOf(slice).Len() == 0, nil
	default:
		return true, fmt.Errorf("invoked with non slice actual: %v", slice)
	}
}
