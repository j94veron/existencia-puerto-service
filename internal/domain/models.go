package domain

type Existencia struct {
 Grano string `json:"grano"`
 NomGrano string `json:"nomGrano"`
 ComCuenta string `json:"comCuenta"`
 ComNombre string `json:"comNombre"`
 TipoMerca string `json:"tipoMerca"`
 Anterior float64 `json:"anterior"`
 Entradas float64 `json:"entradas"`
 Salidas float64 `json:"salidas"`
 Existencia float64 `json:"existencia"`
}
type Consolidada struct { Origen string `json:"origen"`; Existencia }
type OriginError struct { Origen string `json:"origen"`; Codigo string `json:"codigo"`; Mensaje string `json:"mensaje"` }
type Filter struct { Desde, Hasta string; Grano, Cuenta, TipoMerca string }
