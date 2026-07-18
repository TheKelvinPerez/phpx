package constraints

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
)

type composerMatchCase struct {
	name       string
	version    string
	constraint string
	expected   bool
}

type composerIntersectionCase struct {
	name     string
	left     string
	right    string
	expected bool
}

type composerNormalizeCase struct {
	name     string
	input    string
	expected string
}

var composerNormalizeCorpus = []composerNormalizeCase{
	{name: "partial version", input: "8.4", expected: "8.4.0.0"},
	{name: "version prefix", input: "v8.4.1", expected: "8.4.1.0"},
	{name: "build metadata", input: "8.4.1+build.7", expected: "8.4.1.0"},
	{name: "beta qualifier", input: "8.4-beta01", expected: "8.4.0.0-beta01"},
	{name: "release candidate", input: "8.4-RC2", expected: "8.4.0.0-RC2"},
	{name: "patch modifier", input: "8.4-p1", expected: "8.4.0.0-patch1"},
	{name: "development modifier", input: "8.4-dev", expected: "8.4.0.0-dev"},
	{name: "leading zero spelling", input: "0008.04.001", expected: "0008.04.001.0"},
	{name: "compound qualifier", input: "8.4.0-beta1.02", expected: "8.4.0.0-beta1.02"},
}

var composerMatchCorpus = []composerMatchCase{
	{name: "exact partial", version: "8.4.0", constraint: "8.4", expected: true},
	{name: "exact partial mismatch", version: "8.4.1", constraint: "8.4", expected: false},
	{name: "version prefix", version: "v8.4.1", constraint: "8.4.1", expected: true},
	{name: "build metadata", version: "8.4.1+build.7", constraint: "8.4.1", expected: true},
	{name: "prerelease mismatch", version: "8.4.0-beta2", constraint: "8.4.0-beta1", expected: false},
	{name: "missing qualifier differs from zero", version: "8.4-beta", constraint: "8.4-beta0", expected: false},
	{name: "missing qualifier sorts first", version: "8.4-beta", constraint: "<8.4-beta0", expected: true},
	{name: "trailing qualifier differs", version: "8.4-beta1", constraint: "8.4-beta1.0", expected: false},
	{name: "qualifier leading zeros compare numerically", version: "8.4-beta1.1", constraint: "8.4-beta1.01", expected: true},
	{name: "greater equal development boundary", version: "8.4.0-beta1", constraint: ">=8.4", expected: true},
	{name: "less development boundary", version: "8.4.0-alpha1", constraint: "<8.4", expected: false},
	{name: "less equal stable boundary", version: "8.4.0", constraint: "<=8.4", expected: true},
	{name: "greater stable boundary", version: "8.4.0-beta1", constraint: ">8.4", expected: false},
	{name: "patch follows stable", version: "8.4.0-p1", constraint: ">8.4", expected: true},
	{name: "not equal", version: "8.4.1", constraint: "!=8.4", expected: true},
	{name: "logical and", version: "8.4.1", constraint: ">=8.2 <8.5", expected: true},
	{name: "logical comma", version: "8.5.0", constraint: ">=8.2, <8.5", expected: false},
	{name: "logical double pipe", version: "9.1.0", constraint: "<8 || >=9", expected: true},
	{name: "logical single pipe", version: "7.4.0", constraint: "<8 | >=9", expected: true},
	{name: "unbounded wildcard", version: "99.0.0", constraint: "*", expected: true},
	{name: "major wildcard", version: "8.9.0", constraint: "8.*", expected: true},
	{name: "minor wildcard upper", version: "8.5.0", constraint: "8.4.x", expected: false},
	{name: "wildcard prerelease", version: "8.4.0-beta1", constraint: "8.4.*", expected: true},
	{name: "partial hyphen upper", version: "8.3.99", constraint: "8.1 - 8.3", expected: true},
	{name: "partial hyphen ceiling", version: "8.4.0", constraint: "8.1 - 8.3", expected: false},
	{name: "full hyphen inclusive", version: "8.3.0", constraint: "8.1.0 - 8.3.0", expected: true},
	{name: "hyphen prerelease upper", version: "8.3.0-beta2", constraint: "8.1.0 - 8.3.0-beta2", expected: true},
	{name: "tilde patch ceiling", version: "8.4.0", constraint: "~8.3.2", expected: false},
	{name: "tilde minor ceiling", version: "8.9.0", constraint: "~8.3", expected: true},
	{name: "tilde zero major", version: "0.9.0", constraint: "~0.3", expected: true},
	{name: "tilde prerelease floor", version: "8.3.2-alpha1", constraint: "~8.3.2-beta1", expected: false},
	{name: "caret major ceiling", version: "9.0.0", constraint: "^8.3", expected: false},
	{name: "caret zero minor", version: "0.3.99", constraint: "^0.3", expected: true},
	{name: "caret zero patch ceiling", version: "0.0.4", constraint: "^0.0.3", expected: false},
	{name: "caret missing minor", version: "0.9.0", constraint: "^0", expected: true},
	{name: "caret fourth component", version: "0.0.0.9", constraint: "^0.0.0.4", expected: true},
	{name: "beta stability floor", version: "8.4.0-alpha1", constraint: ">=8.4@beta", expected: false},
	{name: "beta stability match", version: "8.4.0-beta1", constraint: ">=8.4@beta", expected: true},
	{name: "beta stability exclusive", version: "8.4.0-beta", constraint: ">8.4@beta", expected: false},
	{name: "beta stability not equal", version: "8.4.0-beta", constraint: "!=8.4@beta", expected: false},
	{name: "stable flag floor", version: "8.4.0-RC1", constraint: ">=8.4@stable", expected: true},
	{name: "rc flag ceiling", version: "8.4.0", constraint: "<=8.4@RC", expected: false},
	{name: "range stability flag", version: "8.4.0-beta1", constraint: "^8.4@beta", expected: true},
	{name: "standalone stability flag", version: "99.0.0", constraint: "@beta", expected: true},
	{name: "spaced range stability flag", version: "8.4.1", constraint: "^8.3 @beta", expected: true},
	{name: "spaced wildcard stability flag", version: "8.4.0-beta1", constraint: "8.4.* @beta", expected: true},
	{name: "release candidate ordering", version: "8.4.0-RC2", constraint: ">8.4.0-RC1", expected: true},
	{name: "large numeric component", version: "20264.10.99999", constraint: ">=20264.10 <20265", expected: true},
}

