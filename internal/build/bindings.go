package build

import (
	"io"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
)

type BindingsConfig map[string]BindingConfig

type BindingConfig struct {
	Metadata map[string]interface{} `toml:"metadata"`
	Secrets  map[string]interface{} `toml:"secrets"`
}

func ReadBindingsConfig(configReader io.Reader) (BindingsConfig, error) {
	var bindings BindingsConfig
	_, err := toml.DecodeReader(configReader, &bindings)
	if err != nil {
		return BindingsConfig{}, errors.Wrap(err, "parsing bindings config")
	}

	for binding, config := range bindings {
		for _, k := range []string{"kind", "provider", "tags"} {
			if _, ok := config.Metadata[k]; !ok {
				return nil, errors.Errorf("binding %s: missing metadata key %s",
					style.Symbol(binding),
					style.Symbol(k),
				)
			}
		}
	}

	return bindings, nil
}
