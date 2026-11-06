package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"aca/existencias-puerto/internal/config"
	"aca/existencias-puerto/internal/domain"

	"github.com/godror/godror"
)

type Oracle struct {
	DB             *sql.DB
	GrainFunction  string
	Empresa        int
	ConnectionMode config.OracleConnectionMode
}

func Open(c config.Config) (*Oracle, error) {
	params := godror.ConnectionParams{
		CommonParams: godror.CommonParams{
			ConnectString: c.OracleDSN,
			Username:      c.OracleUser,
			Password:      godror.NewPassword(c.OraclePassword),
		},
	}

	db := sql.OpenDB(godror.NewConnector(params))

	db.SetMaxOpenConns(c.MaxOpen)
	db.SetMaxIdleConns(c.MaxIdle)
	db.SetConnMaxLifetime(c.ConnLifetime)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf(
			"oracle connection failed: %w",
			err,
		)
	}

	return &Oracle{
		DB:             db,
		GrainFunction:  c.GrainFunction,
		Empresa:        c.Empresa,
		ConnectionMode: c.OracleConnectionMode,
	}, nil
}

func (r *Oracle) tableName(
	name string,
	o config.Origin,
) (string, error) {
	switch r.ConnectionMode {
	case config.OracleDirect:
		return name, nil

	case config.OracleDBLink:
		if o.DBLink == "" {
			return "", fmt.Errorf(
				"DBLINK not configured for origin %s",
				o.Code,
			)
		}
		return name + "@" + o.DBLink, nil

	default:
		return "", fmt.Errorf(
			"invalid Oracle connection mode: %s",
			r.ConnectionMode,
		)
	}
}

func (r *Oracle) Query(
	ctx context.Context,
	o config.Origin,
	f domain.Filter,
) ([]domain.Existencia, error) {

	if !o.Enabled {
		return nil, fmt.Errorf(
			"origin %s is disabled",
			o.Code,
		)
	}

	mp, err := r.tableName("MPESAJE", o)
	if err != nil {
		return nil, err
	}

	mc, err := r.tableName("MCUENTA", o)
	if err != nil {
		return nil, err
	}

	// Los nombres de tablas y DBLINK provienen de configuración
	// validada, nunca de parámetros HTTP.
	query := fmt.Sprintf(`
WITH base AS (
	SELECT
		a.grano,
		a.tipomerca,
		a.fecsal,
		a.codmovie,
		CASE
			WHEN a.fecsal < DATE '2009-01-13'
				THEN a.cuenta
			WHEN a.fecsal >= DATE '2009-01-13'
				AND a.codimovi IN (63,66)
				AND a.codmovie = 0
				THEN a.cuenta
			WHEN a.fecsal >= DATE '2009-01-13'
				AND a.codimovi IN (63,66)
				AND a.codmovie = 1
				THEN a.vendedor
			ELSE a.cuenta
		END AS comcuenta,
		(a.totneto * a.resulcodi) AS importe
	FROM %s a
	WHERE a.empresa = :empresa
		AND a.tiporoma <> 21
		AND (a.status = 200 OR a.uvalueembar IS NOT NULL)
		AND a.fecsal < :hasta_exclusiva
		AND (:tipo_merca IS NULL OR a.tipomerca = :tipo_merca)
		AND (:grano IS NULL OR a.grano = :grano)
),
agrupado AS (
	SELECT
		grano,
		comcuenta,
		tipomerca,
		SUM(
			CASE
				WHEN fecsal < :desde THEN importe
				ELSE 0
			END
		) AS anterior,
		SUM(
			CASE
				WHEN fecsal >= :desde AND codmovie = 0
					THEN importe
				ELSE 0
			END
		) AS entradas,
		SUM(
			CASE
				WHEN fecsal >= :desde AND codmovie = 1
					THEN importe
				ELSE 0
			END
		) AS salidas
	FROM base
	WHERE comcuenta IS NOT NULL
		AND (:cuenta IS NULL OR comcuenta = :cuenta)
	GROUP BY grano, comcuenta, tipomerca
)
SELECT
	TO_CHAR(g.grano),
	%s('GRANO', LPAD(g.grano, 3, ' '), 'GR2'),
	TO_CHAR(g.comcuenta),
	b.nombre,
	TO_CHAR(g.tipomerca),
	g.anterior,
	g.entradas,
	g.salidas,
	(g.anterior + g.entradas + g.salidas)
FROM agrupado g
LEFT JOIN %s b
	ON b.empresa = :empresa
	AND b.cuenta = g.comcuenta
WHERE (g.anterior + g.entradas + g.salidas) <> 0
ORDER BY
	g.tipomerca,
	g.grano,
	g.comcuenta
`, mp, r.GrainFunction, mc)

	start, err := time.Parse("2006-01-02", f.Desde)
	if err != nil {
		return nil, fmt.Errorf("invalid fechaDesde: %w", err)
	}

	end, err := time.Parse("2006-01-02", f.Hasta)
	if err != nil {
		return nil, fmt.Errorf("invalid fechaHasta: %w", err)
	}

	if end.Before(start) {
		return nil, fmt.Errorf(
			"fechaHasta cannot be earlier than fechaDesde",
		)
	}

	var grano, cuenta, merca any

	if strings.TrimSpace(f.Grano) != "" {
		grano = f.Grano
	}
	if strings.TrimSpace(f.Cuenta) != "" {
		cuenta = f.Cuenta
	}
	if strings.TrimSpace(f.TipoMerca) != "" {
		merca = f.TipoMerca
	}

	rows, err := r.DB.QueryContext(
		ctx,
		query,
		sql.Named("empresa", r.Empresa),
		sql.Named("desde", start),
		sql.Named("hasta_exclusiva", end.AddDate(0, 0, 1)),
		sql.Named("grano", grano),
		sql.Named("cuenta", cuenta),
		sql.Named("tipo_merca", merca),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"oracle query failed (mode=%s, origin=%s): %w",
			r.ConnectionMode,
			o.Code,
			err,
		)
	}
	defer rows.Close()

	out := make([]domain.Existencia, 0)

	for rows.Next() {
		var x domain.Existencia
		var name, account sql.NullString
		var anterior, entradas, salidas, existencia sql.NullFloat64

		err = rows.Scan(
			&x.Grano,
			&name,
			&x.ComCuenta,
			&account,
			&x.TipoMerca,
			&anterior,
			&entradas,
			&salidas,
			&existencia,
		)
		if err != nil {
			return nil, err
		}

		x.NomGrano = name.String
		x.ComNombre = account.String
		x.Anterior = anterior.Float64
		x.Entradas = entradas.Float64
		x.Salidas = salidas.Float64
		x.Existencia = existencia.Float64

		out = append(out, x)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
