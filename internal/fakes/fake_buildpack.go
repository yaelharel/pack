package fakes

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/buildpack/pack/internal/archive"
	"github.com/buildpack/pack/internal/dist"
)

type fakeBuildpack struct {
	descriptor dist.BuildpackDescriptor
	chmod      int64
}

// NewFakeBuildpack creates a fake buildpacks with contents:
//
// 	\_ /cnbs/buildpacks/{ID}
// 	\_ /cnbs/buildpacks/{ID}/{version}
// 	\_ /cnbs/buildpacks/{ID}/{version}/buildpack.toml
// 	\_ /cnbs/buildpacks/{ID}/{version}/bin
// 	\_ /cnbs/buildpacks/{ID}/{version}/bin/build
//  	build-contents
// 	\_ /cnbs/buildpacks/{ID}/{version}/bin/detect
//  	detect-contents
func NewFakeBuildpack(descriptor dist.BuildpackDescriptor, chmod int64) (dist.Buildpack, error) {
	return &fakeBuildpack{
		descriptor: descriptor,
		chmod:      chmod,
	}, nil
}

func (b *fakeBuildpack) Descriptor() dist.BuildpackDescriptor {
	return b.descriptor
}

func (b *fakeBuildpack) Open() (reader io.ReadCloser, err error) {
	buf := &bytes.Buffer{}
	if err = toml.NewEncoder(buf).Encode(b.descriptor); err != nil {
		return nil, err
	}

	tarBuilder := archive.TarBuilder{}
	tarBuilder.AddDir(fmt.Sprintf("/cnb/buildpacks/%s", b.descriptor.EscapedID()), b.chmod, time.Now())
	bpDir := fmt.Sprintf("/cnb/buildpacks/%s/%s", b.descriptor.EscapedID(), b.descriptor.Info.Version)
	tarBuilder.AddDir(bpDir, b.chmod, time.Now())
	tarBuilder.AddFile(bpDir+"/buildpack.toml", b.chmod, time.Now(), buf.Bytes())
	tarBuilder.AddDir(bpDir+"/bin", b.chmod, time.Now())
	tarBuilder.AddFile(bpDir+"/bin/build", b.chmod, time.Now(), []byte("build-contents"))
	tarBuilder.AddFile(bpDir+"/bin/detect", b.chmod, time.Now(), []byte("detect-contents"))

	return tarBuilder.Reader(), err
}
