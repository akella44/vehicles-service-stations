package db

import (
	"fmt"

	_ "github.com/lib/pq"

	"vehicles-service-stations/config"
)

type Config struct {
	Addr     string
	Port     int
	User     string
	Password string
	DBName   string
}

func NewConfig(envConfig *config.Config, user, password string) (*Config, error) {
	config := &Config{
		Addr:     envConfig.DbIp,
		Port:     envConfig.DbPort,
		User:     user,
		Password: password,
		DBName:   envConfig.DbName,
	}

	return config, nil
}

func (c *Config) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.User, c.Password, c.Addr, c.Port, c.DBName)
}
