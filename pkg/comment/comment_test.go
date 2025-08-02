package comment

import (
	"context"
	"strings"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

func TestFormatMarkdown(t *testing.T) {
	// Create test results
	results := compute.Results{
		ByFile: []compute.ByFile{
			{
				By: compute.By{
					StatementPercentage: 80.5,
					BlockPercentage:     75.0,
					StatementThreshold:  70.0,
					BlockThreshold:      60.0,
					Failed:              false,
				},
				File: "test.go",
			},
		},
		ByPackage: []compute.ByPackage{
			{
				By: compute.By{
					StatementPercentage: 80.5,
					BlockPercentage:     75.0,
					StatementThreshold:  70.0,
					BlockThreshold:      60.0,
					Failed:              false,
				},
				Package: "github.com/test/package",
			},
		},
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{
				Coverage:   "161/200",
				Percentage: 80.5,
				Threshold:  70.0,
				Failed:     false,
			},
			Blocks: compute.TotalBlocks{
				Coverage:   "75/100",
				Percentage: 75.0,
				Threshold:  60.0,
				Failed:     false,
			},
		},
	}

	cfg := &config.Config{}

	// Test successful coverage report
	result := FormatMarkdown(results, cfg, true)

	// Verify basic structure
	if !strings.Contains(result, "## 🚦 Coverage Report") {
		t.Error("Missing coverage report header")
	}
	if !strings.Contains(result, "Coverage check passed") {
		t.Error("Missing success message")
	}
	if !strings.Contains(result, "### 📊 Total Coverage") {
		t.Error("Missing total coverage section")
	}
	if !strings.Contains(result, "go-covercheck") {
		t.Error("Missing tool attribution")
	}

	// Test with failed coverage
	results.ByTotal.Statements.Failed = true
	results.ByFile[0].By.Failed = true
	results.ByPackage[0].By.Failed = true

	result = FormatMarkdown(results, cfg, true)

	if !strings.Contains(result, "Coverage check failed") {
		t.Error("Missing failure message")
	}
	if !strings.Contains(result, "### 📋 Coverage Details") {
		t.Error("Missing coverage details section")
	}
	if !strings.Contains(result, "### 💡 Required Improvements") {
		t.Error("Missing improvements section")
	}
}

func TestNewPoster(t *testing.T) {
	tests := []struct {
		platform string
		wantErr  bool
	}{
		{"github", false},
		{"gitlab", false},
		{"gitea", false},
		{"GitHub", false}, // case insensitive
		{"GITLAB", false}, // case insensitive
		{"unsupported", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			_, err := NewPoster(tt.platform, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPoster() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGitHubPoster_Validation(t *testing.T) {
	poster := NewGitHubPoster("")
	
	tests := []struct {
		name     string
		cfg      *config.Config
		wantErr  bool
		errMsg   string
	}{
		{
			name: "missing token",
			cfg: &config.Config{
				Comment: config.CommentConfig{
					Platform: config.PlatformConfig{
						Repository:    "owner/repo",
						PullRequestID: 123,
					},
				},
			},
			wantErr: true,
			errMsg:  "github token is required",
		},
		{
			name: "missing repository",
			cfg: &config.Config{
				Comment: config.CommentConfig{
					Platform: config.PlatformConfig{
						Token:         "token",
						PullRequestID: 123,
					},
				},
			},
			wantErr: true,
			errMsg:  "repository is required",
		},
		{
			name: "missing PR ID",
			cfg: &config.Config{
				Comment: config.CommentConfig{
					Platform: config.PlatformConfig{
						Token:      "token",
						Repository: "owner/repo",
					},
				},
			},
			wantErr: true,
			errMsg:  "pull request ID is required",
		},
		{
			name: "valid config",
			cfg: &config.Config{
				Comment: config.CommentConfig{
					Platform: config.PlatformConfig{
						Token:         "token",
						Repository:    "owner/repo",
						PullRequestID: 123,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create empty results for testing
			results := compute.Results{}
			
			err := poster.PostComment(context.Background(), results, tt.cfg)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("PostComment() expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("PostComment() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else if !tt.wantErr && err != nil {
				// For valid config, we expect a network error since we're not mocking the HTTP client
				// but we should not get a validation error
				if strings.Contains(err.Error(), "token is required") ||
					strings.Contains(err.Error(), "repository is required") ||
					strings.Contains(err.Error(), "pull request ID is required") {
					t.Errorf("PostComment() got validation error with valid config: %v", err)
				}
			}
		})
	}
}