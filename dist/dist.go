package dist

type Order []OrderEntry

type OrderEntry struct {
	Group []BuildpackRef `toml:"group" json:"group"`
}

type Location struct {
	URI string `toml:"uri"`
}

type BuildpackRef struct {
	BuildpackInfo
	Optional bool `toml:"optional,omitempty" json:"optional,omitempty"`
}
