package build_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/build"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestBindings(t *testing.T) {
	color.Disable(true)
	spec.Run(t, "testBindings", testBindings, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testBindings(t *testing.T, when spec.G, it spec.S) {
	when("#ReadBindingsConfig", func() {
		it("parses the config when it exists", func() {
			bindings, err := build.ReadBindingsConfig(strings.NewReader(`
[bravo]
[bravo.metadata]
kind = "bravo-kind"
provider = "bravo-provider"
tags = ["bravo-tag-1", "bravo-tag-2"]
other-key = "bravo-other-key"
`))
			h.AssertNil(t, err)

			h.AssertEq(t, bindings["bravo"].Metadata["kind"], "bravo-kind")
			h.AssertEq(t, bindings["bravo"].Metadata["provider"], "bravo-provider")
			h.AssertEq(t, bindings["bravo"].Metadata["tags"].([]interface{})[0], "bravo-tag-1")
			h.AssertEq(t, bindings["bravo"].Metadata["tags"].([]interface{})[1], "bravo-tag-2")
			h.AssertEq(t, bindings["bravo"].Metadata["other-key"], "bravo-other-key")
		})

		type testCase struct {
			requiredKey string
			content     string
		}

		for _, tc := range []testCase{
			{
				"kind", `
[bravo]
[bravo.metadata]
#kind = "bravo-kind"
provider = "bravo-provider"
tags = ["bravo-tag-1", "bravo-tag-2"]
`,
			},

			{
				"provider", `
[bravo]
[bravo.metadata]
kind = "bravo-kind"
#provider = "bravo-provider"
tags = ["bravo-tag-1", "bravo-tag-2"]
`,
			},

			{
				"tags", `
[bravo]
[bravo.metadata]
kind = "bravo-kind"
provider = "bravo-provider"
#tags = ["bravo-tag-1", "bravo-tag-2"]
`,
			},
		} {
			requiredKey := tc.requiredKey
			content := tc.content

			when(fmt.Sprintf("'%s' is missing", requiredKey), func() {
				it("returns an error", func() {
					_, err := build.ReadBindingsConfig(strings.NewReader(content))
					h.AssertError(t, err, fmt.Sprintf("binding 'bravo': missing metadata key '%s'", requiredKey))
				})
			})
		}
	})
}
