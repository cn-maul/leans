package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Analysis 控制分析时的 token 预算。数值越小，单次分析越省 token，
// 但注入的讲义内容越少。可在 config.yaml 的 analysis 段调整。
type Analysis struct {
	LectureBudget int `yaml:"lecture_budget"` // 讲义注入字符上限（控制 prompt token 大头）
	MaxSections   int `yaml:"max_sections"`   // 检索注入章节数
	MaxChars      int `yaml:"max_chars"`      // 每章节最大字符
	OverviewMax   int `yaml:"overview_max"`   // 概述章节数
	MaxTokens     int `yaml:"max_tokens"`     // 输出 token 硬顶（防模型输出失控，0=不限制）
}

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Data struct {
		Dir string `yaml:"dir"`
	} `yaml:"data"`
	AI struct {
		Provider string `yaml:"provider"`
		APIKey   string `yaml:"api_key"`
		BaseURL  string `yaml:"base_url"`
		Model    string `yaml:"model"`
	} `yaml:"ai"`
	Analysis Analysis `yaml:"analysis"`
	Subjects struct {
		Dir string `yaml:"dir"`
	} `yaml:"subjects"`
}

// defaults returns the built-in fallback config. Used when config/config.yaml
// is missing so the single binary still boots without extra files.
func defaults() *Config {
	var cfg Config
	cfg.Server.Port = 8080
	cfg.Data.Dir = "data"
	cfg.Subjects.Dir = "../subjects"
	cfg.AI.Provider = "openai"
	cfg.AI.BaseURL = "https://api.openai.com/v1"
	cfg.Analysis.LectureBudget = 6000
	cfg.Analysis.MaxSections = 2
	cfg.Analysis.MaxChars = 1500
	cfg.Analysis.OverviewMax = 1
	cfg.Analysis.MaxTokens = 4096
	return &cfg
}

// Load reads config/config.yaml when present, falling back to built-in
// defaults. Paths are relative to the working directory (the backend dir).
func Load() (*Config, error) {
	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		if os.IsNotExist(err) {
			return defaults(), nil
		}
		return nil, err
	}

	cfg := defaults()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.AI.BaseURL == "" {
		cfg.AI.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Data.Dir == "" {
		cfg.Data.Dir = "data"
	}
	if cfg.Subjects.Dir == "" {
		cfg.Subjects.Dir = "../subjects"
	}
	if cfg.Analysis.LectureBudget <= 0 {
		cfg.Analysis.LectureBudget = 6000
	}
	if cfg.Analysis.MaxSections <= 0 {
		cfg.Analysis.MaxSections = 2
	}
	if cfg.Analysis.MaxChars <= 0 {
		cfg.Analysis.MaxChars = 1500
	}
	if cfg.Analysis.OverviewMax <= 0 {
		cfg.Analysis.OverviewMax = 1
	}

	return cfg, nil
}
