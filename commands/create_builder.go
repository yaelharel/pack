package commands

import (
	"context"
	"fmt"
	"github.com/buildpack/pack/container"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpack/pack"
	"github.com/buildpack/pack/builder"
	"github.com/buildpack/pack/logging"
	"github.com/buildpack/pack/style"
	dcontainer "github.com/docker/docker/api/types/container"
)

type CreateBuilderFlags struct {
	BuilderTomlPath string
	Publish         bool
	NoPull          bool
}

func CreateBuilder(logger *logging.Logger, client PackClient) *cobra.Command {
	var flags CreateBuilderFlags
	ctx := createCancellableContext()
	cmd := &cobra.Command{
		Use:   "create-builder <image-name> --builder-config <builder-config-path>",
		Args:  cobra.ExactArgs(1),
		Short: "Create builder image",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS == "windows" || true {
				return runForWindows(logger, ctx)
			}
			builderConfig, err := readBuilderConfig(flags.BuilderTomlPath)
			if err != nil {
				return errors.Wrap(err, "invalid builder toml")
			}
			imageName := args[0]
			if err := client.CreateBuilder(ctx, pack.CreateBuilderOptions{
				BuilderName:   imageName,
				BuilderConfig: builderConfig,
				Publish:       flags.Publish,
				NoPull:        flags.NoPull,
			}); err != nil {
				return err
			}
			logger.Info("Successfully created builder image %s", style.Symbol(imageName))
			logger.Tip("Run %s to use this builder", style.Symbol(fmt.Sprintf("pack build <image-name> --builder %s", imageName)))
			return nil
		}),
	}
	cmd.Flags().BoolVar(&flags.NoPull, "no-pull", false, "Skip pulling build image before use")
	cmd.Flags().StringVarP(&flags.BuilderTomlPath, "builder-config", "b", "", "Path to builder TOML file (required)")
	cmd.MarkFlagRequired("builder-config")
	cmd.Flags().BoolVar(&flags.Publish, "publish", false, "Publish to registry")
	AddHelpFlag(cmd, "create-builder")
	return cmd
}

func runForWindows(logger *logging.Logger, ctx context.Context) error {
	ctrConf := &dcontainer.Config{
		User:  "root",
		Image: "cnbs/build:0.0.1-rc.3", // TODO: Find better/smaller image, also fetch it
	}

	me, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "getting pack executable")
	}

	hostConf := &dcontainer.HostConfig{
		Binds: []string{
			me+":/pack:",
			"/var/run/docker.sock:/var/run/docker.sock",
			// fmt.Sprintf("%s:%s", l.LayersVolume, layersDir),
			// fmt.Sprintf("%s:%s", l.AppVolume, appDir),
		},
	}
	ctrConf.Cmd = append([]string{"/pack"}, os.Args...)

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
	if err != nil {
		return errors.Wrap(err, "create-builder docker client create")
	}

	ctr, err := dockerClient.ContainerCreate(ctx, ctrConf, hostConf, nil, "")
	if err != nil {
		return errors.Wrap(err, "create-builder container create")
	}
	defer dockerClient.ContainerRemove(context.Background(), ctr.ID, types.ContainerRemoveOptions{Force: true})
	return container.Run(
		ctx,
		dockerClient,
		ctr.ID,
		logger.RawWriter(),
		logger.RawErrorWriter(),
	)
}

func readBuilderConfig(path string) (builder.Config, error) {
	builderDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return builder.Config{}, err
	}

	builderConfig := builder.Config{}
	if _, err = toml.DecodeFile(path, &builderConfig); err != nil {
		return builderConfig, fmt.Errorf(`failed to decode builder config from file %s: %s`, path, err)
	}

	for i, bp := range builderConfig.Buildpacks {
		uri, err := transformRelativePath(bp.URI, builderDir)
		if err != nil {
			return builder.Config{}, errors.Wrap(err, "transforming buildpack URI")
		}
		builderConfig.Buildpacks[i].URI = uri
	}

	if builderConfig.Lifecycle.URI != "" {
		uri, err := transformRelativePath(builderConfig.Lifecycle.URI, builderDir)
		if err != nil {
			return builder.Config{}, errors.Wrap(err, "transforming lifecycle URI")
		}
		builderConfig.Lifecycle.URI = uri
	}

	return builderConfig, nil
}

func transformRelativePath(uri, relativeTo string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" {
		if !filepath.IsAbs(parsed.Path) {
			return fmt.Sprintf("file://" + filepath.Join(relativeTo, parsed.Path)), nil
		}
	}
	return uri, nil
}
