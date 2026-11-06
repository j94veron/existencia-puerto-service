package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type OracleConnectionMode string

const (
	OracleDirect OracleConnectionMode = "DIRECT"
	OracleDBLink OracleConnectionMode = "DBLINK"
)

type Origin struct {
	Code    string
	Name    string
	DBLink  string
	Enabled bool
}

type Config struct {
	Port        string
	Env         string
	CorsOrigins string
	JWTSecret   string
	JWTIssuer   string
	JWTAudience string

	OracleDSN            string
	OracleUser           string
	OraclePassword       string
	GrainFunction        string
	OracleConnectionMode OracleConnectionMode

	Empresa int
	MaxOpen int
	MaxIdle int

	ConnLifetime time.Duration
	QueryTimeout time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	Origins map[string]Origin
}

// Permite identificadores Oracle y DBLINK con varios segmentos:
// ACANECO.ACACOOP.COM.AR
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
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid %s", k)
	}
	return n, nil
}

func Load() (Config, error) {
	c := Config{
		Port:           env("HTTP_PORT", "9042"),
		Env:            env("APP_ENV", "test"),
		CorsOrigins:    env("CORS_ALLOWED_ORIGINS", ""),
		JWTSecret:      env("JWT_SECRET", ""),
		JWTIssuer:      env("JWT_ISSUER", ""),
		JWTAudience:    env("JWT_AUDIENCE", ""),
		OracleDSN:      env("ORACLE_DSN", ""),
		OracleUser:     env("ORACLE_USER", ""),
		OraclePassword: env("ORACLE_PASSWORD", ""),
		GrainFunction: env(
			"ORACLE_GRAIN_NAME_FUNCTION",
			"B_TB0",
		),
		Origins: make(map[string]Origin),
	}

	mode := OracleConnectionMode(strings.ToUpper(
		env("ORACLE_CONNECTION_MODE", ""),
	))

	switch mode {
	case OracleDirect, OracleDBLink:
		c.OracleConnectionMode = mode
	default:
		return c, fmt.Errorf(
			"ORACLE_CONNECTION_MODE must be DIRECT or DBLINK",
		)
	}

	var err error

	if c.Empresa, err = num("ORACLE_EMPRESA", 1); err != nil {
		return c, err
	}
	if c.MaxOpen, err = num("ORACLE_MAX_OPEN_CONNS", 5); err != nil {
		return c, err
	}
	if c.MaxIdle, err = num("ORACLE_MAX_IDLE_CONNS", 2); err != nil {
		return c, err
	}

	var n int

	if n, err = num("ORACLE_CONN_MAX_LIFETIME_MINUTES", 30); err != nil {
		return c, err
	}
	c.ConnLifetime = time.Duration(n) * time.Minute

	if n, err = num("QUERY_TIMEOUT_SECONDS", 90); err != nil {
		return c, err
	}
	c.QueryTimeout = time.Duration(n) * time.Second

	if n, err = num("HTTP_READ_TIMEOUT_SECONDS", 15); err != nil {
		return c, err
	}
	c.ReadTimeout = time.Duration(n) * time.Second

	if n, err = num("HTTP_WRITE_TIMEOUT_SECONDS", 120); err != nil {
		return c, err
	}
	c.WriteTimeout = time.Duration(n) * time.Second

	if c.JWTSecret == "" ||
		strings.HasPrefix(c.JWTSecret, "REEMPLAZAR") ||
		c.OracleDSN == "" ||
		c.OracleUser == "" ||
		c.OraclePassword == "" ||
		strings.HasPrefix(c.OraclePassword, "REEMPLAZAR") {
		return c, fmt.Errorf(
			"configure JWT_SECRET and Oracle credentials",
		)
	}

	if !ident.MatchString(c.GrainFunction) {
		return c, fmt.Errorf(
			"invalid grain function identifier: %q",
			c.GrainFunction,
		)
	}

	enabledCount := 0

	for _, code := range []string{
		"NEC", "SL", "TIM", "SEL", "VIL", "AC", "CO",
	} {
		prefix := "ORIGIN_" + code + "_"

		enabled, err := strconv.ParseBool(
			env(prefix+"ENABLED", "false"),
		)
		if err != nil {
			return c, fmt.Errorf(
				"invalid %sENABLED: %w",
				prefix,
				err,
			)
		}

		o := Origin{
			Code:    code,
			Name:    env(prefix+"NAME", code),
			DBLink:  env(prefix+"DBLINK", ""),
			Enabled: enabled,
		}

		if enabled {
			enabledCount++

			if c.OracleConnectionMode == OracleDBLink &&
				o.DBLink == "" {
				return c, fmt.Errorf(
					"DBLINK required for %s in DBLINK mode",
					code,
				)
			}
		}

		if o.DBLink != "" && !ident.MatchString(o.DBLink) {
			return c, fmt.Errorf(
				"invalid DBLINK for %s: value=%q",
				code,
				o.DBLink,
			)
		}

		c.Origins[code] = o
	}

	if enabledCount == 0 {
		return c, fmt.Errorf(
			"at least one Oracle origin must be enabled",
		)
	}

	return c, nil
}
