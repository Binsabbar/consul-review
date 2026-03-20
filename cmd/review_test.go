package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ReviewSuite tests the cmd layer logic that can be unit-tested directly.
type ReviewSuite struct {
	suite.Suite
	tmpDir string
}

func TestReviewSuite(t *testing.T) {
	suite.Run(t, new(ReviewSuite))
}

func (s *ReviewSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

// ---------------------------------------------------------------------------
// resolveSkill
// ---------------------------------------------------------------------------

func (s *ReviewSuite) TestResolveSkill_TableDriven() {
	customFile := filepath.Join(s.tmpDir, "custom.md")
	s.Require().NoError(os.WriteFile(customFile, []byte("# custom skill"), 0o600))

	cfgFile := filepath.Join(s.tmpDir, "cfg.md")
	s.Require().NoError(os.WriteFile(cfgFile, []byte("# cfg skill"), 0o600))

	cases := []struct {
		name        string
		flagPath    string
		cfgPath     string
		wantContent string
		wantErr     bool
	}{
		{
			name:        "--skill flag takes highest priority",
			flagPath:    customFile,
			cfgPath:     cfgFile,
			wantContent: "# custom skill",
		},
		{
			name:        "config path used when flag is empty",
			flagPath:    "",
			cfgPath:     cfgFile,
			wantContent: "# cfg skill",
		},
		{
			name:        "bundled default used when both are empty",
			flagPath:    "",
			cfgPath:     "",
			wantContent: "", // non-empty — bundled skill present
		},
		{
			name:     "--skill flag with missing file returns error",
			flagPath: filepath.Join(s.tmpDir, "missing.md"),
			cfgPath:  "",
			wantErr:  true,
		},
		{
			name:     "config path with missing file returns error",
			flagPath: "",
			cfgPath:  filepath.Join(s.tmpDir, "missing.md"),
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			content, err := resolveSkill(tc.flagPath, tc.cfgPath)
			if tc.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			if tc.wantContent != "" {
				s.Require().Equal(tc.wantContent, content)
			} else {
				// Bundled default must be non-empty.
				s.Require().NotEmpty(content)
			}
		})
	}
}

func (s *ReviewSuite) TestResolveSkill_FlagOverridesCfg() {
	flagFile := filepath.Join(s.tmpDir, "flag.md")
	cfgFileP := filepath.Join(s.tmpDir, "cfg.md")

	s.Require().NoError(os.WriteFile(flagFile, []byte("flag content"), 0o600))
	s.Require().NoError(os.WriteFile(cfgFileP, []byte("cfg content"), 0o600))

	content, err := resolveSkill(flagFile, cfgFileP)
	s.Require().NoError(err)
	s.Require().Equal("flag content", content)
}
