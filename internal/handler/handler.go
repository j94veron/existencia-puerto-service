package handler
import (
 "net/http"
 "strings"
 "github.com/gin-gonic/gin"
 "aca/existencias-puerto/internal/domain"
 "aca/existencias-puerto/internal/service"
)
type Handler struct {Svc service.Service}
func(h Handler) filters(c *gin.Context)(domain.Filter,bool){f,e:=service.ParseDates(c.Query("fecha"),c.Query("fechaDesde"),c.Query("fechaHasta"));if e!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"bad_request","message":e.Error()});return f,false};f.Grano=c.Query("grano");f.Cuenta=c.Query("cuenta");f.TipoMerca=c.Query("tipoMerca");return f,true}
func(h Handler) Origins(c *gin.Context){c.JSON(200,gin.H{"data":h.Svc.Origins()})}
func(h Handler) One(c *gin.Context){f,ok:=h.filters(c);if !ok{return};code:=strings.ToUpper(strings.TrimSpace(c.Query("origen")));o,exists:=h.Svc.Config.Origins[code];if !exists||!o.Enabled{c.JSON(404,gin.H{"error":"origin_not_found"});return};d,e:=h.Svc.One(c.Request.Context(),code,f);if e!=nil{c.JSON(502,gin.H{"error":"origin_unavailable","message":"No fue posible consultar el origen"});return};c.JSON(200,gin.H{"origen":code,"nombreOrigen":o.Name,"fechaDesde":f.Desde,"fechaHasta":f.Hasta,"data":d,"totalRegistros":len(d)})}
func(h Handler) Consolidated(c *gin.Context){f,ok:=h.filters(c);if !ok{return};codes:=[]string{};raw:=strings.TrimSpace(c.Query("origenes"));if raw==""{for _,o:=range h.Svc.Origins(){codes=append(codes,o.Code)}}else{seen:=map[string]bool{};for _,part:=range strings.Split(raw,","){code:=strings.ToUpper(strings.TrimSpace(part));o,exists:=h.Svc.Config.Origins[code];if code==""||!exists||!o.Enabled{c.JSON(400,gin.H{"error":"invalid_origin","message":"Origen no habilitado: "+code});return};if !seen[code]{codes=append(codes,code);seen[code]=true}}}
 if len(codes)==0{c.JSON(400,gin.H{"error":"no_origins"});return}
 data:=[]domain.Consolidada{};errs:=[]domain.OriginError{};success:=0
 for _,r:=range h.Svc.Many(c.Request.Context(),codes,f){if r.Err!=nil{errs=append(errs,domain.OriginError{Origen:r.Origen,Codigo:"ORIGIN_UNAVAILABLE",Mensaje:"Origen temporalmente no disponible"});continue};success++;for _,x:=range r.Data{data=append(data,domain.Consolidada{Origen:r.Origen,Existencia:x})}}
 status:=200;if success==0{status=502};c.JSON(status,gin.H{"fechaDesde":f.Desde,"fechaHasta":f.Hasta,"origenesConsultados":len(codes),"origenesExitosos":success,"origenesFallidos":len(errs),"data":data,"errores":errs})
}
