package configs

import yaml "gopkg.in/yaml.v3"

type User struct {
	UserID     string `yaml:"user_id"`
	SessionID  string `yaml:"session_id"`
	Pin        string `yaml:"pin"`
	PinToken   string `yaml:"pin_token"`
	PrivateKey string `yaml:"private_key"`
}

type Option struct {
	HTTP struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"http"`
	Database struct {
		Name string `yaml:"name"`
		Path string `yaml:"path"`
	} `yaml:"database"`
	Mixin struct {
		AppID      string   `yaml:"app_id"`
		SessionID  string   `yaml:"session_id"`
		Secret     string   `yaml:"secret"`
		Pin        string   `yaml:"pin"`
		PinToken   string   `yaml:"pin_token"`
		PrivateKey string   `yaml:"private_key"`
		Receivers  []string `yaml:"receivers"`
		Master     bool     `yaml:"master"`
	} `yaml:"mixin"`
	Environment string
}

var AppConfig *Option

func Init(env string) (*Option, error) {
	var options map[string]Option
	err := yaml.Unmarshal([]byte(dataInYML), &options)
	if err != nil {
		return nil, err
	}
	opt := options[env]
	opt.Environment = env
	AppConfig = &opt
	return AppConfig, nil
}

const (
	BuildVersion = "BUILD_VERSION"
)
