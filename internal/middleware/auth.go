package middleware
import (
 "net/http"
 "strings"
 "github.com/gin-gonic/gin"
 "github.com/golang-jwt/jwt/v5"
)
func JWT(secret,issuer,audience string)gin.HandlerFunc{return func(c *gin.Context){
 h:=c.GetHeader("Authorization");parts:=strings.SplitN(h," ",2)
 if len(parts)!=2||!strings.EqualFold(parts[0],"Bearer")||strings.TrimSpace(parts[1])==""{c.AbortWithStatusJSON(http.StatusUnauthorized,gin.H{"error":"unauthorized","message":"Bearer token required"});return}
 opts:=[]jwt.ParserOption{jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),jwt.WithExpirationRequired()}
 if issuer!=""{opts=append(opts,jwt.WithIssuer(issuer))};if audience!=""{opts=append(opts,jwt.WithAudience(audience))}
 claims:=jwt.MapClaims{}
 token,err:=jwt.ParseWithClaims(parts[1],claims,func(t *jwt.Token)(any,error){return []byte(secret),nil},opts...)
 if err!=nil||!token.Valid{c.AbortWithStatusJSON(http.StatusUnauthorized,gin.H{"error":"unauthorized","message":"Invalid or expired token"});return}
 c.Set("jwtClaims",claims);c.Next()
}}
func CORS(origins string)gin.HandlerFunc{
 allowed:=map[string]bool{};for _,v:=range strings.Split(origins,","){v=strings.TrimSpace(v);if v!=""{allowed[v]=true}}
 return func(c *gin.Context){origin:=c.GetHeader("Origin");if origin!=""&&allowed[origin]{h:=c.Writer.Header();h.Set("Access-Control-Allow-Origin",origin);h.Set("Vary","Origin");h.Set("Access-Control-Allow-Methods","GET, OPTIONS");h.Set("Access-Control-Allow-Headers","Authorization, Content-Type, Accept");h.Set("Access-Control-Max-Age","600")}
 if c.Request.Method==http.MethodOptions {if origin!=""&&!allowed[origin]{c.AbortWithStatus(http.StatusForbidden)}else{c.AbortWithStatus(http.StatusNoContent)};return};c.Next()
}}
