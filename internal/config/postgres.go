package config

import (
	"net"
	"net/url"
	"strconv"

	"github.com/caarlos0/env/v11"
)

// Postgres holds the database connection settings. Defaults match the
// docker-compose setup, so a local run needs no environment at all.
type Postgres struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	DB       string `env:"POSTGRES_DB" envDefault:"games_db"`
	User     string `env:"POSTGRES_USER" envDefault:"parser"`
	Password string `env:"POSTGRES_PASSWORD" envDefault:"parser_password"`
	SSLMode  string `env:"POSTGRES_SSLMODE" envDefault:"disable"`
}

func ParsePostgres() (Postgres, error) {
	return env.ParseAs[Postgres]()
}

func (p Postgres) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(p.User, p.Password),
		Host:     net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
		Path:     p.DB,
		RawQuery: url.Values{"sslmode": {p.SSLMode}}.Encode(),
	}

	return u.String()
}
