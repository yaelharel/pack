package dist

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildpacks/pack/internal/layer"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
)

func BuildpackToLayerTar(dest string, bp Buildpack, imageOS string) (string, error) {
	bpd := bp.Descriptor()
	bpReader, err := bp.Open()
	if err != nil {
		return "", errors.Wrap(err, "opening buildpack blob")
	}
	defer bpReader.Close()

	layerTar := filepath.Join(dest, fmt.Sprintf("%s.%s.tar", bpd.EscapedID(), bpd.Info.Version))
	fh, err := os.Create(layerTar)
	if err != nil {
		return "", errors.Wrap(err, "create file for tar")
	}
	defer fh.Close()

	tw := tar.NewReader(bpReader)
	lw := layer.NewWriter(fh, imageOS)
	defer lw.Close()

	if err := lw.FromTarReader(tw); err != nil {
		return "", errors.Wrap(err, "writing buildpack blob to layer tar")
	}

	return layerTar, nil
}

func LayerDiffID(layerTarPath string) (v1.Hash, error) {
	fh, err := os.Open(layerTarPath)
	if err != nil {
		return v1.Hash{}, errors.Wrap(err, "opening tar file")
	}
	defer fh.Close()

	layer, err := tarball.LayerFromFile(layerTarPath)
	if err != nil {
		return v1.Hash{}, errors.Wrap(err, "reading layer tar")
	}

	hash, err := layer.DiffID()
	if err != nil {
		return v1.Hash{}, errors.Wrap(err, "generating diff id")
	}

	return hash, nil
}
