package discovery

import "testing"

func TestNormalizeRemoteURLRemovesCredentials(t *testing.T) {
	tests := []struct {
		name     string
		remote   string
		expected string
	}{
		{
			name:     "HTTPS credentials",
			remote:   "https://person:secret@example.com/acme/project.git",
			expected: "https://example.com/acme/project.git",
		},
		{
			name:     "SSH user",
			remote:   "ssh://git@example.com/acme/project.git",
			expected: "ssh://example.com/acme/project.git",
		},
		{
			name:     "SCP user",
			remote:   "git@example.com:acme/project.git",
			expected: "example.com:acme/project.git",
		},
		{
			name:     "local path",
			remote:   "/srv/git/project.git",
			expected: "/srv/git/project.git",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := normalizeRemoteURL(test.remote); got != test.expected {
				t.Errorf("expected normalized remote %q, got %q", test.expected, got)
			}
		})
	}
}
