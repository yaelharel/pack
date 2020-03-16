package commands

import (
	"fmt"
	"github.com/buildpacks/pack/internal/config"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack"
	"github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/logging"
)

type CreateBuilderFlags struct {
	BuilderTomlPath string
	Publish         bool
	NoPull          bool
	Registry        string
}

func CreateBuilder(logger logging.Logger, cfg config.Config, client PackClient) *cobra.Command {
	var flags CreateBuilderFlags
	ctx := createCancellableContext()
	cmd := &cobra.Command{
		Use:   "create-builder <image-name> --builder-config <builder-config-path>",
		Args:  cobra.ExactArgs(1),
		Short: "Create builder image",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			builderConfig, warns, err := builder.ReadConfig(flags.BuilderTomlPath)
			if err != nil {
				return errors.Wrap(err, "invalid builder toml")
			}
			for _, w := range warns {
				logger.Warnf("builder configuration: %s", w)
			}

			imageName := args[0]
			if err := client.CreateBuilder(ctx, pack.CreateBuilderOptions{
				BuilderName: imageName,
				Config:      builderConfig,
				Publish:     flags.Publish,
				NoPull:      flags.NoPull,
				Registry:    flags.Registry,
			}); err != nil {
				return err
			}
			logger.Infof("Successfully created builder image %s", style.Symbol(imageName))
			logging.Tip(logger, "Run %s to use this builder", style.Symbol(fmt.Sprintf("pack build <image-name> --builder %s", imageName)))
			return nil
		}),
	}
	cmd.Flags().BoolVar(&flags.NoPull, "no-pull", false, "Skip pulling build image before use")
	cmd.Flags().StringVarP(&flags.BuilderTomlPath, "builder-config", "b", "", "Path to builder TOML file (required)")
	cmd.Flags().StringVarP(&flags.Registry, "buildpack-registry", "r", cfg.DefaultRegistry, "Buildpack Registry URL")
	cmd.MarkFlagRequired("builder-config")
	cmd.Flags().BoolVar(&flags.Publish, "publish", false, "Publish to registry")
	AddHelpFlag(cmd, "create-builder")
	return cmd
}