var composerIntersectionCorpus = []composerIntersectionCase{
	{name: "overlapping ranges", left: ">=8 <9", right: "^8.4", expected: true},
	{name: "exclusive shared boundary", left: "<8", right: ">=8", expected: false},
	{name: "inclusive shared range", left: "<=8", right: ">=8", expected: true},
	{name: "excluded exact version", left: "8.4", right: "!=8.4", expected: false},
	{name: "prerelease survives exclusion", left: "8.4-beta", right: "!=8.4", expected: true},
	{name: "overlapping alternative", left: "<8 || >=9", right: "^9.2", expected: true},
	{name: "disjoint alternative", left: "<8 || >=10", right: "^9.2", expected: false},
	{name: "exclusive stability boundary", left: "<8.4@beta", right: ">=8.4@beta", expected: false},
	{name: "inclusive stability boundary", left: "<=8.4@beta", right: ">=8.4@beta", expected: true},
	{name: "wildcard and tilde", left: "8.4.*", right: "~8.4.2", expected: true},
}

var composerDifferentialVersions = []string{
	"0",
	"0-dev",
	"0-alpha",
	"0-alpha0",
	"0-beta",
	"0-RC1",
	"0.0.1",
	"0.1",
	"0.3.99",
	"0.4",
	"1-dev",
	"1-alpha",
	"1-alpha0",
	"1-alpha1",
	"1-beta",
	"1-beta0",
	"1-beta1",
	"1-RC",
	"1-RC1",
	"1",
	"1-p1",
	"1.0.1",
	"1.2-beta",
	"1.2-beta0",
	"1.2-beta1",
	"1.2-RC1",
	"1.2",
	"1.2-p1",
	"1.2.3-dev",
	"1.2.3-alpha1",
	"1.2.3",
	"1.2.3.4",
	"1.2.4",
	"1.3",
	"2",
	"8.4",
	"9",
}

