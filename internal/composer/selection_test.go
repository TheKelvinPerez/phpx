package composer_test

import (
	"testing"

	"github.com/elefantephp/elefante/internal/composer"
	"github.com/elefantephp/elefante/internal/model"
)

func TestSelectExecutableAppliesConstraintBeforeSourcePrecedence(t *testing.T) {
	t.Parallel()

	selected, err := composer.SelectExecutable(
		[]model.ComposerObservation{
			{
				Version:  "2.7.9",
				Source:   composer.SourceManaged,
				Path:     "/cache/composer-2.7.9.phar",
				Identity: "sha256:managed-incompatible",
			},
			{
				Version:  "2.8.8",
				Source:   composer.SourceManaged,
				Path:     "/cache/composer-2.8.8.phar",
				Identity: "sha256:managed-compatible",
			},
			{
				Version:  "2.8.9",
				Source:   composer.SourceProvider,
				Path:     "/provider/composer",
				Identity: "sha256:provider",
			},
			{
				Version:  "2.8.10",
				Source:   composer.SourceSystem,
				Path:     "/usr/local/bin/composer",
				Identity: "sha256:system",
			},
		},
		"^2.8",
	)
	if err != nil {
		t.Fatalf("select Composer executable: %v", err)
	}
	if selected.Identity != "sha256:managed-compatible" {
		t.Fatalf("expected compatible managed Composer, got %#v", selected)
	}
}
