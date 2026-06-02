package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	EdgePublicIngress   = "public_ingress"
	EdgeOpenAITunnel    = "openai_tunnel"
	EdgeAnthropicTunnel = "anthropic_tunnel"
	EdgePrivateHTTP     = "private_http"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Edge     EdgeConfig     `yaml:"edge"`
	Auth     AuthConfig     `yaml:"auth"`
	Policy   PolicyConfig   `yaml:"policy"`
	Postgres PostgresConfig `yaml:"postgres"`
	Audit    AuditConfig    `yaml:"audit"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type EdgeConfig struct {
	Mode string `yaml:"mode"`
}

type AuthConfig struct {
	Issuer         string   `yaml:"issuer"`
	Audience       string   `yaml:"audience"`
	JWKSURL        string   `yaml:"jwks_url"`
	GroupClaimName string   `yaml:"group_claim_name"`
	RequiredGroups []string `yaml:"required_groups"`
}

type PolicyConfig struct {
	DefaultToolPack string              `yaml:"default_tool_pack"`
	GroupToolPacks  map[string][]string `yaml:"group_tool_packs"`
}

type PostgresConfig struct {
	DSNEnv    string                       `yaml:"dsn_env"`
	MaxRows   int                          `yaml:"max_rows"`
	ToolPacks map[string]ToolPackAllowlist `yaml:"tool_packs"`
}

type ToolPackAllowlist struct {
	Tables map[string]TableAllowlist `yaml:"tables"`
}

type TableAllowlist struct {
	Columns []string `yaml:"columns"`
}

type AuditConfig struct {
	Sink            string `yaml:"sink"`
	IncludePayloads bool   `yaml:"include_payloads"`
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	applyDefaults(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = "127.0.0.1:8080"
	}
	if cfg.Edge.Mode == "" {
		cfg.Edge.Mode = EdgePrivateHTTP
	}
	if cfg.Audit.Sink == "" {
		cfg.Audit.Sink = "stdout"
	}
	if cfg.Auth.GroupClaimName == "" {
		cfg.Auth.GroupClaimName = "groups"
	}
}

func (cfg Config) Validate() error {
	if cfg.Server.Listen == "" {
		return fmt.Errorf("server.listen is required")
	}
	if !validEdgeMode(cfg.Edge.Mode) {
		return fmt.Errorf("edge.mode %q is invalid", cfg.Edge.Mode)
	}
	if cfg.Auth.Issuer == "" {
		return fmt.Errorf("auth.issuer is required")
	}
	if cfg.Auth.Audience == "" {
		return fmt.Errorf("auth.audience is required")
	}
	if len(cfg.Auth.RequiredGroups) == 0 {
		return fmt.Errorf("auth.required_groups must include at least one group")
	}
	if cfg.Policy.DefaultToolPack == "" {
		return fmt.Errorf("policy.default_tool_pack is required")
	}
	if len(cfg.Policy.GroupToolPacks) == 0 {
		return fmt.Errorf("policy.group_tool_packs must include at least one group")
	}
	if err := cfg.Postgres.validate(cfg.Policy); err != nil {
		return err
	}
	return nil
}

func (p PostgresConfig) validate(policy PolicyConfig) error {
	if p.DSNEnv == "" {
		return fmt.Errorf("postgres.dsn_env is required")
	}
	if p.MaxRows <= 0 {
		return fmt.Errorf("postgres.max_rows must be greater than zero")
	}
	if len(p.ToolPacks) == 0 {
		return fmt.Errorf("postgres.tool_packs must include at least one tool pack")
	}
	for packName, pack := range p.ToolPacks {
		if err := pack.validate(packName); err != nil {
			return err
		}
	}
	for _, packs := range policy.GroupToolPacks {
		for _, pack := range packs {
			if _, ok := p.ToolPacks[pack]; !ok {
				return fmt.Errorf("policy.group_tool_packs references unknown postgres tool pack %q", pack)
			}
		}
	}
	if _, ok := p.ToolPacks[policy.DefaultToolPack]; !ok {
		return fmt.Errorf("policy.default_tool_pack %q is not defined in postgres.tool_packs", policy.DefaultToolPack)
	}
	return nil
}

func (p ToolPackAllowlist) validate(packName string) error {
	if len(p.Tables) == 0 {
		return fmt.Errorf("postgres.tool_packs.%s.tables must include at least one table", packName)
	}
	for table, allow := range p.Tables {
		if table == "" {
			return fmt.Errorf("postgres.tool_packs.%s.tables contains an empty table name", packName)
		}
		if err := allow.validate(packName, table); err != nil {
			return err
		}
	}
	return nil
}

func (t TableAllowlist) validate(packName, table string) error {
	if len(t.Columns) == 0 {
		return fmt.Errorf("postgres.tool_packs.%s.tables.%s.columns must include at least one column", packName, table)
	}
	for _, col := range t.Columns {
		if col == "" {
			return fmt.Errorf("postgres.tool_packs.%s.tables.%s.columns contains an empty column name", packName, table)
		}
	}
	return nil
}

func validEdgeMode(mode string) bool {
	switch mode {
	case EdgePublicIngress, EdgeOpenAITunnel, EdgeAnthropicTunnel, EdgePrivateHTTP:
		return true
	default:
		return false
	}
}
