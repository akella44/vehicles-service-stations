package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	DbIp        string `mapstructure:"DB_IP"`
	DbPort      int    `mapstructure:"DB_PORT"`
	DbName      string `mapstructure:"DB_NAME"`
	DbSuperuser string `mapstructure:"DB_SUPERUSER_LOGIN"`
	DbPassword  string `mapstructure:"DB_SUPERUSER_PASSWORD"`
}

func LoadConfig() (*Config, error) {
	var cfg Config
	viper.AutomaticEnv()
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return &cfg, err
		}
	}

	err := viper.Unmarshal(&cfg)
	if err != nil {
		return &cfg, fmt.Errorf("unable to decode into config struct, %v", err)
	}

	return &cfg, nil
}
