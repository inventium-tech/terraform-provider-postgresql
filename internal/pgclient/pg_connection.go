package pgclient

import (
	"fmt"
)

type ConnConfig struct {
	Host     string `json:"host" validate:"required"`
	Port     int    `json:"port" validate:"required"`
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Database string `json:"database" validate:"required"`
	SSLMode  string `json:"ssl_mode" validate:"required,oneof=disable require verify-ca verify-full"`
}

func (cfg *ConnConfig) String() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	)
}

func (cfg *ConnConfig) ID() string {
	return fmt.Sprintf("%s:%s@%s", cfg.Host, cfg.Username, cfg.Database)
}
