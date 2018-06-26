package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Config struct {
	SmtpServer         string `json:"SmtpServer"`
	SmtpUser           string `json:"SmtpUser"`
	SmtpPassword       string `json:"SmtpPassword"`
	EmailFrom          string `json:"EmailFrom"`
	RemoteGitUrlFormat string `json:"RemoteGitUrlFormat"`
	RemoteWebUrlFormat string `json:"RemoteWebUrlFormat"`
}

func LoadConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config *Config
	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
