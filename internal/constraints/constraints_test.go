package constraints

import (
	"errors"
	"reflect"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
)

func TestNormalizeComposerVersions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "8.4", expected: "8.4.0.0"},
		{input: "v8.4.1", expected: "8.4.1.0"},
		{input: "8.4.1+build.7", expected: "8.4.1.0"},
		{input: "8.4-beta01", expected: "8.4.0.0-beta01"},
		{input: "8.4-RC2", expected: "8.4.0.0-RC2"},
		{input: "8.4-p1", expected: "8.4.0.0-patch1"},
		{input: "8.4-dev", expected: "8.4.0.0-dev"},
		{input: "0008.04.001", expected: "0008.04.001.0"},
		{input: "8.4.0-beta1.02", expected: "8.4.0.0-beta1.02"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			version, err := NormalizeVersion(test.input)
			if err != nil {
				t.Fatalf("normalize Composer version: %v", err)
			}
			if actual := version.String(); actual != test.expected {
				t.Fatalf("expected %q, got %q", test.expected, actual)
			}
		})
	}
}

func TestSatisfiesExactComposerVersions(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		constraint string
		expected   bool
	}{
		{
			name:       "missing version component",
			version:    "8.4",
			constraint: "8.4.0",
			expected:   true,
		},
		{
			name:       "v prefix",
			version:    "v8.4.1",
			constraint: "8.4.1",
			expected:   true,
		},
		{
			name:       "build metadata",
			version:    "8.4.1+build.7",
			constraint: "8.4.1",
			expected:   true,
		},
		{
			name:       "exact partial version",
			version:    "8.4.1",
			constraint: "8.4",
			expected:   false,
		},
		{
			name:       "matching prerelease",
			version:    "8.4.0-beta1",
			constraint: "8.4.0-beta1",
			expected:   true,
		},
		{
			name:       "different prerelease",
			version:    "8.4.0-beta2",
			constraint: "8.4.0-beta1",
			expected:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := Satisfies(test.version, test.constraint)
			if err != nil {
				t.Fatalf("evaluate exact constraint: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}

func TestSatisfiesComparisonConstraints(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		expected   bool
	}{
		{version: "8.4.1", constraint: ">=8.4", expected: true},
		{version: "8.3.9", constraint: ">=8.4", expected: false},
		{version: "8.4.0-beta1", constraint: ">=8.4", expected: true},
		{version: "8.4.0-beta1", constraint: ">8.4", expected: false},
		{version: "8.4.0-alpha1", constraint: "<8.4", expected: false},
		{version: "8.3.99", constraint: "<8.4", expected: true},
		{version: "8.4.0", constraint: "<= 8.4", expected: true},
		{version: "8.4.1", constraint: "!=8.4", expected: true},
		{version: "8.4.0", constraint: "<>8.4", expected: false},
	}

	for _, test := range tests {
		t.Run(test.version+" "+test.constraint, func(t *testing.T) {
			actual, err := Satisfies(test.version, test.constraint)
			if err != nil {
				t.Fatalf("evaluate comparison constraint: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}

func TestSatisfiesLogicalConstraints(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		expected   bool
	}{
		{version: "8.4.1", constraint: ">=8.2 <8.5", expected: true},
		{version: "8.5.0", constraint: ">=8.2 <8.5", expected: false},
		{version: "8.4.1", constraint: ">=8.2, <8.5", expected: true},
		{version: "9.1.0", constraint: ">=8.2 <8.5 || >=9.0", expected: true},
		{version: "8.5.0", constraint: ">=8.2 <8.5 || >=9.0", expected: false},
		{version: "7.4.0", constraint: "<8.0 | >=9.0", expected: true},
		{
			version:    "8.4.0",
			constraint: ">=8.0 <8.5 || >=9.0 <10.0",
			expected:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.version+" "+test.constraint, func(t *testing.T) {
			actual, err := Satisfies(test.version, test.constraint)
			if err != nil {
				t.Fatalf("evaluate logical constraint: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}

func TestSatisfiesWildcardConstraints(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		expected   bool
	}{
		{version: "99.0.0", constraint: "*", expected: true},
		{version: "8.9.0", constraint: "8.*", expected: true},
		{version: "9.0.0", constraint: "8.*", expected: false},
		{version: "8.4.7", constraint: "8.4.x", expected: true},
		{version: "8.5.0", constraint: "8.4.*", expected: false},
		{version: "8.4.0-beta1", constraint: "8.4.*", expected: true},
		{version: "0.9.0", constraint: "0.*", expected: true},
		{version: "1.0.0", constraint: "0.*", expected: false},
		{version: "8.2.0", constraint: "v8.*", expected: true},
	}

	for _, test := range tests {
		t.Run(test.version+" "+test.constraint, func(t *testing.T) {
			actual, err := Satisfies(test.version, test.constraint)
			if err != nil {
				t.Fatalf("evaluate wildcard constraint: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}

func TestSatisfiesHyphenRangeConstraints(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		expected   bool
	}{
		{version: "8.3.0", constraint: "8.1.0 - 8.3.0", expected: true},
		{version: "8.3.1", constraint: "8.1.0 - 8.3.0", expected: false},
		{version: "8.3.99", constraint: "8.1 - 8.3", expected: true},
		{version: "8.4.0", constraint: "8.1 - 8.3", expected: false},
		{version: "8.99.0", constraint: "7 - 8", expected: true},
		{version: "9.0.0", constraint: "7 - 8", expected: false},
		{version: "8.1.0-beta1", constraint: "8.1 - 8.3", expected: true},
		{
			version:    "8.3.0-beta2",
			constraint: "8.1.0 - 8.3.0-beta2",
			expected:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.version+" "+test.constraint, func(t *testing.T) {
			actual, err := Satisfies(test.version, test.constraint)
			if err != nil {
				t.Fatalf("evaluate hyphen range: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}

func TestSatisfiesTildeConstraints(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		expected   bool
	}{
		{version: "8.3.99", constraint: "~8.3.2", expected: true},
		{version: "8.4.0", constraint: "~8.3.2", expected: false},
		{version: "8.9.0", constraint: "~8.3", expected: true},
		{version: "9.0.0", constraint: "~8.3", expected: false},
		{version: "8.3.99", constraint: "~8.3.0", expected: true},
		{version: "8.4.0", constraint: "~8.3.0", expected: false},
		{version: "0.9.0", constraint: "~0.3", expected: true},
		{version: "1.0.0", constraint: "~0.3", expected: false},
		{version: "8.3.2-beta1", constraint: "~8.3.2-beta1", expected: true},
		{version: "8.3.2-alpha1", constraint: "~8.3.2-beta1", expected: false},
	}

	for _, test := range tests {
		t.Run(test.version+" "+test.constraint, func(t *testing.T) {
			actual, err := Satisfies(test.version, test.constraint)
			if err != nil {
				t.Fatalf("evaluate tilde constraint: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}

func TestSatisfiesCaretConstraints(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		expected   bool
	}{
		{version: "8.9.0", constraint: "^8.3", expected: true},
		{version: "9.0.0", constraint: "^8.3", expected: false},
		{version: "0.3.99", constraint: "^0.3", expected: true},
		{version: "0.4.0", constraint: "^0.3", expected: false},
		{version: "0.0.3", constraint: "^0.0.3", expected: true},
		{version: "0.0.4", constraint: "^0.0.3", expected: false},
		{version: "0.0.9", constraint: "^0.0", expected: true},
		{version: "0.1.0", constraint: "^0.0", expected: false},
		{version: "0.9.0", constraint: "^0", expected: true},
		{version: "1.0.0", constraint: "^0", expected: false},
		{version: "8.3.2-beta1", constraint: "^8.3.2-beta1", expected: true},
		{version: "8.3.2-alpha1", constraint: "^8.3.2-beta1", expected: false},
	}

	for _, test := range tests {
		t.Run(test.version+" "+test.constraint, func(t *testing.T) {
			actual, err := Satisfies(test.version, test.constraint)
			if err != nil {
				t.Fatalf("evaluate caret constraint: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}

func TestSatisfiesStabilityConstraints(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		expected   bool
	}{
		{version: "8.4.0-alpha1", constraint: ">=8.4@beta", expected: false},
		{version: "8.4.0-beta", constraint: ">=8.4@beta", expected: true},
		{version: "8.4.0-beta1", constraint: ">=8.4@beta", expected: true},
		{version: "8.4.0-RC1", constraint: ">=8.4@beta", expected: true},
		{version: "8.4.0", constraint: ">=8.4@beta", expected: true},
		{version: "8.4.0-beta", constraint: ">8.4@beta", expected: false},
		{version: "8.4.0-beta1", constraint: ">8.4@beta", expected: true},
		{version: "8.4.0-alpha1", constraint: "<=8.4@beta", expected: true},
		{version: "8.4.0-beta", constraint: "<=8.4@beta", expected: true},
		{version: "8.4.0-beta1", constraint: "<=8.4@beta", expected: false},
		{version: "8.4.0", constraint: "8.4@beta", expected: true},
		{version: "8.4.0-beta1", constraint: "8.4@beta", expected: false},
		{version: "8.4.0-beta", constraint: "!=8.4@beta", expected: false},
		{version: "8.4.0", constraint: "!=8.4@beta", expected: true},
		{version: "8.4.0-beta1", constraint: "^8.4@beta", expected: true},
		{version: "8.4.0-dev", constraint: ">=8.4@dev", expected: true},
		{version: "8.4.0-RC1", constraint: ">=8.4@stable", expected: true},
		{version: "99.0.0", constraint: "@beta", expected: true},
		{version: "8.4.1", constraint: "^8.3 @beta", expected: true},
		{version: "8.4.0-beta1", constraint: "8.4.* @beta", expected: true},
	}

	for _, test := range tests {
		t.Run(test.version+" "+test.constraint, func(t *testing.T) {
			actual, err := Satisfies(test.version, test.constraint)
			if err != nil {
				t.Fatalf("evaluate stability constraint: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}

func TestIntersectsComposerConstraints(t *testing.T) {
	tests := []struct {
		left     string
		right    string
		expected bool
	}{
		{left: ">=8 <9", right: "^8.4", expected: true},
		{left: "<8", right: ">=8", expected: false},
		{left: "<=8", right: ">=8", expected: true},
		{left: "8.4", right: "!=8.4", expected: false},
		{left: "8.4-beta", right: "!=8.4", expected: true},
		{left: "<8 || >=9", right: "^9.2", expected: true},
		{left: "<8 || >=10", right: "^9.2", expected: false},
		{left: "<8.4@beta", right: ">=8.4@beta", expected: false},
		{left: "<=8.4@beta", right: ">=8.4@beta", expected: true},
		{left: "8.4.*", right: "~8.4.2", expected: true},
	}

	for _, test := range tests {
		t.Run(test.left+" intersects "+test.right, func(t *testing.T) {
			actual, err := Intersects(test.left, test.right)
			if err != nil {
				t.Fatalf("evaluate constraint intersection: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}

func TestEvaluateRequirementSupportsEveryPlatformKind(t *testing.T) {
	kinds := []model.RequirementKind{
		model.RequirementPHP,
		model.RequirementPHPSubtype,
		model.RequirementExtension,
		model.RequirementSystemLibrary,
		model.RequirementComposer,
		model.RequirementComposerPluginAPI,
		model.RequirementComposerRuntimeAPI,
	}

	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			matches, err := EvaluateRequirement(model.Requirement{
				Name:       string(kind),
				Kind:       kind,
				Constraint: "^8.3",
			}, "8.4.1")
			if err != nil {
				t.Fatalf("evaluate %s requirement: %v", kind, err)
			}
			if !matches {
				t.Fatalf("expected %s requirement to match", kind)
			}
		})
	}
}

func TestParseRejectsUnsupportedConstraintSyntax(t *testing.T) {
	tests := []string{
		"",
		"dev-main",
		"1.0 as 2.0",
		"~>8.3",
		"(>=8.2 <9.0)",
		">=8.2 ||",
		"8.*.1",
		"^8.3@preview",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input)
			if err == nil {
				t.Fatal("expected unsupported constraint to fail")
			}
			var parseError *ParseError
			if !errors.As(err, &parseError) {
				t.Fatalf("expected ParseError, got %T", err)
			}
			if parseError.Input != input {
				t.Fatalf(
					"expected original input %q, got %q",
					input,
					parseError.Input,
				)
			}
		})
	}
}

func TestEvaluateRequirementPreservesSourcesOnUnsupportedSyntax(t *testing.T) {
	sources := []model.SourceReference{
		{
			Path:  "/workspace/composer.json",
			Kind:  "composer_manifest",
			Field: "/require/php",
			Line:  12,
		},
		{
			Path:  "/workspace/composer.lock",
			Kind:  "composer_lock",
			Field: "/packages/3/require/php",
		},
	}
	requirement := model.Requirement{
		Name:       "php",
		Kind:       model.RequirementPHP,
		Constraint: "dev-main",
		Sources:    sources,
	}

	matches, err := EvaluateRequirement(requirement, "8.4.1")
	if matches {
		t.Fatal("expected unsupported constraint not to match")
	}
	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected model.Error, got %T", err)
	}
	if commandError.Code != model.ErrorRequirements {
		t.Fatalf("expected requirements error, got %s", commandError.Code)
	}
	if !reflect.DeepEqual(commandError.Sources, sources) {
		t.Fatalf(
			"expected original sources %#v, got %#v",
			sources,
			commandError.Sources,
		)
	}
	var parseError *ParseError
	if !errors.As(err, &parseError) {
		t.Fatalf("expected wrapped ParseError, got %T", err)
	}
	if parseError.Input != requirement.Constraint {
		t.Fatalf(
			"expected original constraint %q, got %q",
			requirement.Constraint,
			parseError.Input,
		)
	}
}

func TestEvaluateRequirementRejectsNonPlatformKind(t *testing.T) {
	requirement := model.Requirement{
		Name:       "vendor/package",
		Kind:       model.RequirementKind("package"),
		Constraint: "^1.0",
		Sources: []model.SourceReference{
			{Path: "/workspace/composer.json", Field: "/require/vendor~1package"},
		},
	}

	_, err := EvaluateRequirement(requirement, "1.2.0")
	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected model.Error, got %T", err)
	}
	if commandError.Code != model.ErrorRequirements {
		t.Fatalf("expected requirements error, got %s", commandError.Code)
	}
	if !reflect.DeepEqual(commandError.Sources, requirement.Sources) {
		t.Fatalf("expected original sources, got %#v", commandError.Sources)
	}
}
