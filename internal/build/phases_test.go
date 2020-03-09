package build_test

import (
	"bytes"
	"context"
	"github.com/buildpacks/pack/internal/build/fakes"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/client"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/build"
	ilogging "github.com/buildpacks/pack/internal/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

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
			fakePhase := &fakes.FakePhase{}
			fakePhaseManager := fakes.NewFakePhaseManager(fakes.WhichReturnsForNew(fakePhase))

			err := lifecycle.Detect(context.Background(), "test", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakes.NewFakePhaseManager()

			err := lifecycle.Detect(context.Background(), "test", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.NewCalledWithName, "detector")
			h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
			h.AssertIncludeAllExpectedArgPatterns(t,
				fakePhaseManager.WithArgsReceived,
				//[]string{"-log-level", "debug"}, // TODO: test verbose logging
				[]string{"-app", "/workspace"},
				[]string{"-platform", "/platform"},
			)
		})

		it("configures the phase with the expected network mode", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakes.NewFakePhaseManager()
			expectedNetworkMode := "some-network-mode"

			err := lifecycle.Detect(context.Background(), expectedNetworkMode, fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.WithNetworkCallCount, 1)
			h.AssertEq(t, fakePhaseManager.WithNetworkReceived, expectedNetworkMode)
		})
	})

	when("#Restore", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t)
			fakePhase := &fakes.FakePhase{}
			fakePhaseManager := fakes.NewFakePhaseManager(fakes.WhichReturnsForNew(fakePhase))

			err := lifecycle.Restore(context.Background(), "test", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with daemon access", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakes.NewFakePhaseManager()

			err := lifecycle.Restore(context.Background(), "test", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.WithDaemonAccessCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakes.NewFakePhaseManager()

			err := lifecycle.Restore(context.Background(), "test", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.NewCalledWithName, "restorer")
			h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
			h.AssertIncludeAllExpectedArgPatterns(t,
				fakePhaseManager.WithArgsReceived,
				[]string{"-cache-dir", "/cache"},
				[]string{"-layers", "/layers"},
			)
		})

		it("configures the phase with binds", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakes.NewFakePhaseManager()
			expectedBinds := []string{"some-cache:/cache"}

			err := lifecycle.Restore(context.Background(), "some-cache", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.WithBindsCallCount, 1)
			h.AssertEq(t, fakePhaseManager.WithBindsReceived, expectedBinds)
		})
	})

	when("#Analyze", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t)
			fakePhase := &fakes.FakePhase{}
			fakePhaseManager := fakes.NewFakePhaseManager(fakes.WhichReturnsForNew(fakePhase))

			err := lifecycle.Analyze(context.Background(), "test", "test", false, false, fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		when("clear cache", func() {
			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedRepoName := "some-repo-name"

				err := lifecycle.Analyze(context.Background(), expectedRepoName, "test", false, true, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.NewCalledWithName, "analyzer")
				h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
				h.AssertIncludeAllExpectedArgPatterns(t,
					fakePhaseManager.WithArgsReceived,
					[]string{"-skip-layers"},
				)
			})
		})

		when("clear cache is false", func() {
			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedRepoName := "some-repo-name"

				err := lifecycle.Analyze(context.Background(), expectedRepoName, "test", false, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.NewCalledWithName, "analyzer")
				h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
				h.AssertIncludeAllExpectedArgPatterns(t,
					fakePhaseManager.WithArgsReceived,
					[]string{"-cache-dir", "/cache"},
				)
			})
		})

		when("publish", func() {
			it("configures the phase with registry access", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedRepos := []string{"some-repo-name"}

				err := lifecycle.Analyze(context.Background(), expectedRepos[0], "test", true, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithRegistryAccessCallCount, 1)
				h.AssertEq(t, fakePhaseManager.WithRegistryAccessReceived, expectedRepos)
			})

			it("configures the phase with root", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()

				err := lifecycle.Analyze(context.Background(), "test", "test", true, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithRootCallCount, 1)
			})

			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedRepoName := "some-repo-name"

				err := lifecycle.Analyze(context.Background(), expectedRepoName, "test", true, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.NewCalledWithName, "analyzer")
				h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
				h.AssertIncludeAllExpectedArgPatterns(t,
					fakePhaseManager.WithArgsReceived,
					[]string{"-layers", "/layers"},
					[]string{expectedRepoName},
				)
			})

			it("configures the phase with binds", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedBinds := []string{"some-cache:/cache"}

				err := lifecycle.Analyze(context.Background(), "test", "some-cache", true, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithBindsCallCount, 1)
				h.AssertEq(t, fakePhaseManager.WithBindsReceived, expectedBinds)
			})
		})

		when("publish is false", func() {
			it("configures the phase with daemon access", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()

				err := lifecycle.Analyze(context.Background(), "test", "test", false, false, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithDaemonAccessCallCount, 1)
			})

			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedRepoName := "some-repo-name"

				err := lifecycle.Analyze(context.Background(), expectedRepoName, "test", false, true, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.NewCalledWithName, "analyzer")
				h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
				h.AssertIncludeAllExpectedArgPatterns(t,
					fakePhaseManager.WithArgsReceived,
					[]string{"-daemon"},
					[]string{"-layers", "/layers"},
					[]string{expectedRepoName},
				)
			})

			it("configures the phase with binds", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedBinds := []string{"some-cache:/cache"}

				err := lifecycle.Analyze(context.Background(), "test", "some-cache", false, true, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithBindsCallCount, 1)
				h.AssertEq(t, fakePhaseManager.WithBindsReceived, expectedBinds)
			})
		})
	})

	when("#Build", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t)
			fakePhase := &fakes.FakePhase{}
			fakePhaseManager := fakes.NewFakePhaseManager(fakes.WhichReturnsForNew(fakePhase))

			err := lifecycle.Build(context.Background(), "test", []string{}, fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakes.NewFakePhaseManager()

			err := lifecycle.Build(context.Background(), "test", []string{}, fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.NewCalledWithName, "builder")
			h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
			h.AssertIncludeAllExpectedArgPatterns(t,
				fakePhaseManager.WithArgsReceived,
				[]string{"-layers", "/layers"},
				[]string{"-app", "/workspace"},
				[]string{"-platform", "/platform"},
			)
		})

		it("configures the phase with the expected network mode", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakes.NewFakePhaseManager()
			expectedNetworkMode := "some-network-mode"

			err := lifecycle.Build(context.Background(), expectedNetworkMode, []string{}, fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.WithNetworkCallCount, 1)
			h.AssertEq(t, fakePhaseManager.WithNetworkReceived, expectedNetworkMode)
		})

		it("configures the phase with binds", func() {
			lifecycle := fakeLifecycle(t)
			fakePhaseManager := fakes.NewFakePhaseManager()
			expectedBinds := []string{"some-volume"}

			err := lifecycle.Build(context.Background(), "test", expectedBinds, fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseManager.WithBindsCallCount, 1)
			h.AssertEq(t, fakePhaseManager.WithBindsReceived, expectedBinds)
		})
	})

	when("#Export", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t)
			fakePhase := &fakes.FakePhase{}
			fakePhaseManager := fakes.NewFakePhaseManager(fakes.WhichReturnsForNew(fakePhase))

			err := lifecycle.Export(context.Background(), "test", "test", false, "test", "test", fakePhaseManager)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		when("publish", func() {
			it("configures the phase with registry access", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedRepos := []string{"some-repo-name", "some-run-image"}

				err := lifecycle.Export(context.Background(), expectedRepos[0], expectedRepos[1], true, "test", "test", fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithRegistryAccessCallCount, 1)
				h.AssertEq(t, fakePhaseManager.WithRegistryAccessReceived, expectedRepos)
			})

			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedRepoName := "some-repo-name"
				expectedRunImage := "some-run-image"
				expectedLaunchCacheName := "some-launch-cache"
				expectedCacheName := "some-cache"

				err := lifecycle.Export(context.Background(), expectedRepoName, expectedRunImage, true, expectedLaunchCacheName, expectedCacheName, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.NewCalledWithName, "exporter")
				h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
				h.AssertIncludeAllExpectedArgPatterns(t,
					fakePhaseManager.WithArgsReceived,
					[]string{"-image", expectedRunImage},
					[]string{"-cache-dir", "/cache"},
					[]string{"-layers", "/layers"},
					[]string{"-app", "/workspace"},
					[]string{expectedRepoName},
				)
			})

			it("configures the phase with root", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()

				err := lifecycle.Export(context.Background(), "test", "test", true, "test", "test", fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithRootCallCount, 1)
			})

			it("configures the phase with binds", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedBinds := []string{"some-cache:/cache"}

				err := lifecycle.Export(context.Background(), "test", "test", true, "test", "some-cache", fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithBindsCallCount, 1)
				h.AssertEq(t, fakePhaseManager.WithBindsReceived, expectedBinds)
			})
		})

		when("publish is false", func() {
			it("configures the phase with daemon access", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()

				err := lifecycle.Export(context.Background(), "test", "test", false, "test", "test", fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithDaemonAccessCallCount, 1)
			})

			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedRepoName := "some-repo-name"
				expectedRunImage := "some-run-image"
				expectedLaunchCacheName := "some-launch-cache"
				expectedCacheName := "some-cache"

				err := lifecycle.Export(context.Background(), expectedRepoName, expectedRunImage, false, expectedLaunchCacheName, expectedCacheName, fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.NewCalledWithName, "exporter")
				h.AssertEq(t, fakePhaseManager.WithArgsCallCount, 1)
				h.AssertIncludeAllExpectedArgPatterns(t,
					fakePhaseManager.WithArgsReceived,
					[]string{"-image", expectedRunImage},
					[]string{"-cache-dir", "/cache"},
					[]string{"-layers", "/layers"},
					[]string{"-app", "/workspace"},
					[]string{expectedRepoName},
					[]string{"-daemon"},
					[]string{"-launch-cache", "/launch-cache"},
				)
			})

			it("configures the phase with binds", func() {
				lifecycle := fakeLifecycle(t)
				fakePhaseManager := fakes.NewFakePhaseManager()
				expectedBinds := []string{"some-cache:/cache", "some-launch-cache:/launch-cache"}

				err := lifecycle.Export(context.Background(), "test", "test", false, "some-launch-cache", "some-cache", fakePhaseManager)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseManager.WithBindsCallCount, 1)
				h.AssertEq(t, fakePhaseManager.WithBindsReceived, expectedBinds)
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
