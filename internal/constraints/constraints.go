package constraints

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/elefantephp/elefante/internal/model"
)

type operator uint8

const (
	operatorEqual operator = iota
	operatorNotEqual
	operatorLess
	operatorLessEqual
	operatorGreater
	operatorGreaterEqual
)

type predicate struct {
	operator operator
	version  Version
}

type constraintGroup struct {
	predicates []predicate
}

type constraintBound struct {
	version   Version
	inclusive bool
	set       bool
}

type Constraint struct {
	groups []constraintGroup
}

func Parse(input string) (Constraint, error) {
	original := input
	input = strings.TrimSpace(input)
	if input == "" {
		return Constraint{}, newParseError(original, "constraint is empty")
	}

	orPattern := regexp.MustCompile(`\s*\|\|?\s*`)
	rawGroups := orPattern.Split(input, -1)
	groups := make([]constraintGroup, 0, len(rawGroups))
	for _, rawGroup := range rawGroups {
		if strings.TrimSpace(rawGroup) == "" {
			return Constraint{}, newParseError(original, "constraint has an empty OR group")
		}
		atoms, err := splitANDConstraints(rawGroup)
		if err != nil {
			return Constraint{}, newParseError(original, err.Reason)
		}
		group := constraintGroup{}
		for _, atom := range atoms {
			parsed, err := parseAtomicConstraint(atom)
			if err != nil {
				return Constraint{}, newParseError(original, err.Reason)
			}
			group.predicates = append(group.predicates, parsed...)
		}
		groups = append(groups, group)
	}

	return Constraint{groups: groups}, nil
}

func parseAtomicConstraint(input string) ([]predicate, *ParseError) {
	versionInput := strings.TrimSpace(input)
	if strings.HasPrefix(versionInput, "@") {
		if !isStabilityName(versionInput[1:]) {
			return nil, newParseError(input, "constraint has an invalid stability flag")
		}

		return nil, nil
	}
	if rangePredicates, matched, err := parseCaretConstraint(versionInput); matched {
		return rangePredicates, err
	}
	if rangePredicates, matched, err := parseTildeConstraint(versionInput); matched {
		return rangePredicates, err
	}
	if rangePredicates, matched, err := parseHyphenRange(versionInput); matched {
		return rangePredicates, err
	}
	if wildcardPredicates, matched, err := parseWildcardConstraint(versionInput); matched {
		return wildcardPredicates, err
	}

	predicateOperator := operatorEqual
	operators := []struct {
		prefix   string
		operator operator
	}{
		{prefix: "<>", operator: operatorNotEqual},
		{prefix: "!=", operator: operatorNotEqual},
		{prefix: ">=", operator: operatorGreaterEqual},
		{prefix: "<=", operator: operatorLessEqual},
		{prefix: "==", operator: operatorEqual},
		{prefix: "=", operator: operatorEqual},
		{prefix: ">", operator: operatorGreater},
		{prefix: "<", operator: operatorLess},
	}
	for _, candidate := range operators {
		if strings.HasPrefix(versionInput, candidate.prefix) {
			predicateOperator = candidate.operator
			versionInput = strings.TrimSpace(versionInput[len(candidate.prefix):])
			break
		}
	}
	versionInput, flaggedStability, hasStabilityFlag, flagErr := splitStabilityFlag(
		versionInput,
	)
	if flagErr != nil {
		return nil, newParseError(input, "constraint has an invalid stability flag")
	}
	version, err := NormalizeVersion(versionInput)
	if err != nil {
		return nil, newParseError(input, "constraint version is invalid")
	}
	if hasStabilityFlag &&
		flaggedStability != stabilityStable &&
		predicateOperator != operatorEqual &&
		version.stability == stabilityStable {
		version.stability = flaggedStability
	} else if (predicateOperator == operatorLess ||
		predicateOperator == operatorGreaterEqual) &&
		!hasExplicitVersionModifier(versionInput) {
		version.stability = stabilityDev
		version.qualifier = nil
	}

	return []predicate{
		{
			operator: predicateOperator,
			version:  version,
		},
	}, nil
}

