/*
Copyright (c) Advanced Micro Devices, Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the \"License\");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an \"AS IS\" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package charts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mustNot fails the test immediately when err is non-nil. The repository
// vendors only testify's assert package, so hard failures use t.Fatalf.
func mustNot(t *testing.T, err error, format string, args ...interface{}) {
	t.Helper()
	if err != nil {
		t.Fatalf("%v: %v", fmt.Sprintf(format, args...), err)
	}
}

const (
	// the NodeFeatureRule template under test, relative to the repository root
	nfdRuleTemplate = "helm-charts-k8s/templates/gpu-nfd-default-rule.yaml"
	// the same template is the source of truth patched over the generated chart
	// by the helm-k8s make target, so the two copies must not drift
	nfdRulePatchCopy = "hack/k8s-patch/template-patch/gpu-nfd-default-rule.yaml"
)

// ruleNameRegexp matches a rule heading, e.g. "  - name: amd-gpu"
var ruleNameRegexp = regexp.MustCompile(`(?m)^ {2}- name: (\S+)`)

// deviceIDRegexp matches a rendered PCI device match expression.
var deviceIDRegexp = regexp.MustCompile(`device: \{op: In, value: \["([0-9a-fA-F]+)"\]\}`)

// repoRoot walks up from the test's working directory to the repository root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	mustNot(t, err, "read working directory")
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatalf("could not locate repository root from working directory")
	return ""
}

// renderNFDRule renders the NodeFeatureRule template on its own, in a throwaway
// chart, so that the assertions do not depend on the operator chart's subchart
// dependencies being vendored or fetchable.
func renderNFDRule(t *testing.T, values string, setArgs ...string) string {
	t.Helper()

	helm, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("helm is not installed, skipping chart render test")
	}

	root := repoRoot(t)
	chartDir := t.TempDir()
	templateDir := filepath.Join(chartDir, "templates")
	mustNot(t, os.MkdirAll(templateDir, 0o755), "create template directory")

	chartYaml := "apiVersion: v2\nname: gpu-nfd-rule-under-test\nversion: 0.0.0\n"
	mustNot(t, os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartYaml), 0o644), "write Chart.yaml")
	mustNot(t, os.WriteFile(filepath.Join(chartDir, "values.yaml"), []byte(values), 0o644), "write values.yaml")

	template, err := os.ReadFile(filepath.Join(root, nfdRuleTemplate))
	mustNot(t, err, "read %v", nfdRuleTemplate)
	mustNot(t, os.WriteFile(filepath.Join(templateDir, "rule.yaml"), template, 0o644), "write template under test")

	args := append([]string{"template", "under-test", chartDir}, setArgs...)
	out, err := exec.Command(helm, args...).CombinedOutput()
	mustNot(t, err, "helm template failed: %s", string(out))
	return string(out)
}

// deviceIDsByRule maps each rule name in the rendered output to the PCI device
// IDs matched under it.
func deviceIDsByRule(rendered string) map[string][]string {
	byRule := map[string][]string{}
	current := ""
	for _, line := range regexp.MustCompile(`\r?\n`).Split(rendered, -1) {
		if m := ruleNameRegexp.FindStringSubmatch(line); m != nil {
			current = m[1]
			if _, ok := byRule[current]; !ok {
				byRule[current] = []string{}
			}
			continue
		}
		if m := deviceIDRegexp.FindStringSubmatch(line); m != nil && current != "" {
			byRule[current] = append(byRule[current], m[1])
		}
	}
	return byRule
}

const baseValues = `installdefaultNFDRule: true
additionalGPUDeviceIDs: []
additionalVGPUDeviceIDs: []
`

// TestAdditionalDeviceIDsAreAppendedToTheCorrectRule verifies that device IDs
// supplied through values reach the rule they belong to, and only that rule.
func TestAdditionalDeviceIDsAreAppendedToTheCorrectRule(t *testing.T) {
	testCases := []struct {
		name            string
		setArgs         []string
		expectInGPU     []string
		expectInVGPU    []string
		expectNotInGPU  []string
		expectNotInVGPU []string
	}{
		{
			name:            "a single additional GPU device ID",
			setArgs:         []string{"--set", "additionalGPUDeviceIDs={1586}"},
			expectInGPU:     []string{"1586"},
			expectNotInVGPU: []string{"1586"},
		},
		{
			name:            "several additional GPU device IDs",
			setArgs:         []string{"--set", "additionalGPUDeviceIDs={1586,150e}"},
			expectInGPU:     []string{"1586", "150e"},
			expectNotInVGPU: []string{"1586", "150e"},
		},
		{
			name:           "an additional virtual GPU device ID",
			setArgs:        []string{"--set", "additionalVGPUDeviceIDs={74b5}"},
			expectInVGPU:   []string{"74b5"},
			expectNotInGPU: []string{"74b5"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			byRule := deviceIDsByRule(renderNFDRule(t, baseValues, tc.setArgs...))

			for _, id := range tc.expectInGPU {
				assert.Contains(t, byRule["amd-gpu"], id,
					"expected device ID %v to be matched by the amd-gpu rule", id)
			}
			for _, id := range tc.expectInVGPU {
				assert.Contains(t, byRule["amd-vgpu"], id,
					"expected device ID %v to be matched by the amd-vgpu rule", id)
			}
			for _, id := range tc.expectNotInGPU {
				assert.NotContains(t, byRule["amd-gpu"], id,
					"device ID %v must not leak into the amd-gpu rule", id)
			}
			for _, id := range tc.expectNotInVGPU {
				assert.NotContains(t, byRule["amd-vgpu"], id,
					"device ID %v must not leak into the amd-vgpu rule", id)
			}

			// the narrow per-product rules take no user input
			assert.Equal(t, []string{"740f"}, byRule["amd-gpu-mi210"])
			assert.Equal(t, []string{"74a1"}, byRule["amd-gpu-mi300x"])
		})
	}
}

// TestEmptyAdditionalDeviceIDsChangeNothing verifies the default values render
// exactly the device set the chart shipped before the option existed, so that
// enabling the option is the only way to change behaviour.
func TestEmptyAdditionalDeviceIDsChangeNothing(t *testing.T) {
	withOption := deviceIDsByRule(renderNFDRule(t, baseValues))
	// values that predate the option: the template must tolerate their absence
	withoutOption := deviceIDsByRule(renderNFDRule(t, "installdefaultNFDRule: true\n"))

	assert.Equal(t, withoutOption, withOption,
		"empty additional device ID lists must render the same rules as no lists at all")

	for _, rule := range []string{"amd-gpu", "amd-vgpu"} {
		assert.NotEmpty(t, withOption[rule], "expected the %v rule to be rendered", rule)
	}
}

// TestNFDRulePatchCopyMatchesChart guards the copy under hack/k8s-patch, which
// the helm-k8s make target copies over the generated chart. If the two drift,
// a change made only to the chart is silently discarded by the next build.
func TestNFDRulePatchCopyMatchesChart(t *testing.T) {
	root := repoRoot(t)

	chartCopy, err := os.ReadFile(filepath.Join(root, nfdRuleTemplate))
	mustNot(t, err, "read %v", nfdRuleTemplate)
	patchCopy, err := os.ReadFile(filepath.Join(root, nfdRulePatchCopy))
	mustNot(t, err, "read %v", nfdRulePatchCopy)

	assert.Equal(t, string(patchCopy), string(chartCopy),
		fmt.Sprintf("%v and %v must stay identical: the helm-k8s target copies the "+
			"latter over the former, so a change to only one of them is lost",
			nfdRuleTemplate, nfdRulePatchCopy))
}
