package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ConfigSuite contains all tests for the config package.
type ConfigSuite struct {
	suite.Suite
	tmpDir string
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

func (s *ConfigSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

// ---------------------------------------------------------------------------
// Load + Unmarshal
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestLoad_ValidYAML() {
	skillFile := filepath.Join(s.tmpDir, "SKILL.md")
	s.Require().NoError(os.WriteFile(skillFile, []byte("# skill"), 0o600))

	yaml := fmt.Sprintf(`
code_review_skill: "%s"
gemini:
  enabled: true
  model: "gemini-3-pro"
copilot:
  enabled: false
  model: "gpt-4"
oz:
  enabled: false
  model: "claude-3-5-sonnet"
`, skillFile)

	cfgFile := filepath.Join(s.tmpDir, "config.yaml")
	s.Require().NoError(os.WriteFile(cfgFile, []byte(yaml), 0o600))

	cfg, err := Load(cfgFile)
	s.Require().NoError(err)
	s.Require().Equal(skillFile, cfg.CodeReviewSkill)

	s.Require().Equal("gemini-3-pro", cfg.Gemini.Model)
	s.Require().True(cfg.Gemini.Enabled)
	s.Require().False(cfg.Copilot.Enabled)
	s.Require().False(cfg.Oz.Enabled)
}

func (s *ConfigSuite) TestLoad_ExtraArgs() {
	yaml := `
gemini:
  enabled: true
  model: "gemini-3-pro"
  extra_args: ["--yolo", "--sandbox"]
`
	cfgFile := filepath.Join(s.tmpDir, "config.yaml")
	s.Require().NoError(os.WriteFile(cfgFile, []byte(yaml), 0o600))

	cfg, err := Load(cfgFile)
	s.Require().NoError(err)
	s.Require().Equal([]string{"--yolo", "--sandbox"}, cfg.Gemini.ExtraArgs)
}

func (s *ConfigSuite) TestLoad_FileNotFound() {
	_, err := Load(filepath.Join(s.tmpDir, "nonexistent.yaml"))
	s.Require().Error(err)
}

// ---------------------------------------------------------------------------
// DefaultExtraArgs
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestDefaultExtraArgs() {
	s.Require().Equal([]string{"--yolo"}, DefaultExtraArgs("gemini"))
	s.Require().Equal([]string{"--allow-all-tools"}, DefaultExtraArgs("copilot"))
	s.Require().Equal([]string{"--no-interactive"}, DefaultExtraArgs("oz"))
	s.Require().Nil(DefaultExtraArgs("unknown"))
}

// ---------------------------------------------------------------------------
// EnabledConsuls
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestEnabledConsuls_OnlyReturnsEnabled() {
	cfg := &Config{
		Gemini:  ConsulConfig{Enabled: true, Model: "gemini-3-pro"},
		Copilot: ConsulConfig{Enabled: false, Model: "gpt-4"},
		Oz:      ConsulConfig{Enabled: false, Model: "claude"},
	}

	enabled := cfg.EnabledConsuls()
	s.Require().Len(enabled, 1)
	s.Require().Contains(enabled, "gemini")
	s.Require().NotContains(enabled, "copilot")
	s.Require().NotContains(enabled, "oz")
}

func (s *ConfigSuite) TestEnabledConsuls_AllEnabled() {
	cfg := &Config{
		Gemini:  ConsulConfig{Enabled: true},
		Copilot: ConsulConfig{Enabled: true},
		Oz:      ConsulConfig{Enabled: true},
	}
	s.Require().Len(cfg.EnabledConsuls(), 3)
}

func (s *ConfigSuite) TestEnabledConsuls_NoneEnabled() {
	cfg := &Config{}
	s.Require().Empty(cfg.EnabledConsuls())
}

// ---------------------------------------------------------------------------
// Validate
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidate_TableDriven() {
	skillFile := filepath.Join(s.tmpDir, "SKILL.md")
	s.Require().NoError(os.WriteFile(skillFile, []byte("# skill"), 0o600))

	cases := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config with skill file",
			cfg: &Config{
				CodeReviewSkill: skillFile,
				Gemini:          ConsulConfig{Enabled: true, Model: "gemini-3-pro"},
			},
			wantErr: false,
		},
		{
			name:    "empty skill path uses bundled default — valid",
			cfg:     &Config{Gemini: ConsulConfig{Enabled: true}},
			wantErr: false,
		},
		{
			name: "skill file not on disk",
			cfg: &Config{
				CodeReviewSkill: filepath.Join(s.tmpDir, "missing.md"),
				Gemini:          ConsulConfig{Enabled: true},
			},
			wantErr: true,
		},
		{
			name:    "no enabled consuls",
			cfg:     &Config{CodeReviewSkill: skillFile},
			wantErr: true,
		},
		{
			name: "all consuls disabled",
			cfg: &Config{
				Gemini:  ConsulConfig{Enabled: false},
				Copilot: ConsulConfig{Enabled: false},
				Oz:      ConsulConfig{Enabled: false},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			err := Validate(tc.cfg)
			if tc.wantErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CheckBinaries
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestCheckBinaries_TableDriven() {
	cases := []struct {
		name        string
		cfg         *Config
		fakeLookup  map[string]bool
		wantErr     bool
		errContains string
	}{
		{
			name:       "all binaries present",
			cfg:        &Config{Gemini: ConsulConfig{Enabled: true}},
			fakeLookup: map[string]bool{"gemini": true, "gh": true, "claude": true},
			wantErr:    false,
		},
		{
			name:        "missing gemini binary",
			cfg:         &Config{Gemini: ConsulConfig{Enabled: true}},
			fakeLookup:  map[string]bool{"gh": true, "claude": true},
			wantErr:     true,
			errContains: "gemini",
		},
		{
			name:        "missing gh binary",
			cfg:         &Config{Gemini: ConsulConfig{Enabled: true}},
			fakeLookup:  map[string]bool{"gemini": true, "claude": true},
			wantErr:     true,
			errContains: "gh",
		},
		{
			name:       "disabled consul binary not required",
			cfg:        &Config{Gemini: ConsulConfig{Enabled: true}, Copilot: ConsulConfig{Enabled: false}},
			fakeLookup: map[string]bool{"gemini": true, "gh": true, "claude": true},
			wantErr:    false,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			orig := LookPathFunc
			defer func() { LookPathFunc = orig }()

			LookPathFunc = func(file string) (string, error) {
				if tc.fakeLookup[file] {
					return "/usr/local/bin/" + file, nil
				}
				return "", fmt.Errorf("%q not found", file)
			}

			err := CheckBinaries(tc.cfg)
			if tc.wantErr {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errContains)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