func parseCaretConstraint(input string) ([]predicate, bool, *ParseError) {
	if !strings.HasPrefix(input, "^") {
		return nil, false, nil
	}

	versionInput, err := stripStabilityFlag(strings.TrimSpace(input[1:]))
	if err != nil {
		return nil, true, newParseError(input, "caret constraint has an invalid stability flag")
	}
	componentCount, explicitModifier, valid := numericVersionShape(versionInput)
	if !valid {
		return nil, true, newParseError(input, "caret constraint has an invalid version")
	}
	lower, normalizeErr := NormalizeVersion(versionInput)
	if normalizeErr != nil {
		return nil, true, newParseError(input, "caret constraint has an invalid version")
	}
	if !explicitModifier {
		lower.stability = stabilityDev
		lower.qualifier = nil
	}

	upperPosition := 0
	if compareDecimal(lower.components[0], "0") == 0 && componentCount > 1 {
		upperPosition = 1
		if compareDecimal(lower.components[1], "0") == 0 && componentCount > 2 {
			upperPosition = 2
		}
	}
	upper := incrementVersionComponent(lower, upperPosition)

	return []predicate{
		{operator: operatorGreaterEqual, version: lower},
		{operator: operatorLess, version: upper},
	}, true, nil
}

func parseTildeConstraint(input string) ([]predicate, bool, *ParseError) {
	if !strings.HasPrefix(input, "~") {
		return nil, false, nil
	}
	if strings.HasPrefix(input, "~>") {
		return nil, true, newParseError(input, "pessimistic constraints are unsupported")
	}

	versionInput, err := stripStabilityFlag(strings.TrimSpace(input[1:]))
	if err != nil {
		return nil, true, newParseError(input, "tilde constraint has an invalid stability flag")
	}
	componentCount, explicitModifier, valid := numericVersionShape(versionInput)
	if !valid {
		return nil, true, newParseError(input, "tilde constraint has an invalid version")
	}
	lower, normalizeErr := NormalizeVersion(versionInput)
	if normalizeErr != nil {
		return nil, true, newParseError(input, "tilde constraint has an invalid version")
	}
	if !explicitModifier {
		lower.stability = stabilityDev
		lower.qualifier = nil
	}

	upperPosition := max(1, componentCount-1) - 1
	upper := incrementVersionComponent(lower, upperPosition)

	return []predicate{
		{operator: operatorGreaterEqual, version: lower},
		{operator: operatorLess, version: upper},
	}, true, nil
}

func parseHyphenRange(input string) ([]predicate, bool, *ParseError) {
	if !strings.Contains(input, " - ") {
		return nil, false, nil
	}
	endpoints := strings.Split(input, " - ")
	if len(endpoints) != 2 ||
		strings.TrimSpace(endpoints[0]) == "" ||
		strings.TrimSpace(endpoints[1]) == "" {
		return nil, true, newParseError(input, "hyphen range must have two endpoints")
	}

	lowerCount, lowerExplicit, lowerValid := numericVersionShape(endpoints[0])
	if !lowerValid || lowerCount == 0 {
		return nil, true, newParseError(input, "hyphen range has an invalid lower endpoint")
	}
	lower, err := NormalizeVersion(endpoints[0])
	if err != nil {
		return nil, true, newParseError(input, "hyphen range has an invalid lower endpoint")
	}
	if !lowerExplicit {
		lower.stability = stabilityDev
		lower.qualifier = nil
	}

	upperCount, upperExplicit, upperValid := numericVersionShape(endpoints[1])
	if !upperValid || upperCount == 0 {
		return nil, true, newParseError(input, "hyphen range has an invalid upper endpoint")
	}
	upper, err := NormalizeVersion(endpoints[1])
	if err != nil {
		return nil, true, newParseError(input, "hyphen range has an invalid upper endpoint")
	}
	upperOperator := operatorLessEqual
	if upperCount < 3 && !upperExplicit {
		upper = incrementVersionComponent(upper, upperCount-1)
		upperOperator = operatorLess
	}

	return []predicate{
		{operator: operatorGreaterEqual, version: lower},
		{operator: upperOperator, version: upper},
	}, true, nil
}

