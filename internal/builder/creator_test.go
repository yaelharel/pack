package builder_test

import (
	"errors"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	pubbldr "github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/internal/builder"
	h "github.com/buildpacks/pack/testhelpers"
)

type fakeBuilderConfigValidator struct {
	validateReturns error
}

func (b *fakeBuilderConfigValidator) Validate(config pubbldr.Config) error {
	return b.validateFunc(config)
}

func TestCreator(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Creator", testCreator, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testCreator(t *testing.T, when spec.G, it spec.S) {
	when("Create", func() {
		var (
			builderConfigValidator *fakeBuilderConfigValidator
			builderCreator         *builder.Creator
		)

		it.Before(func() {
			builderConfigValidator = &fakeBuilderConfigValidator{}

			builderCreator = builder.NewCreator(builderConfigValidator)
		})

		it("succeeds", func() {
			err := builderCreator.Create()
			h.AssertNil(t, err)
		})

		when("builder config validator returns an error", func() {
			it("returns an error", func() {
				builderConfigValidator.validateReturns = errors.New("Something went wrong")

				err := builderCreator.Create()
			})
		})

		when("builder config validator returns an error", func() {
			it("returns an error", func() {
				builderCreator = localCreatorMaker(withValidator(&fakeBuilderConfigValidator{validateReturns: errors.New("")}))

				err := builderCreator.Create()
			})
		})
	})
}

func localCreatorMaker(ops ...func(creator *builder.Creator)) {

}
