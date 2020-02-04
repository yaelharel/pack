package builder

import (
	pubbldr "github.com/buildpacks/pack/builder"
)

type Creator struct {
}

type ConfigValidator interface {
	Validate(config pubbldr.Config) error
}

func NewCreator() *Creator {
	return &Creator{}
}

func (c *Creator) Create() error {
	return nil
}
