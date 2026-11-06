# ACA Existencias en Puerto — Backend TEST

API Go/Gin con CORS, JWT HS256, una conexión Oracle central y DBLINK por origen.

## Configuración

1. `cp .env.example .env` y completar la conexión Oracle central y JWT_SECRET.
2. Reemplazar **todos** los `ORIGIN_*_DBLINK` por los nombres reales existentes en Oracle. Los nombres de ejemplo NO son reales.
3. `ORIGIN_AC_ENABLED` y `ORIGIN_CO_ENABLED` son opcionales. Si se habilitan con DBLINK vacío se consultan tablas locales de la conexión central.
4. Confirmar que la función `B_TB0` esté accesible localmente desde la conexión central. Si no, adaptar la resolución de nombre del grano (función remota/sinónimo) antes de probar.
5. `docker compose up -d --build`; `docker compose logs -f api`.
6. `curl http://127.0.0.1:9042/health`.

## Endpoints

- `GET /health` (sin JWT)
- `GET /api/v1/origenes` (JWT)
- `GET /api/v1/existencias?origen=NEC&fecha=2026-10-15` (JWT)
- `GET /api/v1/existencias/consolidado?origenes=NEC,SL&fechaDesde=2026-10-01&fechaHasta=2026-10-15` (JWT)
- Filtros opcionales: `grano`, `cuenta`, `tipoMerca`.

## JWT

Valida **HS256** con `JWT_SECRET` (firma, no descifrado), `exp` obligatorio, y `iss`/`aud` si están configurados. Debe ser compatible con el emisor JWT de ACA. No implementa generación de tokens.

## Consideraciones importantes

- **Pendiente de validar en Oracle:** nombres DBLINK, privilegios de acceso, disponibilidad B_TB0 y equivalencia de saldos con el SELECT de referencia.
- El SELECT original usa `TOTNETO * RESULCODI`; no invierte signos desde Go.
- `anterior` = movimientos antes de fechaDesde; entradas/salidas = fechaDesde a fechaHasta inclusive.
- La API excluye existencias finales iguales a cero, igual que el SQL original.
- El pool central está limitado por `ORACLE_MAX_OPEN_CONNS`; consolidado máximo 3 consultas simultáneas.
- No se ha verificado compilación ni conectividad con las bases de ACA en este entorno.
- La imagen requiere Oracle Instant Client Basic disponible en el host en `ORACLE_CLIENT_HOST_PATH` y montado en el contenedor. Ver `docs/DEPLOY.md`.
