# BTR 202604 · Buscador Especializado

Buscador de la planilla única de pagos del **BTR 202604** (MINEDU, Perú).
Interfaz moderna azul/celeste con búsqueda instantánea sobre **686.686** registros.

## Arquitectura

```
┌──────────────┐    /api/*    ┌──────────────┐    SQL    ┌──────────────┐
│  Frontend    │ ───────────► │   Backend    │ ────────► │   MySQL 8    │
│ React+Nginx  │ ◄─────────── │     Go       │ ◄──────── │ 686.686 filas│
└──────────────┘   JSON      └──────────────┘           └──────────────┘
```

| Capa        | Tecnología                              | Puerto |
|-------------|-----------------------------------------|-------|
| Frontend    | React 18 + Vite 6, servido por Nginx     | 3000  |
| Backend     | Go 1.24 (`database/sql`, driver MySQL)   | 8080  |
| Base de datos | MySQL 8.0 (InnoDB)                     | 3307  |

Orquestado con **Docker Compose**: `mysql` → `backend` (espera a MySQL sano) →
`frontend` (espera a backend sano).

## Puesta en marcha

```bash
cd btr_docker
docker compose up -d --build
```

- Web: http://localhost:3000
- API: http://localhost:8080/api/... (también vía proxy http://localhost:3000/api/...)

> Los datos ya vienen cargados en el volumen `mysql_data`. Si es la primera vez
> (volumen vacío) sigue el paso de *Carga de datos* más abajo.

Variables de entorno (opcional, ver `.env.example`): `MYSQL_ROOT_PASSWORD`,
`MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_PORT`, `BACKEND_PORT`, `FRONTEND_PORT`.

## Carga de datos (solo primera vez)

El archivo fuente es un **`.xlsb`** (Excel binario) de ~137 MB con 686.686 filas.
El pipeline:

1. **Extraer** el `.xlsb` a TSV (requiere `pyxlsb`):
   ```bash
   python scripts/export_tsv.py            # → db/seed/btr_data.tsv (~160 MB)
   ```
2. **Cargar** a MySQL y crear índices:
   ```bash
   ./scripts/load-data.sh                 # LOAD DATA INFILE + índices (~3-4 min)
   ```

La semilla (`db/seed/btr_data.tsv`) ya está incluida, así que el paso 1 se puede
omitir si ya la tienes.

## Importar datos (botón en la UI)

Hay un botón **"Importar datos"** en la cabecera de la web. Permite subir un
archivo nuevo (`.xlsb` o `.tsv`) y **reemplazar toda la planilla** sin tocar
Docker a mano.

**⚠️ Operación destructiva:** vacía y reemplaza los 686.686 registros. No hay
"deshacer"; el único respaldo es el volumen `mysql_data` y la semilla en
`db/seed/`.

Funcionamiento (no bloquea la UI):

1. El frontend sube el archivo (barra de progreso de subida vía XHR) y muestra
   el avance del servidor sondeando `GET /api/import` cada 1 s.
2. El backend lanza la importación en segundo plano (`202 Accepted`):
   - Si es `.xlsb` → `python3 /app/xlsb_to_tsv.py` lo convierte a TSV canónico
     (27 columnas, mismo orden que `load-data.sh`).
   - Quita los 8 índices pesados (`ft_search` + 7 compuestos) para acelerar
     la carga y hace `TRUNCATE TABLE records`.
   - `LOAD DATA LOCAL INFILE` del TSV (el backend lee el archivo y lo transmite
     al servidor; por eso el DSN lleva `allowAllFiles=true`).
   - Recrea los 8 índices (FULLTEXT amplio + 7 compuestos por prefijo).
   - `COUNT(*)` final y el frontend refresca estadísticas y filtros.
3. Formatos aceptados: `.tsv` (columnas canónicas separadas por TAB, con
   cabecera) o `.xlsb` (Excel binario; requiere `pyxlsb` en el runtime).

Límites: archivo máximo **512 MB** (`nginx client_max_body_size` + límite del
backend). Una sola importación a la vez; si ya hay una en curso, el endpoint
responde `409`.

## API