func parseWildcardConstraint(input string) ([]predicate, bool, *ParseError) {
	if at := strings.LastIndexByte(input, '@'); at >= 0 {
		if !isStabilityName(input[at+1:]) {
			return nil, true, newParseError(
				input,
				"wildcard constraint has an invalid stability flag",
			)
		}
		input = input[:at]
	}
	input = strings.TrimSpace(input)
	if strings.HasPrefix(strings.ToLower(input), "v") {
		input = input[1:]
	}

	parts := strings.Split(input, ".")
	if len(parts) == 0 {
		return nil, false, nil
	}
	firstWildcard := -1
	for index, part := range parts {
		if part == "*" || strings.EqualFold(part, "x") {
			if firstWildcard < 0 {
				firstWildcard = index
			}
			continue
		}
		if firstWildcard >= 0 {
			return nil, false, nil
		}
		if !decimalComponent(part) {
			return nil, false, nil
		}
	}
	if firstWildcard < 0 {
		return nil, false, nil
	}
	if firstWildcard == 0 {
		return nil, true, nil
	}
	if firstWildcard > 3 {
		return nil, true, newParseError(
			input,
			"wildcard constraint has too many numeric components",
		)
	}

	lower := Version{stability: stabilityDev}
	for index := range lower.components {
		if index < firstWildcard {
			lower.components[index] = parts[index]
		} else {
			lower.components[index] = "0"
		}
	}
	upper := lower
	upper.components[firstWildcard-1] = incrementDecimal(
		upper.components[firstWildcard-1],
	)

	return []predicate{
		{operator: operatorGreaterEqual, version: lower},
		{operator: operatorLess, version: upper},
	}, true, nil
}

func incrementVersionComponent(version Version, position int) Version {
	version.components[position] = incrementDecimal(version.components[position])
	for index := position + 1; index < len(version.components); index++ {
		version.components[index] = "0"
	}
	version.stability = stabilityDev
	version.qualifier = nil

	return version
}

func stripStabilityFlag(input string) (string, *ParseError) {
	stripped, _, _, err := splitStabilityFlag(input)

	return stripped, err
}

func splitStabilityFlag(
	input string,
) (string, stability, bool, *ParseError) {
	if at := strings.LastIndexByte(input, '@'); at >= 0 {
		if strings.Contains(input[:at], "@") {
			return "", stabilityStable, false, newParseError(
				input,
				"multiple stability flags are invalid",
			)
		}
		flaggedStability, valid := stabilityFromName(input[at+1:])
		if !valid {
			return "", stabilityStable, false, newParseError(
				input,
				"invalid stability flag",
			)
		}

		return strings.TrimSpace(input[:at]), flaggedStability, true, nil
	}

	return strings.TrimSpace(input), stabilityStable, false, nil
}

func Satisfies(version string, constraint string) (bool, error) {
	normalizedVersion, err := NormalizeVersion(version)
	if err != nil {
		return false, err
	}
	parsedConstraint, err := Parse(constraint)
	if err != nil {
		return false, err
	}

	return parsedConstraint.Matches(normalizedVersion), nil
}

func EvaluateRequirement(
	requirement model.Requirement,
	version string,
) (bool, error) {
	if !isPlatformRequirementKind(requirement.Kind) {
		cause := fmt.Errorf(
			"requirement kind %q is not a Composer platform package",
			requirement.Kind,
		)

		return false, requirementEvaluationError(requirement, cause)
	}

	matches, err := Satisfies(version, requirement.Constraint)
	if err != nil {
		return false, requirementEvaluationError(requirement, err)
	}

	return matches, nil
}

func isPlatformRequirementKind(kind model.RequirementKind) bool {
	switch kind {
	case model.RequirementPHP,
		model.RequirementPHPSubtype,
		model.RequirementExtension,
		model.RequirementSystemLibrary,
		model.RequirementComposer,
		model.RequirementComposerPluginAPI,
		model.RequirementComposerRuntimeAPI:
		return true
	default:
		return false
	}
}

func requirementEvaluationError(
	requirement model.Requirement,
	cause error,
) *model.Error {
	commandError := model.WrapError(
		model.ErrorRequirements,
		fmt.Sprintf(
			"Cannot evaluate Composer platform requirement %q.",
			requirement.Name,
		),
		cause,
	)
	commandError.Detail = cause.Error()
	commandError.Hint = "Use a supported Composer platform version constraint."
	commandError.Sources = append(
		[]model.SourceReference(nil),
		requirement.Sources...,
	)

	return commandError
}

func Intersects(left string, right string) (bool, error) {
	leftConstraint, err := Parse(left)
	if err != nil {
		return false, err
	}
	rightConstraint, err := Parse(right)
	if err != nil {
		return false, err
	}

	for _, leftGroup := range leftConstraint.groups {
		for _, rightGroup := range rightConstraint.groups {
			predicates := make(
				[]predicate,
				0,
				len(leftGroup.predicates)+len(rightGroup.predicates),
			)
			predicates = append(predicates, leftGroup.predicates...)
			predicates = append(predicates, rightGroup.predicates...)
			if predicatesHaveSolution(predicates) {
				return true, nil
			}
		}
	}

	return false, nil
}

