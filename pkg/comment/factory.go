package comment

import (
	"fmt"
	"strings"
)

// SupportedPlatforms lists all supported platforms.
var SupportedPlatforms = []string{"github", "gitlab", "gitea"}

// NewPoster creates a new Poster instance based on the platform type.
func NewPoster(platformType, baseURL string) (Poster, error) {
	switch strings.ToLower(platformType) {
	case "github":
		return NewGitHubPoster(baseURL), nil
	case "gitlab":
		return NewGitLabPoster(baseURL), nil
	case "gitea":
		return NewGiteaPoster(baseURL), nil
	default:
		return nil, fmt.Errorf("unsupported platform type: %s (supported: %s)",
			platformType, strings.Join(SupportedPlatforms, ", "))
	}
}