| Endpoint             | Descripción                                         |
|----------------------|-----------------------------------------------------|
| `GET /api/search`    | Búsqueda. `q`, `dni`, `paterno`, `materno`, `nombres`, `oficina`, `tservidor`, `cargo`, `reglab`, `secuencial`, `modular`, `cod_ie`, `cod_cargo`, `cussp`, `plaza`, `sexo`, `situacion`, `limit`, `offset`. Los campos de código usan coincidencia por prefijo; `sexo`/`situacion` son exactos. |
| `GET /api/stats`     | Totales globales (registros, UGEL, IE, cargos).     |
| `GET /api/distinct`  | Valores distintos de un campo para los filtros.      |
| `GET /api/health`    | Health check.                                       |
| `POST /api/import`   | **Importar datos** (botón "Importar datos"). Recibe un `.tsv` o `.xlsb` por multipart (`file`) y reemplaza toda la tabla. Responde `202` y se sondea con `GET /api/import`. |
| `GET /api/import`    | Estado de la importación en curso (stage, filas, error). |

Ejemplo:
```
GET /api/search?q=HUAMAN&limit=30&offset=0
→ { total, results: [ {PATERNO, MATERNO, NOMBRES, NDOCUMENTO, OFICINA, ...} ] }
```

## Interfaz

- **Vista Tarjetas / Tabla**: alterna con el conmutador superior.
- **Detalle al hacer clic**: al pulsar cualquier tarjeta (o fila de la tabla) se
  abre un modal con **todos los 27 campos** del registro, agrupados
  (Identificación, Datos personales, Datos laborales, Remuneraciones). Los montos
  se muestran en soles y las fechas de nacimiento/ingreso (`FNACIMIENT`,
  `FINGRESO`) —que vienen como número serial de Excel— se convierten a fecha
  legible. Cerrar con la **×**, el botón *Cerrar* o **Esc**.

## Rendimiento de la búsqueda

El requisito era *"que sea rápido"*. Tras medir sobre los datos reales:

- **Búsqueda por nombre/DNI (1 letra a medida que se teclea): 12–80 ms.**
- **Varios tokens** (`HUAMAN PEREZ`): ramas por prefijo con filtros, ~100–600 ms.
- **Respaldo FULLTEXT** para términos que no están al inicio del campo
  (`LA MAR`, `DE LA CRUZ`): ~150–300 ms.

Claves del diseño:

1. **Ramas por prefijo sobre índices compuestos** (la vía rápida). En lugar de
   ordenar decenas de miles de filas (FULLTEXT costaba ~460 ms), se recorre un
   índice B-tree ya ordenado y se detiene en el `LIMIT`: **~0,6 ms** por página.
   Siete ramas (`PATERNO`, `MATERNO`, `NOMBRES`, `NDOCUMENTO`, `DES_CARGO`,
   `OFICINA`, `NOMBRE_IE`), disjuntas entre sí para paginar sin duplicados.
2. **FULLTEXT `ft_search`** solo como respaldo (términos intermedios). Por encima
   de 20.000 coincidencias se omite el `ORDER BY` de relevancia (costaba 4 s).
3. `innodb_ft_min_token_size=2`, `innodb_ft_enable_stopword=OFF` y
   `sort_buffer_size=8M` se pasan por **línea de comando** en el compose, porque
   MySQL ignora los `.cnf` montados desde Windows/Mac ("world-writable").

## Estructura

```
btr_docker/
├── docker-compose.yml          # orquestación de los 3 servicios
├── db/
│   ├── init/01-schema.sql      # esquema de la tabla
│   ├── mysql.cnf               # (no se monta; histórico)
│   └── seed/btr_data.tsv       # semilla 686.686 filas
├── scripts/
│   ├── export_tsv.py           # .xlsb → .tsv
│   └── load-data.sh            # carga + creación de índices
├── backend/                    # Go: db.go, main.go, import.go, Dockerfile, go.mod
│   └── xlsb_to_tsv.py          # conversor .xlsb → TSV canónico (27 columnas)
└── frontend/                   # React: src/, nginx.conf, Dockerfile, vite.config.js
    └── src/components/ImportModal.jsx  # modal "Importar datos"
```

## Notas

- El archivo fuente original era `BTR_202604.xlsb` (en `C:\github\workbuddy\`).
- La búsqueda y orden son **insensibles a acentos y mayúsculas**
  (`utf8mb4_unicode_ci`).
- Algunas personas aparecen más de una vez (mismo DNI) porque figuran en dos
  UGEL distintos en la fuente; no es un error del buscador.
