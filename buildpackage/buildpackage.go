package buildpackage

import "github.com/buildpack/pack/dist"

const MetadataLabel = "io.buildpacks.buildpackage.metadata"

type Config struct {
	Default    dist.BuildpackInfo `toml:"default"`
	Buildpacks []dist.Location    `toml:"buildpacks"`
	Packages   []dist.ImageRef    `toml:"packages"`
	Stacks     []dist.Stack       `toml:"stacks"`
}

type Metadata struct {
	dist.BuildpackInfo
	Stacks []dist.Stack `toml:"stacks" json:"stacks"`
}