func predicatesHaveSolution(predicates []predicate) bool {
	for _, candidate := range predicates {
		if candidate.operator != operatorEqual {
			continue
		}
		for _, requirement := range predicates {
			comparison := candidate.version.Compare(requirement.version)
			if !predicateMatches(requirement.operator, comparison) {
				return false
			}
		}

		return true
	}

	var lower constraintBound
	var upper constraintBound
	excluded := make([]Version, 0)
	for _, candidate := range predicates {
		switch candidate.operator {
		case operatorNotEqual:
			excluded = append(excluded, candidate.version)
		case operatorGreater:
			lower = tighterLowerBound(lower, candidate.version, false)
		case operatorGreaterEqual:
			lower = tighterLowerBound(lower, candidate.version, true)
		case operatorLess:
			upper = tighterUpperBound(upper, candidate.version, false)
		case operatorLessEqual:
			upper = tighterUpperBound(upper, candidate.version, true)
		}
	}

	if !lower.set || !upper.set {
		return true
	}
	comparison := lower.version.Compare(upper.version)
	if comparison > 0 {
		return false
	}
	if comparison < 0 {
		return true
	}
	if !lower.inclusive || !upper.inclusive {
		return false
	}
	for _, version := range excluded {
		if lower.version.Compare(version) == 0 {
			return false
		}
	}

	return true
}

func tighterLowerBound(
	current constraintBound,
	version Version,
	inclusive bool,
) constraintBound {
	if !current.set {
		return constraintBound{
			version:   version,
			inclusive: inclusive,
			set:       true,
		}
	}
	comparison := version.Compare(current.version)
	if comparison > 0 {
		return constraintBound{
			version:   version,
			inclusive: inclusive,
			set:       true,
		}
	}
	if comparison == 0 {
		current.inclusive = current.inclusive && inclusive
	}

	return current
}

func tighterUpperBound(
	current constraintBound,
	version Version,
	inclusive bool,
) constraintBound {
	if !current.set {
		return constraintBound{
			version:   version,
			inclusive: inclusive,
			set:       true,
		}
	}
	comparison := version.Compare(current.version)
	if comparison < 0 {
		return constraintBound{
			version:   version,
			inclusive: inclusive,
			set:       true,
		}
	}
	if comparison == 0 {
		current.inclusive = current.inclusive && inclusive
	}

	return current
}

func (constraint Constraint) Matches(version Version) bool {
	for _, group := range constraint.groups {
		matches := true
		for _, predicate := range group.predicates {
			comparison := version.Compare(predicate.version)
			if !predicateMatches(predicate.operator, comparison) {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}

	return false
}

func predicateMatches(operator operator, comparison int) bool {
	switch operator {
	case operatorEqual:
		return comparison == 0
	case operatorNotEqual:
		return comparison != 0
	case operatorLess:
		return comparison < 0
	case operatorLessEqual:
		return comparison <= 0
	case operatorGreater:
		return comparison > 0
	case operatorGreaterEqual:
		return comparison >= 0
	default:
		return false
	}
}

func splitANDConstraints(input string) ([]string, *ParseError) {
	input = strings.TrimSpace(input)
	if input == "" ||
		strings.HasPrefix(input, ",") ||
		strings.HasSuffix(input, ",") ||
		strings.Contains(input, ",,") {
		return nil, newParseError(input, "constraint has an empty AND term")
	}

	fields := strings.Fields(strings.ReplaceAll(input, ",", " "))
	atoms := make([]string, 0, len(fields))
	for index := 0; index < len(fields); {
		if index+2 < len(fields) && fields[index+1] == "-" {
			atoms = append(
				atoms,
				fields[index]+" - "+fields[index+2],
			)
			index += 3
			continue
		}
		if fields[index] == "-" {
			return nil, newParseError(input, "constraint has an incomplete hyphen range")
		}
		if isComparisonOperator(fields[index]) {
			if index+1 >= len(fields) {
				return nil, newParseError(input, "comparison operator has no version")
			}
			atoms = append(atoms, fields[index]+fields[index+1])
			index += 2
			continue
		}

		atoms = append(atoms, fields[index])
		index++
	}
	if len(atoms) == 0 {
		return nil, newParseError(input, "constraint has no terms")
	}

	return atoms, nil
}

func isComparisonOperator(input string) bool {
	switch input {
	case "=", "==", "!=", "<>", ">", ">=", "<", "<=":
		return true
	default:
		return false
	}
}
