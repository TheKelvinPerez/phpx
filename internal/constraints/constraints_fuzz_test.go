package constraints

import "testing"

func FuzzConstraintEngine(f *testing.F) {
	seeds := [][4]string{
		{"8.4.1", "^8.3", ">=8 <9", "^8.4"},
		{"8.4.0-beta1", ">=8.4@beta", "<8.4@beta", ">=8.4@beta"},
		{"0.0.3", "^0.0.3", "8.4.*", "~8.4.2"},
		{"not-a-version", "dev-main", ">=8.2 ||", "~>8.3"},
		{"20264.10.99999", ">=20264.10 <20265", "*", "!=8.4"},
	}
	for _, seed := range seeds {
		f.Add(seed[0], seed[1], seed[2], seed[3])
	}

	f.Fuzz(func(
		t *testing.T,
		version string,
		constraint string,
		left string,
		right string,
	) {
		if len(version)+len(constraint)+len(left)+len(right) > 4096 {
			return
		}

		firstMatch, firstMatchErr := Satisfies(version, constraint)
		secondMatch, secondMatchErr := Satisfies(version, constraint)
		assertDeterministicResult(
			t,
			firstMatch,
			firstMatchErr,
			secondMatch,
			secondMatchErr,
		)

		_, leftErr := Parse(left)
		_, rightErr := Parse(right)
		if leftErr == nil && rightErr == nil {
			forward, forwardErr := Intersects(left, right)
			reverse, reverseErr := Intersects(right, left)
			if forwardErr != nil || reverseErr != nil {
				t.Fatalf(
					"parsed constraints failed intersection: %v, %v",
					forwardErr,
					reverseErr,
				)
			}
			if forward != reverse {
				t.Fatalf(
					"intersection is not symmetric: %t versus %t",
					forward,
					reverse,
				)
			}
		}
	})
}

func assertDeterministicResult(
	t *testing.T,
	first bool,
	firstErr error,
	second bool,
	secondErr error,
) {
	t.Helper()

	if first != second {
		t.Fatalf("nondeterministic result: %t versus %t", first, second)
	}
	if (firstErr == nil) != (secondErr == nil) {
		t.Fatalf("nondeterministic error presence: %v versus %v", firstErr, secondErr)
	}
	if firstErr != nil && firstErr.Error() != secondErr.Error() {
		t.Fatalf("nondeterministic errors: %v versus %v", firstErr, secondErr)
	}
}
