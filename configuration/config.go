package configuration

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	LastUpdateDate string `yaml:"last-update-date"`
	Hostname       string `yaml:"host"`
	Port           int    `yaml:"port"`
	Username       string `yaml:"user"`
	MaxConnections int    `yaml:"max-concurrent-connections"`
	ServerFolder   string `yaml:"server-folder"`
}

const Filename = ".sftp.config.yaml"

func Read() (*Configuration, error) {
	content, err := os.ReadFile(Filename)
	if err != nil {
		return nil, err
	}
	var config Configuration
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func epochTime() string {
	return time.Date(1970, time.January, 01, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
}

func currentTime() string {
	return time.Now().Format(time.RFC3339)
}

func New() Configuration {
	return Configuration{epochTime(), "localhost", 22, "test", 3, "."}
}

func (c *Configuration) UpdateTime() {
	(*c).LastUpdateDate = currentTime()
}

func (c *Configuration) Store() error {
	yamlData, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(Filename, yamlData, 0644)
}
