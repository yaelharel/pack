package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/project"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/logging"
)

type BuildFlags struct {
	AppPath        string
	Builder        string
	Registry       string
	RunImage       string
	Env            []string
	EnvFiles       []string
	Publish        bool
	NoPull         bool
	ClearCache     bool
	Buildpacks     []string
	Network        string
	DescriptorPath string
	Volumes        []string
}

func Build(logger logging.Logger, cfg config.Config, packClient PackClient) *cobra.Command {
	var flags BuildFlags
	ctx := createCancellableContext()

	cmd := &cobra.Command{
		Use:   "build <image-name>",
		Args:  cobra.ExactArgs(1),
		Short: "Generate app image from source code",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			imageName := args[0]
			if flags.Builder == "" {
				suggestSettingBuilder(logger, packClient)
				return MakeSoftError()
			}

			descriptor, actualDescriptorPath, err := parseProjectToml(flags.AppPath, flags.DescriptorPath)
			if err != nil {
				return err
			}
			if actualDescriptorPath != "" {
				logger.Debugf("Using project descriptor located at %s", style.Symbol(actualDescriptorPath))
			}

			env, err := parseEnv(descriptor, flags.EnvFiles, flags.Env)
			if err != nil {
				return err
			}

			buildpacks := flags.Buildpacks
			if len(buildpacks) == 0 {
				buildpacks = []string{}
				projectDescriptorDir := filepath.Dir(actualDescriptorPath)
				for _, bp := range descriptor.Build.Buildpacks {
					if len(bp.URI) == 0 {
						// there are several places through out the pack code where the "id@version" format is used.
						// we should probably central this, but it's not clear where it belongs
						buildpacks = append(buildpacks, fmt.Sprintf("%s@%s", bp.ID, bp.Version))
					} else {
						uri, err := paths.ToAbsolute(bp.URI, projectDescriptorDir)
						if err != nil {
							return err
						}
						buildpacks = append(buildpacks, uri)
					}
				}
			}

			if err := packClient.Build(ctx, pack.BuildOptions{
				AppPath:           flags.AppPath,
				Builder:           flags.Builder,
				Registry:          flags.Registry,
				AdditionalMirrors: getMirrors(cfg),
				RunImage:          flags.RunImage,
				Env:               env,
				Image:             imageName,
				Publish:           flags.Publish,
				NoPull:            flags.NoPull,
				ClearCache:        flags.ClearCache,
				Buildpacks:        buildpacks,
				ContainerConfig: pack.ContainerConfig{
					Network: flags.Network,
					Volumes: flags.Volumes,
				},
			}); err != nil {
				return err
			}
			logger.Infof("Successfully built image %s", style.Symbol(imageName))
			return nil
		}),
	}
	buildCommandFlags(cmd, &flags, cfg)
	cmd.Flags().BoolVar(&flags.Publish, "publish", false, "Publish to registry")
	AddHelpFlag(cmd, "build")
	return cmd
}

func buildCommandFlags(cmd *cobra.Command, buildFlags *BuildFlags, cfg config.Config) {
	cmd.Flags().StringVarP(&buildFlags.AppPath, "path", "p", "", "Path to app dir or zip-formatted file (defaults to current working directory)")
	cmd.Flags().StringVarP(&buildFlags.Builder, "builder", "B", cfg.DefaultBuilder, "Builder image")
	cmd.Flags().StringVarP(&buildFlags.Registry, "buildpack-registry", "r", cfg.DefaultRegistry, "Buildpack Registry URL")
	cmd.Flags().StringVar(&buildFlags.RunImage, "run-image", "", "Run image (defaults to default stack's run image)")
	cmd.Flags().StringArrayVarP(&buildFlags.Env, "env", "e", []string{}, "Build-time environment variable, in the form 'VAR=VALUE' or 'VAR'.\nWhen using latter value-less form, value will be taken from current\n  environment at the time this command is executed.\nThis flag may be specified multiple times and will override\n  individual values defined by --env-file.")
	cmd.Flags().StringArrayVar(&buildFlags.EnvFiles, "env-file", []string{}, "Build-time environment variables file\nOne variable per line, of the form 'VAR=VALUE' or 'VAR'\nWhen using latter value-less form, value will be taken from current\n  environment at the time this command is executed")
	cmd.Flags().BoolVar(&buildFlags.NoPull, "no-pull", false, "Skip pulling builder and run images before use")
	cmd.Flags().BoolVar(&buildFlags.ClearCache, "clear-cache", false, "Clear image's associated cache before building")
	cmd.Flags().StringSliceVarP(&buildFlags.Buildpacks, "buildpack", "b", nil, "Buildpack reference in the form of '<buildpack>@<version>',\n  path to a buildpack directory (not supported on Windows),\n  path/URL to a buildpack .tar or .tgz file, or\n  the name of a packaged buildpack image"+multiValueHelp("buildpack"))
	cmd.Flags().StringVar(&buildFlags.Network, "network", "", "Connect detect and build containers to network")
	cmd.Flags().StringVarP(&buildFlags.DescriptorPath, "descriptor", "d", "", "Path to the project descriptor file")
	cmd.Flags().StringArrayVar(&buildFlags.Volumes, "volume", nil, "Mount host volume into the build container, in the form '<host path>:<target path>'. Target path will be prefixed with '/platform/'"+multiValueHelp("volume"))
}

func parseEnv(project project.Descriptor, envFiles []string, envVars []string) (map[string]string, error) {
	env := map[string]string{}

	for _, envVar := range project.Build.Env {
		env[envVar.Name] = envVar.Value
	}
	for _, envFile := range envFiles {
		envFileVars, err := parseEnvFile(envFile)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse env file '%s'", envFile)
		}

		for k, v := range envFileVars {
			env[k] = v
		}
	}
	for _, envVar := range envVars {
		env = addEnvVar(env, envVar)
	}
	return env, nil
}

func parseEnvFile(filename string) (map[string]string, error) {
	out := make(map[string]string)
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "open %s", filename)
	}
	for _, line := range strings.Split(string(f), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = addEnvVar(out, line)
	}
	return out, nil
}

func addEnvVar(env map[string]string, item string) map[string]string {
	arr := strings.SplitN(item, "=", 2)
	if len(arr) > 1 {
		env[arr[0]] = arr[1]
	} else {
		env[arr[0]] = os.Getenv(arr[0])
	}
	return env
}

func parseProjectToml(appPath, descriptorPath string) (project.Descriptor, string, error) {
	actualPath := descriptorPath
	computePath := descriptorPath == ""

	if computePath {
		actualPath = filepath.Join(appPath, "project.toml")
	}

	if _, err := os.Stat(actualPath); err != nil {
		if computePath {
			return project.Descriptor{}, "", nil
		}
		return project.Descriptor{}, "", errors.Wrap(err, "stat project descriptor")
	}

	descriptor, err := project.ReadProjectDescriptor(actualPath)
	return descriptor, actualPath, err
}
