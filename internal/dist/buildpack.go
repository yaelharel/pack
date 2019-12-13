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
const BuildpacksDir = "/cnb/buildpacks"

type Blob interface {
	Open() (io.ReadCloser, error)
}

type buildpack struct {
	descriptor BuildpackDescriptor
	Blob       `toml:"-"`
}

func (b *buildpack) Descriptor() BuildpackDescriptor {
	return b.descriptor
}

//go:generate mockgen -package testmocks -destination testmocks/mock_buildpack.go github.com/buildpack/pack/internal/dist Buildpack
type Buildpack interface {
	// Open returns a reader to a tar with contents structured as per the distribution spec
	// (currently '/cnbs/buildpacks/{ID}/{version}/*', all entries with a zeroed-out
	// timestamp and root UID/GID).
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
// a tar with contents under '/cnbs/buildpacks/{ID}/{version}/*').
func BuildpackFromRootBlob(blob Blob) (Buildpack, error) {
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
	
	return &buildpack{
		descriptor: bpd,
		Blob:       toDistBlob(bpd, blob),
	}, nil
}

// BuildpackFromTarBlob constructs a buildpack from a ReadCloser to a tar. It is assumed that the buildpack
// contents are structured as per the distribution spec (currently '/cnbs/buildpacks/{ID}/{version}/*').
func BuildpackFromTarBlob(bpd BuildpackDescriptor, blob Blob) Buildpack {
	return &buildpack{
		Blob:       blob,
		descriptor: bpd,
	}
}

type distBlob struct {
	rc io.ReadCloser
}

func (b *distBlob) Open() (io.ReadCloser, error) {
	return b.rc, nil
}

func toDistBlob(bpd BuildpackDescriptor, blob Blob) Blob {
	// errChan := make(chan error, 1)
	pr, pw := io.Pipe()
	tw := tar.NewWriter(pw)
	ts := archive.NormalizedDateTime

	go func() {
		var err error
		defer func() {
			pw.CloseWithError(err)
		}()
		
		defer tw.Close()

		if err = tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     path.Join(BuildpacksDir, bpd.EscapedID()),
			Mode:     0755,
			ModTime:  ts,
		}); err != nil {
			return
		}

		baseTarDir := path.Join(BuildpacksDir, bpd.EscapedID(), bpd.Info.Version)
		if err = tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     baseTarDir,
			Mode:     0755,
			ModTime:  ts,
		}); err != nil {
			return
		}
	
		if err = writeTar(tw, blob, baseTarDir); err != nil {
			err = errors.Wrapf(err, "creating layer tar for buildpack '%s:%s'", bpd.Info.ID, bpd.Info.Version)
			return
		}
	}()

	return &distBlob{
		rc: pr,
	}
	// return &distBlob{
	// 	rc: &MyNewReader{
	// 		source:  pr,
	// 		errChan: errChan,
	// 	},
	// }
}

type MyNewReader struct {
	source   io.ReadCloser
	errChan  chan error
	foundErr error
}

func (r *MyNewReader) Read(p []byte) (n int, err error) {
	go func() {
		r.foundErr = <-r.errChan
	}()

	n, err = r.source.Read(p)

	if r.foundErr != nil {
		return n, r.foundErr
	}
	return n, err
}

func (r *MyNewReader) Close() error {
	err := r.source.Close()

	return err
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

		header.Mode = calcFileMode(header)
		header.Name = path.Join(baseTarDir, header.Name)
		header.Uid = 0
		header.Gid = 0
		err = tw.WriteHeader(header)
		if err != nil {
			return errors.Wrapf(err, "failed to write header for '%s'", header.Name)
		}

		// TODO: copy here instead?
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

func calcFileMode(header *tar.Header) int64 {
	switch {
	case header.Typeflag == tar.TypeDir:
		return 0755
	case nameOneOf(header.Name,
		path.Join("bin", "detect"),
		path.Join("bin", "build"),
	):
		return 0755
	case anyExecBit(header.Mode):
		return 0755
	}

	return 0644
}

func nameOneOf(name string, paths ...string) bool {
	for _, p := range paths {
		if name == p {
			return true
		}
	}
	return false
}

func anyExecBit(mode int64) bool {
	return mode&0111 != 0
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
