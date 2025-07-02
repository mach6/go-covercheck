package formatter //nolint:testpackage

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func Test_normalizeNames(t *testing.T) {
	names := []string{
		"do.main/pkg/main.go",
		"do.main/pkg/utils/utils.go",
		"do.main/pkg/foo/foo.go",
		"do.main/pkg/bar.go",
		"do.main/internal/baz.go",
	}
	expect := []string{
		"pkg/main.go",
		"pkg/utils/utils.go",
		"pkg/foo/foo.go",
		"pkg/bar.go",
		"internal/baz.go",
	}

	profiles := make([]*cover.Profile, len(names))
	for i, name := range names {
		profiles[i] = &cover.Profile{
			FileName: name,
		}
	}

	normalizeNames(profiles)
	for i, profile := range profiles {
		require.Equal(t, expect[i], profile.FileName)
	}
}
