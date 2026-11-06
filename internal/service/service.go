package service
import (
 "context"
 "fmt"
 "sort"
 "sync"
 "time"
 "aca/existencias-puerto/internal/config"
 "aca/existencias-puerto/internal/domain"
 "aca/existencias-puerto/internal/repository"
)
type Service struct { Config config.Config; Repo *repository.Oracle }
func (s Service) Origins() []config.Origin { out:=[]config.Origin{};for _,o:=range s.Config.Origins {if o.Enabled {o.DBLink="";out=append(out,o)}};sort.Slice(out,func(i,j int)bool{return out[i].Code<out[j].Code});return out }
func(s Service) One(ctx context.Context,code string,f domain.Filter)([]domain.Existencia,error){o,ok:=s.Config.Origins[code];if !ok||!o.Enabled{return nil,fmt.Errorf("unknown or disabled origin")};ctx,cancel:=context.WithTimeout(ctx,s.Config.QueryTimeout);defer cancel();return s.Repo.Query(ctx,o,f)}
type Result struct {Origen string; Data []domain.Existencia; Err error}
func(s Service) Many(ctx context.Context,codes []string,f domain.Filter)[]Result {
 results:=make([]Result,len(codes));sem:=make(chan struct{},3);var wg sync.WaitGroup
 for i,code:=range codes {wg.Add(1);go func(i int,code string){defer wg.Done();select{case sem<-struct{}{}:defer func(){<-sem}();case <-ctx.Done():results[i]=Result{Origen:code,Err:ctx.Err()};return};d,e:=s.One(ctx,code,f);results[i]=Result{Origen:code,Data:d,Err:e}}(i,code)}
 wg.Wait();return results
}
func ParseDates(day,from,to string)(domain.Filter,error){f:=domain.Filter{};if day!=""{if from!=""||to!=""{return f,fmt.Errorf("fecha cannot be combined with range")};from,to=day,day};if from==""||to==""{return f,fmt.Errorf("fecha or fechaDesde and fechaHasta required")};a,e:=time.Parse("2006-01-02",from);if e!=nil{return f,fmt.Errorf("invalid fechaDesde")};b,e:=time.Parse("2006-01-02",to);if e!=nil{return f,fmt.Errorf("invalid fechaHasta")};if b.Before(a){return f,fmt.Errorf("fechaHasta before fechaDesde")};f.Desde=from;f.Hasta=to;return f,nil}
