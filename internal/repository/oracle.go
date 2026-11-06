package repository

import (
 "context"
 "database/sql"
 "fmt"
 "strings"
 "time"
 _ "github.com/godror/godror"
 "aca/existencias-puerto/internal/config"
 "aca/existencias-puerto/internal/domain"
)
type Oracle struct { DB *sql.DB; GrainFunction string; Empresa int }
func Open(c config.Config)(*Oracle,error){
 db,e:=sql.Open("godror",fmt.Sprintf(`user="%s" password="%s" connectString="%s"`,c.OracleUser,c.OraclePassword,c.OracleDSN));if e!=nil{return nil,e}
 db.SetMaxOpenConns(c.MaxOpen);db.SetMaxIdleConns(c.MaxIdle);db.SetConnMaxLifetime(c.ConnLifetime)
 ctx,cancel:=context.WithTimeout(context.Background(),15*time.Second);defer cancel()
 if e=db.PingContext(ctx);e!=nil{db.Close();return nil,fmt.Errorf("oracle central ping failed: %w",e)}
 return &Oracle{DB:db,GrainFunction:c.GrainFunction,Empresa:c.Empresa},nil
}
func table(name,link string) string {if link=="" {return name};return name+"@"+link}
func (r *Oracle) Query(ctx context.Context, o config.Origin,f domain.Filter)([]domain.Existencia,error){
 // Solo identificadores validados al cargar configuracion; nunca DBLINK desde HTTP.
 mp,mc:=table("MPESAJE",o.DBLink),table("MCUENTA",o.DBLink)
 query:=fmt.Sprintf(`WITH base AS (
 SELECT a.grano, a.tipomerca, a.fecsal, a.codmovie,
 CASE WHEN a.fecsal < DATE '2009-01-13' THEN a.cuenta
 WHEN a.fecsal >= DATE '2009-01-13' AND a.codimovi IN (63,66) AND a.codmovie=0 THEN a.cuenta
 WHEN a.fecsal >= DATE '2009-01-13' AND a.codimovi IN (63,66) AND a.codmovie=1 THEN a.vendedor
 ELSE a.cuenta END AS comcuenta,
 (a.totneto*a.resulcodi) AS importe
 FROM %s a
 WHERE a.empresa=:empresa AND a.tiporoma<>21
 AND (a.status=200 OR a.uvalueembar IS NOT NULL)
 AND a.fecsal < :hasta_exclusiva
 AND (:tipo_merca IS NULL OR a.tipomerca=:tipo_merca)
 AND (:grano IS NULL OR a.grano=:grano)
 ), agrupado AS (
 SELECT grano,comcuenta,tipomerca,
 SUM(CASE WHEN fecsal<:desde THEN importe ELSE 0 END) anterior,
 SUM(CASE WHEN fecsal>=:desde AND codmovie=0 THEN importe ELSE 0 END) entradas,
 SUM(CASE WHEN fecsal>=:desde AND codmovie=1 THEN importe ELSE 0 END) salidas
 FROM base WHERE comcuenta IS NOT NULL AND (:cuenta IS NULL OR comcuenta=:cuenta)
 GROUP BY grano,comcuenta,tipomerca
 )
 SELECT TO_CHAR(g.grano), %s('GRANO',LPAD(g.grano,3,' '),'GR2'),
 TO_CHAR(g.comcuenta), b.nombre, TO_CHAR(g.tipomerca),
 g.anterior,g.entradas,g.salidas,(g.anterior+g.entradas+g.salidas)
 FROM agrupado g LEFT JOIN %s b ON b.empresa=:empresa AND b.cuenta=g.comcuenta
 WHERE (g.anterior+g.entradas+g.salidas)<>0
 ORDER BY g.tipomerca,g.grano,g.comcuenta`,mp,r.GrainFunction,mc)
 start,e:=time.Parse("2006-01-02",f.Desde);if e!=nil{return nil,e};end,e:=time.Parse("2006-01-02",f.Hasta);if e!=nil{return nil,e}
 var grano,cuenta,merca any
 if strings.TrimSpace(f.Grano)!=""{grano=f.Grano};if strings.TrimSpace(f.Cuenta)!=""{cuenta=f.Cuenta};if strings.TrimSpace(f.TipoMerca)!=""{merca=f.TipoMerca}
 rows,e:=r.DB.QueryContext(ctx,query,sql.Named("empresa",r.Empresa),sql.Named("desde",start),sql.Named("hasta_exclusiva",end.AddDate(0,0,1)),sql.Named("grano",grano),sql.Named("cuenta",cuenta),sql.Named("tipo_merca",merca))
 if e!=nil{return nil,e};defer rows.Close()
 out:=make([]domain.Existencia,0)
 for rows.Next(){var x domain.Existencia;var name,account sql.NullString;var a,b,c,d sql.NullFloat64
  if e=rows.Scan(&x.Grano,&name,&x.ComCuenta,&account,&x.TipoMerca,&a,&b,&c,&d);e!=nil{return nil,e}
  x.NomGrano=name.String;x.ComNombre=account.String;x.Anterior=a.Float64;x.Entradas=b.Float64;x.Salidas=c.Float64;x.Existencia=d.Float64;out=append(out,x)
 }
 return out,rows.Err()
}
