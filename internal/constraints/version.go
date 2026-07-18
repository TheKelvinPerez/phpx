package constraints

import (
	"fmt"
	"regexp"
	"strings"
)

type stability uint8

const (
	stabilityDev stability = iota
	stabilityAlpha
	stabilityBeta
	stabilityRC
	stabilityStable
	stabilityPatch
)

type Version struct {
	components [4]string
	stability  stability
	qualifier  []string
}

func NormalizeVersion(input string) (Version, error) {
	original := input
	input = strings.TrimSpace(input)
	if input == "" {
		return Version{}, newParseError(original, "version is empty")
	}

	if at := strings.LastIndexByte(input, '@'); at >= 0 {
		if !isStabilityName(input[at+1:]) {
			return Version{}, newParseError(original, "version has an invalid stability flag")
		}
		input = input[:at]
	}
	if plus := strings.IndexByte(input, '+'); plus >= 0 {
		if plus == len(input)-1 || strings.ContainsAny(input[plus+1:], " \t\r\n+") {
			return Version{}, newParseError(original, "version has invalid build metadata")
		}
		input = input[:plus]
	}

	matches := numericVersionPattern().FindStringSubmatch(input)
	if matches == nil {
		return Version{}, newParseError(original, "version is not a supported numeric Composer version")
	}

	version := Version{stability: stabilityStable}
	for index := range version.components {
		component := matches[index+1]
		if component == "" {
			component = "0"
		}
		version.components[index] = component
	}

	modifier := strings.ToLower(matches[5])
	devModifier := matches[7] != ""
	if modifier != "" && devModifier {
		return Version{}, newParseError(
			original,
			"compound prerelease and dev modifiers are unsupported for platform versions",
		)
	}
	switch modifier {
	case "", "stable":
		version.stability = stabilityStable
	case "a", "alpha":
		version.stability = stabilityAlpha
	case "b", "beta":
		version.stability = stabilityBeta
	case "rc":
		version.stability = stabilityRC
	case "p", "pl", "patch":
		version.stability = stabilityPatch
	default:
		return Version{}, newParseError(original, "version has an unsupported modifier")
	}
	if devModifier {
		version.stability = stabilityDev
	}
	version.qualifier = normalizeQualifier(matches[6])

	return version, nil
}

func numericVersionPattern() *regexp.Regexp {
	return regexp.MustCompile(
		`(?i)^v?([0-9]{1,5})(?:\.([0-9]+))?(?:\.([0-9]+))?(?:\.([0-9]+))?` +
			`(?:[._-]?(stable|beta|b|rc|alpha|a|patch|pl|p)((?:[.-]?[0-9]+)*))?` +
			`([.-]?dev)?$`,
	)
}

func numericVersionShape(input string) (int, bool, bool) {
	input = strings.TrimSpace(input)
	if plus := strings.IndexByte(input, '+'); plus >= 0 {
		input = input[:plus]
	}
	matches := numericVersionPattern().FindStringSubmatch(input)
	if matches == nil {
		return 0, false, false
	}

	componentCount := 1
	for index := 2; index <= 4; index++ {
		if matches[index] != "" {
			componentCount = index
		}
	}
	explicitModifier := matches[5] != "" || matches[7] != ""

	return componentCount, explicitModifier, true
}

func (version Version) Compare(other Version) int {
	for index := range version.components {
		if comparison := compareDecimal(
			version.components[index],
			other.components[index],
		); comparison != 0 {
			return comparison
		}
	}
	if version.stability < other.stability {
		return -1
	}
	if version.stability > other.stability {
		return 1
	}

	commonLength := min(len(version.qualifier), len(other.qualifier))
	for index := 0; index < commonLength; index++ {
		if comparison := compareDecimal(
			version.qualifier[index],
			other.qualifier[index],
		); comparison != 0 {
			return comparison
		}
	}
	if len(version.qualifier) < len(other.qualifier) {
		return -1
	}
	if len(version.qualifier) > len(other.qualifier) {
		return 1
	}

	return 0
}

func (version Version) String() string {
	normalized := strings.Join(version.components[:], ".")
	switch version.stability {
	case stabilityDev:
		return normalized + "-dev"
	case stabilityAlpha:
		return normalized + "-alpha" + strings.Join(version.qualifier, ".")
	case stabilityBeta:
		return normalized + "-beta" + strings.Join(version.qualifier, ".")
	case stabilityRC:
		return normalized + "-RC" + strings.Join(version.qualifier, ".")
	case stabilityPatch:
		return normalized + "-patch" + strings.Join(version.qualifier, ".")
	default:
		return normalized
	}
}

func normalizeQualifier(input string) []string {
	input = strings.Trim(input, ".-")
	if input == "" {
		return nil
	}

	parts := strings.FieldsFunc(input, func(character rune) bool {
		return character == '.' || character == '-'
	})
	return parts
}

func normalizeDecimal(input string) string {
	normalized := strings.TrimLeft(input, "0")
	if normalized == "" {
		return "0"
	}

	return normalized
}

func compareDecimal(left string, right string) int {
	left = normalizeDecimal(left)
	right = normalizeDecimal(right)
	switch {
	case len(left) < len(right):
		return -1
	case len(left) > len(right):
		return 1
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func decimalComponent(input string) bool {
	if input == "" {
		return false
	}
	for _, character := range input {
		if character < '0' || character > '9' {
			return false
		}
	}

	return true
}

func incrementDecimal(input string) string {
	digits := []byte(normalizeDecimal(input))
	carry := byte(1)
	for index := len(digits) - 1; index >= 0 && carry == 1; index-- {
		if digits[index] == '9' {
			digits[index] = '0'
			continue
		}
		digits[index]++
		carry = 0
	}
	if carry == 1 {
		digits = append([]byte{'1'}, digits...)
	}

	return string(digits)
}

func isStabilityName(input string) bool {
	_, valid := stabilityFromName(input)

	return valid
}

func stabilityFromName(input string) (stability, bool) {
	switch strings.ToLower(input) {
	case "stable":
		return stabilityStable, true
	case "rc":
		return stabilityRC, true
	case "beta":
		return stabilityBeta, true
	case "alpha":
		return stabilityAlpha, true
	case "dev":
		return stabilityDev, true
	default:
		return stabilityStable, false
	}
}

func hasExplicitVersionModifier(input string) bool {
	if at := strings.LastIndexByte(input, '@'); at >= 0 {
		input = input[:at]
	}
	if plus := strings.IndexByte(input, '+'); plus >= 0 {
		input = input[:plus]
	}
	modifierPattern := regexp.MustCompile(
		`(?i)(?:[._-]?(?:stable|beta|b|rc|alpha|a|patch|pl|p)` +
			`(?:[.-]?[0-9]+)*|[.-]?dev)$`,
	)

	return modifierPattern.MatchString(input)
}

type ParseError struct {
	Input  string
	Reason string
}

func newParseError(input string, reason string) *ParseError {
	return &ParseError{
		Input:  input,
		Reason: reason,
	}
}

func (parseError *ParseError) Error() string {
	return fmt.Sprintf(
		"could not parse Composer value %q: %s",
		parseError.Input,
		parseError.Reason,
	)
}