var composerDifferentialConstraints = []string{
	"*",
	"@beta",
	"0",
	"0.0",
	"0.0.0",
	"1.2-beta",
	"1.2-beta0",
	"1.2-beta1.0",
	"!=0",
	"<>1.2",
	">=0",
	"<1",
	"<=1",
	">1",
	">=1.2@alpha",
	"<1.2@beta",
	"!=1.2@RC",
	">=0 <1",
	">=1, <2",
	"<1 || >=2",
	"<1 | >=2",
	"0.*",
	"1.x",
	"1.2.*",
	"v1.*",
	"0 - 1",
	"1.2 - 1.3",
	"1.2.3 - 1.3.0",
	"1.2-beta - 1.3-RC1",
	"~0",
	"~0.3",
	"~1.2",
	"~1.2.3",
	"~1.2.3.4",
	"^0",
	"^0.0",
	"^0.0.3",
	"^0.3",
	"^1.2",
	"^1.2.3.4",
	"^1.2@beta",
	"~1.2 @alpha",
	"1.2.* @dev",
	">1.2-p1",
}

func TestComposerConformance(t *testing.T) {
	t.Run("committed corpus", func(t *testing.T) {
		for _, test := range composerNormalizeCorpus {
			t.Run("normalize "+test.name, func(t *testing.T) {
				version, err := NormalizeVersion(test.input)
				if err != nil {
					t.Fatalf("normalize version: %v", err)
				}
				if actual := version.String(); actual != test.expected {
					t.Fatalf("expected %q, got %q", test.expected, actual)
				}
			})
		}
		for _, test := range composerMatchCorpus {
			t.Run("match "+test.name, func(t *testing.T) {
				actual, err := Satisfies(test.version, test.constraint)
				if err != nil {
					t.Fatalf("evaluate match: %v", err)
				}
				if actual != test.expected {
					t.Fatalf("expected %t, got %t", test.expected, actual)
				}
			})
		}
		for _, test := range composerIntersectionCorpus {
			t.Run("intersection "+test.name, func(t *testing.T) {
				actual, err := Intersects(test.left, test.right)
				if err != nil {
					t.Fatalf("evaluate intersection: %v", err)
				}
				if actual != test.expected {
					t.Fatalf("expected %t, got %t", test.expected, actual)
				}
			})
		}
	})

	t.Run("official composer semver", func(t *testing.T) {
		response, available := runComposerOracle(t)
		if !available {
			return
		}
		if len(response.Normalizations) != len(composerNormalizeCorpus) {
			t.Fatalf(
				"expected %d oracle normalizations, got %d",
				len(composerNormalizeCorpus),
				len(response.Normalizations),
			)
		}
		for index, result := range response.Normalizations {
			test := composerNormalizeCorpus[index]
			if result.Error != "" {
				t.Fatalf("%s oracle error: %s", test.name, result.Error)
			}
			if result.Normalized != test.expected {
				t.Fatalf(
					"%s expected oracle normalization %q, got %q",
					test.name,
					test.expected,
					result.Normalized,
				)
			}
		}
		expectedMatchCount := len(composerMatchCorpus) +
			len(composerDifferentialVersions)*len(composerDifferentialConstraints)
		if len(response.Matches) != expectedMatchCount {
			t.Fatalf(
				"expected %d oracle matches, got %d",
				expectedMatchCount,
				len(response.Matches),
			)
		}
		for index, result := range response.Matches[:len(composerMatchCorpus)] {
			test := composerMatchCorpus[index]
			if result.Error != "" {
				t.Fatalf("%s oracle error: %s", test.name, result.Error)
			}
			if result.Result != test.expected {
				t.Fatalf(
					"%s expected oracle result %t, got %t",
					test.name,
					test.expected,
					result.Result,
				)
			}
		}
		matchOffset := len(composerMatchCorpus)
		for _, constraint := range composerDifferentialConstraints {
			for _, version := range composerDifferentialVersions {
				result := response.Matches[matchOffset]
				matchOffset++
				if result.Error != "" {
					t.Fatalf(
						"oracle error for %q against %q: %s",
						version,
						constraint,
						result.Error,
					)
				}
				actual, err := Satisfies(version, constraint)
				if err != nil {
					t.Fatalf(
						"engine error for %q against %q: %v",
						version,
						constraint,
						err,
					)
				}
				if actual != result.Result {
					t.Fatalf(
						"Composer mismatch for %q against %q, engine %t, oracle %t",
						version,
						constraint,
						actual,
						result.Result,
					)
				}
			}
		}
		expectedIntersectionCount := len(composerIntersectionCorpus) +
			len(composerDifferentialConstraints)*len(composerDifferentialConstraints)
		if len(response.Intersections) != expectedIntersectionCount {
			t.Fatalf(
				"expected %d oracle intersections, got %d",
				expectedIntersectionCount,
				len(response.Intersections),
			)
		}
		for index, result := range response.Intersections[:len(composerIntersectionCorpus)] {
			test := composerIntersectionCorpus[index]
			if result.Error != "" {
				t.Fatalf("%s oracle error: %s", test.name, result.Error)
			}
			if result.Result != test.expected {
				t.Fatalf(
					"%s expected oracle result %t, got %t",
					test.name,
					test.expected,
					result.Result,
				)
			}
		}
		intersectionOffset := len(composerIntersectionCorpus)
		for _, left := range composerDifferentialConstraints {
			for _, right := range composerDifferentialConstraints {
				result := response.Intersections[intersectionOffset]
				intersectionOffset++
				if result.Error != "" {
					t.Fatalf(
						"oracle intersection error for %q and %q: %s",
						left,
						right,
						result.Error,
					)
				}
				actual, err := Intersects(left, right)
				if err != nil {
					t.Fatalf(
						"engine intersection error for %q and %q: %v",
						left,
						right,
						err,
					)
				}
				if actual != result.Result {
					t.Fatalf(
						"Composer intersection mismatch for %q and %q, engine %t, oracle %t",
						left,
						right,
						actual,
						result.Result,
					)
				}
			}
		}
	})
}

