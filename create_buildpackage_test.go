package pack_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/buildpack/imgutil/fakes"
	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpack/pack"
	"github.com/buildpack/pack/api"
	"github.com/buildpack/pack/buildpackage"
	"github.com/buildpack/pack/dist"
	ifakes "github.com/buildpack/pack/internal/fakes"
	"github.com/buildpack/pack/internal/logging"
	h "github.com/buildpack/pack/testhelpers"
	"github.com/buildpack/pack/testmocks"
)

func TestCreatePackage(t *testing.T) {
	color.Disable(true)
	defer func() { color.Disable(false) }()
	spec.Run(t, "CreatePackage", testCreatePackage, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testCreatePackage(t *testing.T, when spec.G, it spec.S) {
	var (
		client           *pack.Client
		mockController   *gomock.Controller
		mockDownloader   *testmocks.MockDownloader
		mockImageFactory *testmocks.MockImageFactory
		mockImageFetcher *testmocks.MockImageFetcher
		fakePackageImage *fakes.Image
		tmpDir           string
		out              bytes.Buffer
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockDownloader = testmocks.NewMockDownloader(mockController)
		mockImageFetcher = testmocks.NewMockImageFetcher(mockController)
		mockImageFactory = testmocks.NewMockImageFactory(mockController)

		fakePackageImage = fakes.NewImage("some/package", "", "")
		mockImageFactory.EXPECT().NewImage("some/package", true).Return(fakePackageImage, nil).AnyTimes()

		var err error
		tmpDir, err = ioutil.TempDir("", "create-package-layers")
		h.AssertNil(t, err)

		client, err = pack.NewClient(
			pack.WithLogger(logging.NewLogWithWriters(&out, &out)),
			pack.WithDownloader(mockDownloader),
			pack.WithImageFactory(mockImageFactory),
			pack.WithImageFetcher(mockImageFetcher),
		)
		h.AssertNil(t, err)
	})

	it.After(func() {
		fakePackageImage.Cleanup()
		mockController.Finish()
	})

	when("#CreatePackage", func() {
		when("package config is valid", func() {
			var (
				opts                     pack.CreatePackageOptions
				fakeExistingPackageImage *fakes.Image
			)

			it.Before(func() {
				opts = pack.CreatePackageOptions{
					Name: fakePackageImage.Name(),
					Config: buildpackage.Config{
						Default: dist.BuildpackInfo{
							ID:      "bp.one",
							Version: "1.2.3",
						},
						Buildpacks: []dist.Location{
							{URI: "https://example.com/bp.one.tgz"},
						},
						Packages: []dist.ImageRef{
							{Reference: "package/bp2-bp3"},
						},
						Stacks: []dist.Stack{
							{ID: "some.stack.id"},
						},
					},
				}

				bp1, err := ifakes.NewBuildpackFromDescriptor(dist.BuildpackDescriptor{
					API: api.MustParse("0.2"),
					Info: dist.BuildpackInfo{
						ID:      "bp.one",
						Version: "1.2.3",
					},
					Stacks: []dist.Stack{
						{ID: "some.stack.id"},
					},
					Order: nil,
				}, 0644)
				h.AssertNil(t, err)

				mockDownloader.EXPECT().Download(gomock.Any(), "https://example.com/bp.one.tgz").Return(bp1, nil).AnyTimes()

				bp2, err := ifakes.NewBuildpackFromDescriptor(dist.BuildpackDescriptor{
					API: api.MustParse("0.2"),
					Info: dist.BuildpackInfo{
						ID:      "bp.two",
						Version: "4.5.6",
					},
					Stacks: []dist.Stack{
						{ID: "some.stack.id"},
					},
					Order: nil,
				}, 0644)
				h.AssertNil(t, err)

				bp3, err := ifakes.NewBuildpackFromDescriptor(dist.BuildpackDescriptor{
					API: api.MustParse("0.2"),
					Info: dist.BuildpackInfo{
						ID:      "bp.three",
						Version: "7.8.9",
					},
					Stacks: []dist.Stack{
						{ID: "some.stack.id"},
					},
					Order: nil,
				}, 0644)
				h.AssertNil(t, err)

				fakeExistingPackageImage = fakes.NewImage("package/bp2-bp3", "", "")
				h.AssertNil(t, err)

				addBuildpackToImage(t, tmpDir, fakeExistingPackageImage, bp2)
				addBuildpackToImage(t, tmpDir, fakeExistingPackageImage, bp3)

				_, err = fakeExistingPackageImage.Save()
				h.AssertNil(t, err)

				mockImageFetcher.EXPECT().Fetch(context.TODO(), "package/bp2-bp3", true, true).Return(fakeExistingPackageImage, nil)
			})

			it.After(func() {
				fakeExistingPackageImage.Cleanup()
			})

			it("sets metadata", func() {
				h.AssertNil(t, client.CreatePackage(context.TODO(), opts))
				h.AssertEq(t, fakePackageImage.IsSaved(), true)

				labelData, err := fakePackageImage.Label("io.buildpacks.buildpackage.metadata")
				h.AssertNil(t, err)
				var md buildpackage.Metadata
				h.AssertNil(t, json.Unmarshal([]byte(labelData), &md))

				h.AssertEq(t, md.ID, "bp.one")
				h.AssertEq(t, md.Version, "1.2.3")
				h.AssertEq(t, len(md.Stacks), 1)
				h.AssertEq(t, md.Stacks[0].ID, "some.stack.id")
			})

			it("adds buildpack layers", func() {
				h.AssertNil(t, client.CreatePackage(context.TODO(), opts))
				h.AssertEq(t, fakePackageImage.IsSaved(), true)

				assertLayerExists(t, fakePackageImage, "bp.one", "1.2.3")
				assertLayerExists(t, fakePackageImage, "bp.two", "4.5.6")
				assertLayerExists(t, fakePackageImage, "bp.three", "7.8.9")
			})

			when("when publish is true", func() {
				var fakeRemotePackageImage *fakes.Image

				it.Before(func() {
					fakeRemotePackageImage = fakes.NewImage("some/package", "", "")
					mockImageFactory.EXPECT().NewImage("some/package", false).Return(fakeRemotePackageImage, nil).AnyTimes()

					opts.Publish = true
				})

				it.After(func() {
					fakeRemotePackageImage.Cleanup()
				})

				it("saves remote image", func() {
					h.AssertNil(t, client.CreatePackage(context.TODO(), opts))
					h.AssertEq(t, fakeRemotePackageImage.IsSaved(), true)
				})
			})
		})
	})
}

func addBuildpackToImage(t *testing.T, tmpDir string, image *fakes.Image, bp dist.Buildpack) {
	tarFile, err := ioutil.TempFile(tmpDir, "bp-*.tar")
	h.AssertNil(t, err)

	defer func() {
		h.AssertNil(t, tarFile.Close())
		h.AssertNil(t, os.Remove(tarFile.Name()))
	}()

	r, err := bp.Open()
	h.AssertNil(t, err)

	_, err = io.Copy(tarFile, r)
	h.AssertNil(t, err)

	h.AssertNil(t, image.AddLayer(tarFile.Name()))
}

func assertLayerExists(t *testing.T, image *fakes.Image, id, version string) {
	t.Helper()
	
	dirPath := fmt.Sprintf("/cnb/buildpacks/%s/%s", id, version)
	layerTar, err := image.FindLayerWithPath(dirPath)
	h.AssertNil(t, err)

	h.AssertOnTarEntry(t, layerTar, dirPath,
		h.IsDirectory(),
	)

	h.AssertOnTarEntry(t, layerTar, dirPath+"/bin/build",
		h.ContentEquals("build-contents"),
		h.HasOwnerAndGroup(0, 0),
		h.HasFileMode(0644),
	)

	h.AssertOnTarEntry(t, layerTar, dirPath+"/bin/detect",
		h.ContentEquals("detect-contents"),
		h.HasOwnerAndGroup(0, 0),
		h.HasFileMode(0644),
	)
}
