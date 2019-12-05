package dist

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"path"

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
type buildpack struct {
	descriptor BuildpackDescriptor
	Blob       `toml:"-"`
}

func (b *buildpack) Descriptor() BuildpackDescriptor {
	return b.descriptor
}

//go:generate mockgen -package testmocks -destination testmocks/mock_buildpack.go github.com/buildpack/pack/internal/dist Buildpack
type Buildpack interface {
	// Open returns a reader with contents structured as per the distribution spec
	// (currently '/cnbs/buildpacks/{ID}/{version}/*').
	Open() (io.ReadCloser, error)
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

// BuildpackFromRootBlob constructs a buildpack from a blob. It is assumed that the buildpack contents reside at the root of the
// blob. The constructed buildpack contents will be structured as per the distribution spec (currently
// '/cnbs/buildpacks/{ID}/{version}/*').
func BuildpackFromRootBlob(blob Blob) (*buildpack, error) {
	bpd := BuildpackDescriptor{}
	rc, err := blob.Open()
	if err != nil {
		return nil, errors.Wrap(err, "open buildpack")
	}
	defer rc.Close()

	_, buf, err := archive.ReadTarEntry(rc, "buildpack.toml")
	if err != nil {
		return nil, errors.Wrap(err, "reading buildpack.toml")
	}

	bpd.API = api.MustParse(AssumedBuildpackAPIVersion)
	_, err = toml.Decode(string(buf), &bpd)
	if err != nil {
		return nil, errors.Wrap(err, "decoding buildpack.toml")
	}

	err = validateDescriptor(bpd)
	if err != nil {
		return nil, errors.Wrap(err, "invalid buildpack.toml")
	}
	
	db, err := toDistBlob(bpd, blob)
	if err != nil {
		return nil, errors.Wrap(err, "creating distribution blob")
	}
	
	return &buildpack{descriptor: bpd, Blob: db}, nil
}

// BuildpackFromTarReadCloser constructs a buildpack from a ReadCloser to a tar. It is assumed that the buildpack
// contents are structured as per the distribution spec (currently '/cnbs/buildpacks/{ID}/{version}/*').
func BuildpackFromTarReadCloser(bpd BuildpackDescriptor, rc io.ReadCloser) *buildpack {
	return &buildpack{
		Blob: &distBlob{
			rc: rc,
		},
		descriptor: bpd,
	}
}

type distBlob struct {
	rc io.ReadCloser
}

func (b *distBlob) Open() (io.ReadCloser, error) {
	return b.rc, nil
}

// main thread        coroutine
// r* <--------------- w*
// [r  w]-->tw <- 
func toDistBlob(bpd BuildpackDescriptor, blob Blob) (Blob, error) {
	pr, pw := io.Pipe()
	
	tw := tar.NewWriter(pw)
	defer tw.Close()

	ts := archive.NormalizedDateTime

	go func() {
		if err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     path.Join(BuildpacksDir, bpd.EscapedID()),
			Mode:     0755,
			ModTime:  ts,
		}); err != nil {
			// return nil, err
			panic("fooooooo!")
		}
	
		baseTarDir := path.Join(BuildpacksDir, bpd.EscapedID(), bpd.Info.Version)
		if err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     baseTarDir,
			Mode:     0755,
			ModTime:  ts,
		}); err != nil {
			// return nil, err
			panic("fooooooo!!!!!!11111")
		}
	
		if err := writeTar(tw, blob, baseTarDir); err != nil {
			// return nil, errors.Wrapf(err, "creating layer tar for buildpack '%s:%s'", bpd.Info.ID, bpd.Info.Version)
			panic("fooooooo!!!!!!11111222222345trrt")
		}
	}()

	return &distBlob{
		rc: pr,
	}, nil
}

func writeTar(tw *tar.Writer, blob Blob, baseTarDir string) error {
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
		header.Uid = 0
		header.Gid = 0
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
