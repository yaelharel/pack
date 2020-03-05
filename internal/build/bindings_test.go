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
[[services]]
  name = "bravo"
  
  [services.metadata]
  kind = "bravo-kind"
  provider = "bravo-provider"
  tags = ["bravo-tag-1", "bravo-tag-2"]
  some-key = "bravo-key"
  other-keys = ["bravo-key-1", "bravo-key-2"]

  [services.secrets]
  some-secret = "bravo-secret"
  other-secrets = ["bravo-secret-1", "bravo-secret-2"]
`))
			h.AssertNil(t, err)

			h.AssertEq(t, bindings.Services[0].Name, "bravo")
			h.AssertEq(t, bindings.Services[0].Metadata["kind"], "bravo-kind")
			h.AssertEq(t, bindings.Services[0].Metadata["provider"], "bravo-provider")
			h.AssertEq(t, bindings.Services[0].Metadata["tags"].([]interface{})[0], "bravo-tag-1")
			h.AssertEq(t, bindings.Services[0].Metadata["tags"].([]interface{})[1], "bravo-tag-2")
			h.AssertEq(t, bindings.Services[0].Metadata["some-key"], "bravo-key")
			h.AssertEq(t, bindings.Services[0].Metadata["other-keys"].([]interface{})[0], "bravo-key-1")
			h.AssertEq(t, bindings.Services[0].Metadata["other-keys"].([]interface{})[1], "bravo-key-2")
			h.AssertEq(t, bindings.Services[0].Secrets["some-secret"], "bravo-secret")
			h.AssertEq(t, bindings.Services[0].Secrets["other-secrets"].([]interface{})[0], "bravo-secret-1")
			h.AssertEq(t, bindings.Services[0].Secrets["other-secrets"].([]interface{})[1], "bravo-secret-2")
		})

		type testCase struct {
			requiredKey string
			content     string
		}

		for _, tc := range []testCase{
			{
				"kind", `
[[services]]
  name = "bravo"
  [services.metadata]
# kind = "bravo-kind"
  provider = "bravo-provider"
  tags = ["bravo-tag-1", "bravo-tag-2"]
`,
			},

			{
				"provider", `
[[services]]
  name = "bravo"
  [services.metadata]
  kind = "bravo-kind"
# provider = "bravo-provider"
  tags = ["bravo-tag-1", "bravo-tag-2"]
`,
			},

			{
				"tags", `
[[services]]
  name = "bravo"
  [services.metadata]
  kind = "bravo-kind"
  provider = "bravo-provider"
# tags = ["bravo-tag-1", "bravo-tag-2"]
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
