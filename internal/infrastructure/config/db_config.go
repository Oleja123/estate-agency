package config

type DatabaseConfig struct {
	Username         string `yaml:"username"`
	Password         string `yaml:"password"`
	Host             string `yaml:"host"`
	Port             string `yaml:"port"`
	Database         string `yaml:"database"`
	MaxAttempts      int    `yaml:"max_attempts"`
	SecondsToConnect int    `yaml:"seconds_to_connect"`
}
