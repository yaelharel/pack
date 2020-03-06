package build

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver"
)

const (
	layersDir      = "/layers"
	appDir         = "/workspace"
	cacheDir       = "/cache"
	launchCacheDir = "/launch-cache"
	platformDir    = "/platform"
)

type RunnerCleaner interface {
	Cleanup() error
	Run(ctx context.Context) error
}

type PhaseManager interface {
	New(name string, ops ...PhaseOperation) (RunnerCleaner, error)
	WithArgs(args ...string) PhaseOperation
	WithNetwork(networkMode string) PhaseOperation
	WithDaemonAccess() PhaseOperation
	WithBinds(binds ...string) PhaseOperation
	WithRegistryAccess(repos ...string) PhaseOperation
	WithRoot() PhaseOperation
}

func (l *Lifecycle) Detect(ctx context.Context, networkMode string, volumes []string, phaseManager PhaseManager) error {
	detect, err := phaseManager.New(
		"detector",
		phaseManager.WithArgs(
			l.withLogLevel(
				"-app", appDir,
				"-platform", platformDir,
			)...,
		),
		phaseManager.WithNetwork(networkMode),
		phaseManager.WithBinds(volumes...),
	)
	if err != nil {
		return err
	}
	defer detect.Cleanup()
	return detect.Run(ctx)
}

func (l *Lifecycle) Restore(ctx context.Context, cacheName string, phaseManager PhaseManager) error {
	restore, err := phaseManager.New(
		"restorer",
		phaseManager.WithDaemonAccess(),
		phaseManager.WithArgs(
			l.withLogLevel(
				"-cache-dir", cacheDir,
				"-layers", layersDir,
			)...,
		),
		phaseManager.WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
	)
	if err != nil {
		return err
	}
	defer restore.Cleanup()
	return restore.Run(ctx)
}

func (l *Lifecycle) Analyze(ctx context.Context, repoName, cacheName string, publish, clearCache bool, phaseManager PhaseManager) error {
	analyze, err := l.newAnalyze(repoName, cacheName, publish, clearCache, phaseManager)
	if err != nil {
		return err
	}
	defer analyze.Cleanup()
	return analyze.Run(ctx)
}

func (l *Lifecycle) newAnalyze(repoName, cacheName string, publish, clearCache bool, phaseManager PhaseManager) (RunnerCleaner, error) {
	args := []string{
		"-layers", layersDir,
		repoName,
	}
	if clearCache {
		args = prependArg("-skip-layers", args)
	} else {
		args = append([]string{"-cache-dir", cacheDir}, args...)
	}

	if publish {
		return phaseManager.New(
			"analyzer",
			phaseManager.WithRegistryAccess(repoName),
			phaseManager.WithRoot(),
			phaseManager.WithArgs(args...),
			phaseManager.WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
		)
	}
	return phaseManager.New(
		"analyzer",
		phaseManager.WithDaemonAccess(),
		phaseManager.WithArgs(
			l.withLogLevel(
				prependArg(
					"-daemon",
					args,
				)...,
			)...,
		),
		phaseManager.WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
	)
}

func prependArg(arg string, args []string) []string {
	return append([]string{arg}, args...)
}

func (l *Lifecycle) Build(ctx context.Context, networkMode string, volumes []string) error {
	build, err := l.NewPhase(
		"builder",
		WithArgs(
			"-layers", layersDir,
			"-app", appDir,
			"-platform", platformDir,
		),
		WithNetwork(networkMode),
		WithBinds(volumes...),
	)
	if err != nil {
		return err
	}
	defer build.Cleanup()
	return build.Run(ctx)
}

func (l *Lifecycle) Export(ctx context.Context, repoName string, runImage string, publish bool, launchCacheName, cacheName string) error {
	export, err := l.newExport(repoName, runImage, publish, launchCacheName, cacheName)
	if err != nil {
		return err
	}
	defer export.Cleanup()
	return export.Run(ctx)
}

func (l *Lifecycle) newExport(repoName, runImage string, publish bool, launchCacheName, cacheName string) (*Phase, error) {
	args := []string{
		"-image", runImage,
		"-cache-dir", cacheDir,
		"-layers", layersDir,
		"-app", appDir,
		repoName,
	}

	binds := []string{fmt.Sprintf("%s:%s", cacheName, cacheDir)}

	if publish {
		return l.NewPhase(
			"exporter",
			WithRegistryAccess(repoName, runImage),
			WithArgs(
				l.withLogLevel(args...)...,
			),
			WithRoot(),
			WithBinds(binds...),
		)
	}

	args = append([]string{"-daemon", "-launch-cache", launchCacheDir}, args...)
	binds = append(binds, fmt.Sprintf("%s:%s", launchCacheName, launchCacheDir))
	return l.NewPhase(
		"exporter",
		WithDaemonAccess(),
		WithArgs(
			l.withLogLevel(args...)...,
		),
		WithBinds(binds...),
	)
}

func (l *Lifecycle) withLogLevel(args ...string) []string {
	version := semver.MustParse(l.version)
	if semver.MustParse("0.4.0").LessThan(version) {
		if l.logger.IsVerbose() {
			return append([]string{"-log-level", "debug"}, args...)
		}
	}
	return args
}
