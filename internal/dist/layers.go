package dist

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
)

func BuildpackToLayerTar(dest string, bp Buildpack) (string, error) {
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

	if _, err := io.Copy(fh, bpReader); err != nil {
		return "", errors.Wrap(err, "writing buildpack blob to tar")
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

func TranslateLayerPath(layerPath, os string) string {
	if os == "windows" {
		return path.Join("Files", layerPath)
	}
	return layerPath
}

func InitializeWindowsLayer(tw *tar.Writer, paths ...string) error {
	paths = append([]string{"Files", "Hives"}, paths...)

	for _, path := range paths {
		if err := tw.WriteHeader(&tar.Header{Name: path, Typeflag: tar.TypeDir}); err != nil {
			return err
		}
	}

	return nil
}
