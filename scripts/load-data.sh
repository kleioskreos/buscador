#!/usr/bin/env bash
# ============================================================
# Carga el TSV a MySQL y crea el indice FULLTEXT
# Uso: ./scripts/load-data.sh
# ============================================================
set -euo pipefail

CONTAINER="${MYSQL_CONTAINER:-btr-mysql}"
DB_USER="root"
DB_PASS="${MYSQL_ROOT_PASSWORD:-btr_root_2026}"
DB_NAME="btr_db"
TSV_IN_CONTAINER="/var/lib/mysql-files/btr_data.tsv"

echo "================================================"
echo "  Carga de datos BTR 202604 -> MySQL"
echo "================================================"

mysql_exec() {
  docker exec -i "$CONTAINER" mysql \
    -u"$DB_USER" -p"$DB_PASS" \
    --default-character-set=utf8mb4 \
    -D "$DB_NAME" "$@"
}

echo "[1/4] Verificando que MySQL responda..."
until docker exec "$CONTAINER" mysqladmin -u"$DB_USER" -p"$DB_PASS" ping --silent 2>/dev/null; do
  echo "  esperando MySQL..."
  sleep 2
done
echo "  MySQL OK"

echo "[2/4] Copiando TSV a /var/lib/mysql-files (requerido por secure_file_priv)..."
docker exec "$CONTAINER" test -f /seed/btr_data.tsv || {
  echo "  ERROR: no se encontro /seed/btr_data.tsv dentro del contenedor"
  echo "  Asegurate de montar ./db/seed en /seed"
  exit 1
}
docker exec "$CONTAINER" cp /seed/btr_data.tsv "$TSV_IN_CONTAINER"
docker exec "$CONTAINER" chmod 644 "$TSV_IN_CONTAINER"
echo "  TSV copiado ($(docker exec "$CONTAINER" du -h "$TSV_IN_CONTAINER" | cut -f1))"

echo "[3/4] Cargando datos (LOAD DATA INFILE)..."
mysql_exec <<'SQL'
SET SESSION sql_log_bin = 0;
SET SESSION foreign_key_checks = 0;
SET SESSION unique_checks = 0;
SET SESSION autocommit = 0;

LOAD DATA INFILE '/var/lib/mysql-files/btr_data.tsv'
INTO TABLE records
CHARACTER SET utf8mb4
FIELDS TERMINATED BY '\t'
ENCLOSED BY ''
ESCAPED BY ''
LINES TERMINATED BY '\n'
IGNORE 1 LINES
(PERPAGO, MODULAR, SECUENCIAL, NDOCUMENTO, PATERNO, MATERNO, NOMBRES,
 OFICINA, DTSERVIDOR, NOMBRE_IE, DES_CARGO, REGLAB, JORLABORAL, SEXO,
 DSITUACION, THABER, TDESCUENTO, TLIQUIDO, PLAZA, DESC_PLAZA, DPLANILLA,
 FNACIMIENT, FINGRESO, COD_IE, COD_CARGO, CREG_PENS, CUSSP);

COMMIT;
SELECT COUNT(*) AS total_cargado FROM records;
SQL
echo "  Datos cargados"

echo "[4/4] Creando indices FULLTEXT..."
mysql_exec <<'SQL'
-- Indice amplio: respaldo para terminos que NO estan al principio del campo
-- (p. ej. "LA MAR" dentro de OFICINA = "UGEL LA MAR", o "DE LA CRUZ").
ALTER TABLE records
  ADD FULLTEXT INDEX ft_search
  (NDOCUMENTO, PATERNO, MATERNO, NOMBRES, OFICINA, NOMBRE_IE,
   DES_CARGO, REGLAB, DTSERVIDOR, PLAZA, DESC_PLAZA);

-- Indices compuestos para la busqueda por prefijo (la via rapida del backend).
-- Permiten que "PATERNO LIKE 'hua%' ORDER BY PATERNO, MATERNO, NOMBRES LIMIT 30"
-- recorra el indice ya en orden y se detenga a los 30 primeros: 0.6 ms en lugar
-- de los ~460 ms que cuesta ordenar las 30.000 filas que devuelve FULLTEXT.
-- Cubren apellido paterno/materno, nombre, DNI y los campos institucionales
-- (cargo, UGEL, institucion) para que cualquier termino inicial sea rapido.
ALTER TABLE records
  ADD INDEX idx_pmn   (PATERNO, MATERNO, NOMBRES),
  ADD INDEX idx_mpn   (MATERNO, PATERNO, NOMBRES),
  ADD INDEX idx_npm   (NOMBRES, PATERNO, MATERNO),
  ADD INDEX idx_dni_n (NDOCUMENTO, PATERNO, MATERNO),
  ADD INDEX idx_cargo_n (DES_CARGO, PATERNO, MATERNO),
  ADD INDEX idx_ofic_n  (OFICINA, PATERNO, MATERNO),
  ADD INDEX idx_ie_n    (NOMBRE_IE, PATERNO, MATERNO);
SQL
echo "  Indices creados (ft_search + 7 indices compuestos por prefijo)"

echo ""
echo "================================================"
echo "  Carga completada"
echo "================================================"
mysql_exec -e "SELECT COUNT(*) AS total_registros FROM records;"
mysql_exec -e "SELECT COUNT(DISTINCT OFICINA) AS ugels, COUNT(DISTINCT NOMBRE_IE) AS instituciones, COUNT(DISTINCT DES_CARGO) AS cargos FROM records;"