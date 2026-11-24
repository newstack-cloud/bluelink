package core

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

// PostgresDatabaseURL returns the connection URL for the PostgreSQL storage engine, with the
// pool settings as query parameters.
//
// The user, password and database are escaped, so a password containing characters such as
// "@", "/", ":", "#" or "?" reaches the server intact rather than being misparsed into the
// host or the path.
func (c *StateConfig) PostgresDatabaseURL() string {
	query := url.Values{}
	query.Set("sslmode", c.PostgresSSLMode)
	query.Set("pool_max_conns", strconv.Itoa(c.PostgresPoolMaxConns))
	query.Set("pool_max_conn_lifetime", c.PostgresPoolMaxConnLifetime)

	databaseURL := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.PostgresUser, c.PostgresPassword),
		Host:     net.JoinHostPort(unbracketedHost(c.PostgresHost), strconv.Itoa(c.PostgresPort)),
		Path:     "/" + c.PostgresDatabase,
		RawQuery: query.Encode(),
	}
	return databaseURL.String()
}

// An IPv6 host may be configured already bracketed, as it would be written in a URL, and
// JoinHostPort would bracket it a second time.
func unbracketedHost(host string) string {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return host[1 : len(host)-1]
	}
	return host
}
