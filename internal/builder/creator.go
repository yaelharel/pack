package builder

import (
	pubbldr "github.com/buildpacks/pack/builder"
)

type Creator struct {
	configValidator ConfigValidator
}

type ConfigValidator interface {
	Validate(config pubbldr.Config) error
}

func NewCreator(configValidator ConfigValidator) *Creator {
	return &Creator{
		configValidator: configValidator,
	}
}

func (c *Creator) Create(config pubbldr.Config) error {
	return c.configValidator.Validate(config)
}
