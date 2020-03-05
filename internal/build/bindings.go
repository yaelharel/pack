package build

import (
	"io"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
)

type ServiceBindingsConfig struct {
	Services []ServiceBindingConfig `toml:"services"`
}

type ServiceBindingConfig struct {
	Name     string                 `toml:"name"`
	Metadata map[string]interface{} `toml:"metadata"`
	Secrets  map[string]interface{} `toml:"secrets"`
}

func ReadBindingsConfig(configReader io.Reader) (ServiceBindingsConfig, error) {
	var bindings ServiceBindingsConfig
	_, err := toml.DecodeReader(configReader, &bindings)
	if err != nil {
		return ServiceBindingsConfig{}, errors.Wrap(err, "parsing service bindings")
	}

	for _, sb := range bindings.Services {
		for _, k := range []string{"kind", "provider", "tags"} {
			if _, ok := sb.Metadata[k]; !ok {
				return ServiceBindingsConfig{}, errors.Errorf("binding %s: missing metadata key %s",
					style.Symbol(sb.Name),
					style.Symbol(k),
				)
			}
		}
	}

	return bindings, nil
}
