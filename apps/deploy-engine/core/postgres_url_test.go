package core

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type PostgresURLTestSuite struct {
	suite.Suite
}

// Parsed by pgx rather than compared as a string, since what matters is that the server is
// sent exactly the configured values.
func (s *PostgresURLTestSuite) Test_credentials_and_database_with_url_delimiters_survive_parsing() {
	stateConfig := &StateConfig{
		PostgresUser:                "deploy@engine",
		PostgresPassword:            "p@ss:w/rd#?%&=",
		PostgresHost:                "db.internal",
		PostgresPort:                6543,
		PostgresDatabase:            "state/db?x",
		PostgresSSLMode:             "require",
		PostgresPoolMaxConns:        12,
		PostgresPoolMaxConnLifetime: "45m",
	}

	poolConfig, err := pgxpool.ParseConfig(stateConfig.PostgresDatabaseURL())
	s.Require().NoError(err)

	connConfig := poolConfig.ConnConfig
	s.Assert().Equal("deploy@engine", connConfig.User)
	s.Assert().Equal("p@ss:w/rd#?%&=", connConfig.Password)
	s.Assert().Equal("db.internal", connConfig.Host)
	s.Assert().Equal(uint16(6543), connConfig.Port)
	s.Assert().Equal("state/db?x", connConfig.Database)
	s.Assert().NotNil(connConfig.TLSConfig, "sslmode=require was not applied")
	s.Assert().Equal(int32(12), poolConfig.MaxConns)
	s.Assert().Equal(45*time.Minute, poolConfig.MaxConnLifetime)
}

func (s *PostgresURLTestSuite) Test_an_ipv6_host_is_bracketed() {
	for _, host := range []string{"::1", "[::1]"} {
		s.Run(host, func() {
			s.assertIPv6HostParses(host)
		})
	}
}

func (s *PostgresURLTestSuite) assertIPv6HostParses(host string) {
	stateConfig := &StateConfig{
		PostgresUser:                "bluelink",
		PostgresPassword:            "secret",
		PostgresHost:                host,
		PostgresPort:                5432,
		PostgresDatabase:            "bluelink",
		PostgresSSLMode:             "disable",
		PostgresPoolMaxConns:        100,
		PostgresPoolMaxConnLifetime: "1h30m",
	}

	poolConfig, err := pgxpool.ParseConfig(stateConfig.PostgresDatabaseURL())
	s.Require().NoError(err)
	s.Assert().Equal("::1", poolConfig.ConnConfig.Host)
	s.Assert().Equal(uint16(5432), poolConfig.ConnConfig.Port)
}

func TestPostgresURLTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresURLTestSuite))
}