type composerOracleRequest struct {
	Normalizations []composerOracleNormalization `json:"normalizations"`
	Matches        []composerOracleMatch         `json:"matches"`
	Intersections  []composerOracleIntersection  `json:"intersections"`
}

type composerOracleNormalization struct {
	Version string `json:"version"`
}

type composerOracleMatch struct {
	Version    string `json:"version"`
	Constraint string `json:"constraint"`
}

type composerOracleIntersection struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

type composerOracleResponse struct {
	Normalizations []composerOracleResult `json:"normalizations"`
	Matches        []composerOracleResult `json:"matches"`
	Intersections  []composerOracleResult `json:"intersections"`
}

type composerOracleResult struct {
	Result     bool   `json:"result"`
	Normalized string `json:"normalized,omitempty"`
	Error      string `json:"error,omitempty"`
}

func runComposerOracle(t *testing.T) (composerOracleResponse, bool) {
	t.Helper()

	phpPath, err := exec.LookPath("php")
	if err != nil {
		t.Skip("PHP is unavailable, skipping the live Composer oracle")
		return composerOracleResponse{}, false
	}
	composerPath, err := exec.LookPath("composer")
	if err != nil {
		t.Skip("Composer is unavailable, skipping the live Composer oracle")
		return composerOracleResponse{}, false
	}

	request := composerOracleRequest{
		Normalizations: make(
			[]composerOracleNormalization,
			0,
			len(composerNormalizeCorpus),
		),
		Matches: make(
			[]composerOracleMatch,
			0,
			len(composerMatchCorpus)+
				len(composerDifferentialVersions)*len(composerDifferentialConstraints),
		),
		Intersections: make(
			[]composerOracleIntersection,
			0,
			len(composerIntersectionCorpus)+
				len(composerDifferentialConstraints)*len(composerDifferentialConstraints),
		),
	}
	for _, test := range composerNormalizeCorpus {
		request.Normalizations = append(
			request.Normalizations,
			composerOracleNormalization{Version: test.input},
		)
	}
	for _, test := range composerMatchCorpus {
		request.Matches = append(request.Matches, composerOracleMatch{
			Version:    test.version,
			Constraint: test.constraint,
		})
	}
	for _, constraint := range composerDifferentialConstraints {
		for _, version := range composerDifferentialVersions {
			request.Matches = append(request.Matches, composerOracleMatch{
				Version:    version,
				Constraint: constraint,
			})
		}
	}
	for _, test := range composerIntersectionCorpus {
		request.Intersections = append(
			request.Intersections,
			composerOracleIntersection{
				Left:  test.left,
				Right: test.right,
			},
		)
	}
	for _, left := range composerDifferentialConstraints {
		for _, right := range composerDifferentialConstraints {
			request.Intersections = append(
				request.Intersections,
				composerOracleIntersection{
					Left:  left,
					Right: right,
				},
			)
		}
	}
	input, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("encode Composer oracle request: %v", err)
	}

	scriptPath := filepath.Join("testdata", "composer_semver.php")
	command := exec.Command(phpPath, scriptPath, composerPath)
	command.Stdin = bytes.NewReader(input)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run Composer oracle: %v\n%s", err, output)
	}

	var response composerOracleResponse
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatalf("decode Composer oracle response: %v\n%s", err, output)
	}

	return response, true
}
