package builder_test

import (
	"testing"

	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	pubbldr "github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/internal/builder"
	h "github.com/buildpacks/pack/testhelpers"
)

type fakeBuilderConfigValidator struct {
	validateReturns error
	argForValidate  pubbldr.Config
}

func (b *fakeBuilderConfigValidator) Validate(config pubbldr.Config) error {
	b.argForValidate = config

	return b.validateReturns
}

func TestCreator(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Creator", testCreator, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testCreator(t *testing.T, when spec.G, it spec.S) {
	when("Create", func() {
		it("succeeds", func() {
			builderCreator := newCreator()

			config := pubbldr.Config{}

			err := builderCreator.Create(config)
			h.AssertNil(t, err)
		})

		it("calls the validator with the config passed in to create", func() {
			fakeBuilderConfigValidator := &fakeBuilderConfigValidator{}

			builderCreator := newCreator(withValidator(fakeBuilderConfigValidator))

			config := pubbldr.Config{Description: "Right!"}

			err := builderCreator.Create(config)
			h.AssertNil(t, err)

			h.AssertEq(t, fakeBuilderConfigValidator.argForValidate, config)
		})

		when("builder config validator returns an error", func() {
			it("returns an error", func() {
				builderCreator := newCreator(withValidator(&fakeBuilderConfigValidator{
					validateReturns: errors.New("Something went wrong"),
				}))

				config := pubbldr.Config{}

				err := builderCreator.Create(config)
				h.AssertNotNil(t, err)
			})
		})
	})
}

type creatorDependencies struct {
	configValidator *fakeBuilderConfigValidator
}

type creatorOption func(creator *creatorDependencies)

func withValidator(validator *fakeBuilderConfigValidator) creatorOption {
	return func(creator *creatorDependencies) {
		creator.configValidator = validator
	}
}

func newCreator(ops ...creatorOption) *builder.Creator {
	conf := &creatorDependencies{
		configValidator: &fakeBuilderConfigValidator{},
	}

	for _, op := range ops {
		op(conf)
	}

	return builder.NewCreator(conf.configValidator)
}
