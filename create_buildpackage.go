package pack

import (
	"context"
	"io"

	"github.com/pkg/errors"

	"github.com/buildpack/pack/builder"
	"github.com/buildpack/pack/buildpackage"
	"github.com/buildpack/pack/dist"
	"github.com/buildpack/pack/style"
)

type CreatePackageOptions struct {
	Name    string
	Config  buildpackage.Config
	Publish bool
}

func (c *Client) CreatePackage(ctx context.Context, opts CreatePackageOptions) error {
	packageBuilder := buildpackage.NewBuilder(c.imageFactory)

	for _, bpLoc := range opts.Config.Buildpacks {
		blob, err := c.downloader.Download(ctx, bpLoc.URI)
		if err != nil {
			return errors.Wrapf(err, "downloading buildpack from %s", style.Symbol(bpLoc.URI))
		}

		bp, err := dist.NewBuildpack(blob)
		if err != nil {
			return errors.Wrapf(err, "creating buildpack from %s", style.Symbol(bpLoc.URI))
		}

		packageBuilder.AddBuildpack(bp)
	}

	for _, p := range opts.Config.Packages {
		// TODO: daemon/pull logic?
		img, err := c.imageFetcher.Fetch(ctx, p.Reference, true, true)
		if err != nil {
			return errors.Wrapf(err, "reading buildpacks from package %s", style.Symbol(p.Reference))
		}

		bpLayers := builder.BuildpackLayers{}
		if _, err := dist.GetLabel(img, builder.BuildpackLayersLabel, &bpLayers); err != nil {
			return err
		}

		for id, vInfo := range bpLayers {
			for version, bpInfo := range vInfo {
				// TODO: if this is compressed will it work on daemon?
				readCloser, err := img.GetLayer(bpInfo.LayerDigest) 
				if err != nil {
					return errors.Wrapf(err, "reading layer from package %s", style.Symbol(p.Reference))
				}

				defer readCloser.Close()

				bp, err := dist.NewBuildpack(&ReaderBlob{reader: readCloser})
				packageBuilder.AddBuildpack(bp)
			}
		}
	}

	packageBuilder.SetDefaultBuildpack(opts.Config.Default)

	for _, s := range opts.Config.Stacks {
		packageBuilder.AddStack(s)
	}

	_, err := packageBuilder.Save(opts.Name, opts.Publish)
	if err != nil {
		return errors.Wrapf(err, "saving image")
	}

	return err
}

type ReaderBlob struct {
	reader io.ReadCloser
}

func (r *ReaderBlob) Open() (io.ReadCloser, error) {
	return r.reader, nil
}
