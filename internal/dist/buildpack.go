package dist

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpack/pack/internal/api"
	"github.com/buildpack/pack/internal/archive"
	"github.com/buildpack/pack/internal/style"
)

const AssumedBuildpackAPIVersion = "0.1"

type Blob interface {
	Open() (io.ReadCloser, error)
}

// TODO: rename this
type BuildpackImpl struct {
	descriptor BuildpackDescriptor
	Blob       `toml:"-"`
}

func (b *BuildpackImpl) Descriptor() BuildpackDescriptor {
	return b.descriptor
}

//go:generate mockgen -package testmocks -destination testmocks/mock_buildpack.go github.com/buildpack/pack/internal/dist Buildpack
type Buildpack interface {
	Blob
	Descriptor() BuildpackDescriptor
}

type BuildpackInfo struct {
	ID      string `toml:"id" json:"id"`
	Version string `toml:"version" json:"version,omitempty"`
}

func (b BuildpackInfo) FullName() string {
	if b.Version != "" {
		return b.ID + "@" + b.Version
	}
	return b.ID
}

type Stack struct {
	ID     string   `json:"id"`
	Mixins []string `json:"mixins,omitempty"`
}

// NewBuildpack constructs a buildpack from a blob. It is assumed that the buildpack contents reside at the root of the
// blob. The constructed buildpack contents will be structured as per the distribution spec (currently
// '/cnbs/buildpacks/{ID}/{version}/*').
func NewBuildpack(blob Blob) (*BuildpackImpl, error) {
	bpd := BuildpackDescriptor{}
	rc, err := blob.Open()
	if err != nil {
		return nil, errors.Wrap(err, "open buildpack")
	}
	defer rc.Close()

	_, buf, err := archive.ReadTarEntry(rc, "buildpack.toml")
	if err != nil {
		return nil, errors.Wrapf(err, "reading buildpack.toml")
	}

	bpd.API = api.MustParse(AssumedBuildpackAPIVersion)
	_, err = toml.Decode(string(buf), &bpd)
	if err != nil {
		return nil, errors.Wrapf(err, "decoding buildpack.toml")
	}

	err = validateDescriptor(bpd)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid buildpack.toml")
	}

	return &BuildpackImpl{descriptor: bpd, Blob: forDist(bpd, blob)}, nil
}

type distBlob struct {
	r io.ReadCloser
}

func (b *distBlob) Open() (io.ReadCloser, error) {
	return b.r, nil
}

func forDist(bpd BuildpackDescriptor, blob Blob) (Blob, error) {
	pr, pw := io.Pipe()

	tw := tar.NewWriter(pw)
	defer tw.Close()

	ts := archive.NormalizedDateTime

	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     path.Join(BuildpacksDir, bpd.EscapedID()),
		Mode:     0755,
		ModTime:  ts,
	}); err != nil {
		return nil, err
	}

	baseTarDir := path.Join(BuildpacksDir, bpd.EscapedID(), bpd.Info.Version)
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     baseTarDir,
		Mode:     0755,
		ModTime:  ts,
	}); err != nil {
		return nil, err
	}

	if err := embedBuildpackTar2(tw, uid, gid, blob, baseTarDir); err != nil {
		return nil, errors.Wrapf(err, "creating layer tar for buildpack '%s:%s'", bpd.Info.ID, bpd.Info.Version)
	}

	return &distBlob{
		r: pr,
	}, nil
}

func embedBuildpackTar2(tw *tar.Writer, uid, gid int, blob Blob, baseTarDir string) error {
	var (
		err error
	)

	rc, err := blob.Open()
	if err != nil {
		return errors.Wrap(err, "read buildpack blob")
	}
	defer rc.Close()

	tr := tar.NewReader(rc)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to get next tar entry")
		}

		header.Name = path.Clean(header.Name)
		if header.Name == "." || header.Name == "/" {
			continue
		}

		header.Name = path.Clean(path.Join(baseTarDir, header.Name))
		header.Uid = uid
		header.Gid = gid
		err = tw.WriteHeader(header)
		if err != nil {
			return errors.Wrapf(err, "failed to write header for '%s'", header.Name)
		}

		buf, err := ioutil.ReadAll(tr)
		if err != nil {
			return errors.Wrapf(err, "failed to read contents of '%s'", header.Name)
		}

		_, err = tw.Write(buf)
		if err != nil {
			return errors.Wrapf(err, "failed to write contents to '%s'", header.Name)
		}
	}

	return nil
}

func validateDescriptor(bpd BuildpackDescriptor) error {
	if bpd.Info.ID == "" {
		return errors.Errorf("%s is required", style.Symbol("buildpack.id"))
	}

	if bpd.Info.Version == "" {
		return errors.Errorf("%s is required", style.Symbol("buildpack.version"))
	}

	if len(bpd.Order) == 0 && len(bpd.Stacks) == 0 {
		return errors.Errorf(
			"buildpack %s: must have either %s or an %s defined",
			style.Symbol(bpd.Info.FullName()),
			style.Symbol("stacks"),
			style.Symbol("order"),
		)
	}

	if len(bpd.Order) >= 1 && len(bpd.Stacks) >= 1 {
		return errors.Errorf(
			"buildpack %s: cannot have both %s and an %s defined",
			style.Symbol(bpd.Info.FullName()),
			style.Symbol("stacks"),
			style.Symbol("order"),
		)
	}

	return nil
}
