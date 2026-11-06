# Despliegue TEST — Oracle Instant Client

`godror` utiliza OCI (CGO). Instalar Oracle Instant Client Basic compatible en el servidor en la ruta indicada por `ORACLE_CLIENT_HOST_PATH` (por defecto `/opt/oracle/instantclient`). Docker Compose la monta como `/opt/oracle/instantclient` y configura `LD_LIBRARY_PATH`. Verificar que `libclntsh.so` y dependencias estén presentes.

`cp .env.example .env`, reemplazar secretos, DBLINK reales y ejecutar `docker compose up -d --build`. El puerto se publica solo en `127.0.0.1`, detrás de NGINX.

**Pendientes de validación en ACA:** DBLINK, permisos, función `B_TB0` disponible en conexión central, compatibilidad JWT HS256 y equivalencia de resultados SQL.

El proyecto requiere `go mod tidy` con acceso a dependencias para generar `go.sum` antes del build.
