package compute_test

import (
	"encoding/json"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
	"gopkg.in/yaml.v3"
)

const (
	expectJSON = `{
  "byFile": [
    {
      "statementCoverage": "150/150",
      "blockCoverage": "1/1",
      "functionCoverage": "0/0",
      "statementPercentage": 100,
      "blockPercentage": 100,
      "functionPercentage": 100,
      "statementThreshold": 70,
      "blockThreshold": 50,
      "functionThreshold": 60,
      "failed": false,
      "file": "foo"
    }
  ],
  "byPackage": [
    {
      "statementCoverage": "150/150",
      "blockCoverage": "1/1",
      "functionCoverage": "0/0",
      "statementPercentage": 100,
      "blockPercentage": 100,
      "functionPercentage": 100,
      "statementThreshold": 70,
      "blockThreshold": 50,
      "functionThreshold": 60,
      "failed": false,
      "package": "."
    }
  ],
  "byTotal": {
    "statements": {
      "coverage": "150/150",
      "threshold": 70,
      "percentage": 100,
      "failed": false
    },
    "blocks": {
      "coverage": "1/1",
      "threshold": 50,
      "percentage": 100,
      "failed": false
    },
    "functions": {
      "coverage": "0/0",
      "threshold": 60,
      "percentage": 100,
      "failed": false
    }
  }
}`

	expectYAML = `byFile:
    - statementCoverage: 150/150
      blockCoverage: 1/1
      functionCoverage: 0/0
      statementPercentage: 100
      blockPercentage: 100
      functionPercentage: 100
      statementThreshold: 70
      blockThreshold: 50
      functionThreshold: 60
      failed: false
      file: foo
byPackage:
    - statementCoverage: 150/150
      blockCoverage: 1/1
      functionCoverage: 0/0
      statementPercentage: 100
      blockPercentage: 100
      functionPercentage: 100
      statementThreshold: 70
      blockThreshold: 50
      functionThreshold: 60
      failed: false
      package: .
byTotal:
    statements:
        coverage: 150/150
        threshold: 70
        percentage: 100
        failed: false
    blocks:
        coverage: 1/1
        threshold: 50
        percentage: 100
        failed: false
    functions:
        coverage: 0/0
        threshold: 60
        percentage: 100
        failed: false
`
)

func TestByFile_GetBy(t *testing.T) {
	f := &compute.ByFile{
		File: "foo",
		By: compute.By{
			Failed: true,
		},
	}
	require.NotEmpty(t, f.GetBy())
}

func TestByPackage_GetBy(t *testing.T) {
	p := &compute.ByPackage{
		Package: "foo",
		By: compute.By{
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

	cfg := new(config.Config)
	cfg.ApplyDefaults()
	r, _ := compute.CollectResults(profiles, cfg)
	out, err := yaml.Marshal(r)
	require.NoError(t, err)
	require.NotEmpty(t, out)
	require.YAMLEq(t, expectYAML, string(out))
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

	cfg := new(config.Config)
	cfg.ApplyDefaults()
	r, _ := compute.CollectResults(profiles, cfg)
	out, err := json.MarshalIndent(r, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, out)
	require.JSONEq(t, expectJSON, string(out))
}
