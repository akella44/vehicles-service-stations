package db

import (
	"fmt"

	_ "github.com/lib/pq"
)

type Config struct {
	Addr     string
	Port     uint16
	User     string
	Password string
	DBName   string
}

func NewConfig(addr string, port uint16, user, password, dbName string) (*Config, error) {
	if addr == "" || port == 0 || user == "" || dbName == "" {
		return nil, fmt.Errorf("invalid values, one of fileds is empty")
	}

	config := &Config{
		Addr:     addr,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbName,
	}

	return config, nil
}

func (c *Config) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.User, c.Password, c.Addr, c.Port, c.DBName)
}
