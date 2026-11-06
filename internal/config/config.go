package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Origin struct {
	Code, Name, DBLink string
	Enabled            bool
}
type Config struct {
	Port, Env, CorsOrigins, JWTSecret, JWTIssuer, JWTAudience string
	OracleDSN, OracleUser, OraclePassword, GrainFunction      string
	Empresa, MaxOpen, MaxIdle                                 int
	ConnLifetime, QueryTimeout, ReadTimeout, WriteTimeout     time.Duration
	Origins                                                   map[string]Origin
}

var ident = regexp.MustCompile(
	`^[A-Za-z][A-Za-z0-9_$#]*(\.[A-Za-z][A-Za-z0-9_$#]*)*$`,
)

func env(k, d string) string {
	if v, ok := os.LookupEnv(k); ok {
		return strings.TrimSpace(v)
	}
	return d
}
func num(k string, d int) (int, error) {
	v := env(k, strconv.Itoa(d))
	n, e := strconv.Atoi(v)
	if e != nil || n < 0 {
		return 0, fmt.Errorf("invalid %s", k)
	}
	return n, nil
}
func Load() (Config, error) {
	c := Config{Port: env("HTTP_PORT", "9042"), Env: env("APP_ENV", "test"), CorsOrigins: env("CORS_ALLOWED_ORIGINS", ""), JWTSecret: env("JWT_SECRET", ""), JWTIssuer: env("JWT_ISSUER", ""), JWTAudience: env("JWT_AUDIENCE", ""), OracleDSN: env("ORACLE_DSN", ""), OracleUser: env("ORACLE_USER", ""), OraclePassword: env("ORACLE_PASSWORD", ""), GrainFunction: env("ORACLE_GRAIN_NAME_FUNCTION", "B_TB0"), Origins: map[string]Origin{}}
	var e error
	if c.Empresa, e = num("ORACLE_EMPRESA", 1); e != nil {
		return c, e
	}
	if c.MaxOpen, e = num("ORACLE_MAX_OPEN_CONNS", 5); e != nil {
		return c, e
	}
	if c.MaxIdle, e = num("ORACLE_MAX_IDLE_CONNS", 2); e != nil {
		return c, e
	}
	var n int
	if n, e = num("ORACLE_CONN_MAX_LIFETIME_MINUTES", 30); e != nil {
		return c, e
	}
	c.ConnLifetime = time.Duration(n) * time.Minute
	if n, e = num("QUERY_TIMEOUT_SECONDS", 90); e != nil {
		return c, e
	}
	c.QueryTimeout = time.Duration(n) * time.Second
	if n, e = num("HTTP_READ_TIMEOUT_SECONDS", 15); e != nil {
		return c, e
	}
	c.ReadTimeout = time.Duration(n) * time.Second
	if n, e = num("HTTP_WRITE_TIMEOUT_SECONDS", 120); e != nil {
		return c, e
	}
	c.WriteTimeout = time.Duration(n) * time.Second
	if c.JWTSecret == "" || strings.HasPrefix(c.JWTSecret, "REEMPLAZAR") || c.OracleDSN == "" || c.OracleUser == "" || c.OraclePassword == "" || strings.HasPrefix(c.OraclePassword, "REEMPLAZAR") {
		return c, fmt.Errorf("configure JWT_SECRET and central Oracle credentials")
	}
	if !ident.MatchString(c.GrainFunction) {
		return c, fmt.Errorf("invalid grain function identifier")
	}
	for _, code := range []string{"NEC", "SL", "TIM", "SEL", "VIL", "AC", "CO"} {
		prefix := "ORIGIN_" + code + "_"
		enabled, e := strconv.ParseBool(env(prefix+"ENABLED", "false"))
		if e != nil {
			return c, fmt.Errorf("invalid %sENABLED", prefix)
		}

		o := Origin{
			Code:    code,
			Name:    env(prefix+"NAME", code),
			DBLink:  strings.TrimSpace(env(prefix+"DBLINK", "")),
			Enabled: enabled,
		}

		if o.DBLink != "" && !ident.MatchString(o.DBLink) {
			return c, fmt.Errorf(
				"invalid DBLINK for %s: value=%q",
				code,
				o.DBLink,
			)
		}

		if enabled && o.DBLink == "" && code != "AC" && code != "CO" {
			return c, fmt.Errorf(
				"DBLINK required for %s",
				code,
			)
		}

		c.Origins[code] = o
		c.Origins[code] = o
	}
	return c, nil
}
