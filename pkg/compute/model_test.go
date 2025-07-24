package compute

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

const (
	expectJson = `{
  "byFile": [
    {
      "statementCoverage": "150/150",
      "blockCoverage": "1/1",
      "statementPercentage": 100,
      "blockPercentage": 100,
      "statementThreshold": 0,
      "blockThreshold": 0,
      "failed": false,
      "file": "foo"
    }
  ],
  "byPackage": [
    {
      "statementCoverage": "150/150",
      "blockCoverage": "1/1",
      "statementPercentage": 100,
      "blockPercentage": 100,
      "statementThreshold": 0,
      "blockThreshold": 0,
      "failed": false,
      "package": "."
    }
  ],
  "byTotal": {
    "statements": {
      "coverage": "150/150",
      "threshold": 0,
      "percentage": 100,
      "failed": false
    },
    "blocks": {
      "coverage": "1/1",
      "threshold": 0,
      "percentage": 100,
      "failed": false
    }
  }
}`

	expectYaml = `byFile:
    - statementCoverage: 150/150
      blockCoverage: 1/1
      statementPercentage: 100
      blockPercentage: 100
      statementThreshold: 0
      blockThreshold: 0
      failed: false
      file: foo
byPackage:
    - statementCoverage: 150/150
      blockCoverage: 1/1
      statementPercentage: 100
      blockPercentage: 100
      statementThreshold: 0
      blockThreshold: 0
      failed: false
      package: .
byTotal:
    statements:
        coverage: 150/150
        threshold: 0
        percentage: 100
        failed: false
    blocks:
        coverage: 1/1
        threshold: 0
        percentage: 100
        failed: false
`
)

func TestByFile_GetBy(t *testing.T) {
	f := &ByFile{
		File: "foo",
		By: By{
			Failed: true,
		},
	}
	require.NotEmpty(t, f.GetBy())
}

func TestByPackage_GetBy(t *testing.T) {
	p := &ByPackage{
		Package: "foo",
		By: By{
			Failed: true,
		},
	}
	require.NotEmpty(t, p.GetBy())
}

func TestModelMarshalYaml(t *testing.T) {
	profiles := []*cover.Profile{
		{
			FileName: "foo",
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{
					StartLine: 0,
					StartCol:  0,
					EndLine:   10,
					EndCol:    120,
					NumStmt:   150,
					Count:     1,
				},
			},
		},
	}

	r, _ := CollectResults(profiles, new(config.Config))
	out, err := yaml.Marshal(r)
	require.NoError(t, err)
	require.NotEmpty(t, out)
	require.Equal(t, expectYaml, string(out))
}

func TestModelMarshalJson(t *testing.T) {
	profiles := []*cover.Profile{
		{
			FileName: "foo",
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{
					StartLine: 0,
					StartCol:  0,
					EndLine:   10,
					EndCol:    120,
					NumStmt:   150,
					Count:     1,
				},
			},
		},
	}

	r, _ := CollectResults(profiles, new(config.Config))
	out, err := json.MarshalIndent(r, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, out)
	require.Equal(t, expectJson, string(out))
}
