-- ============================================================
-- BTR 202604 — Esquema MySQL
-- Planilla Unica de Pagos · MINEDU
-- ============================================================

CREATE DATABASE IF NOT EXISTS btr_db
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE btr_db;

DROP TABLE IF EXISTS records;

CREATE TABLE records (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,

  -- Identificacion
  PERPAGO      VARCHAR(16)  DEFAULT '',
  MODULAR      VARCHAR(32)  DEFAULT '',
  SECUENCIAL   VARCHAR(32)  DEFAULT '',
  NDOCUMENTO   VARCHAR(24)  DEFAULT '',

  -- Persona
  PATERNO      VARCHAR(64)  DEFAULT '',
  MATERNO      VARCHAR(64)  DEFAULT '',
  NOMBRES      VARCHAR(96)  DEFAULT '',
  SEXO         VARCHAR(16)  DEFAULT '',
  FNACIMIENT   VARCHAR(32)  DEFAULT '',
  FINGRESO     VARCHAR(32)  DEFAULT '',

  -- Ubicacion / Institucion
  OFICINA      VARCHAR(128) DEFAULT '',
  NOMBRE_IE    VARCHAR(255) DEFAULT '',
  COD_IE       VARCHAR(32)  DEFAULT '',

  -- Puesto
  DTSERVIDOR   VARCHAR(64)  DEFAULT '',
  DES_CARGO    VARCHAR(128) DEFAULT '',
  COD_CARGO    VARCHAR(32)  DEFAULT '',
  REGLAB       VARCHAR(128) DEFAULT '',
  JORLABORAL   VARCHAR(16)  DEFAULT '',
  PLAZA        VARCHAR(32)  DEFAULT '',
  DESC_PLAZA   VARCHAR(128) DEFAULT '',
  DPLANILLA    VARCHAR(64)  DEFAULT '',
  DSITUACION   VARCHAR(32)  DEFAULT '',

  -- Montos
  THABER       DECIMAL(12,2) DEFAULT 0.00,
  TDESCUENTO   DECIMAL(12,2) DEFAULT 0.00,
  TLIQUIDO     DECIMAL(12,2) DEFAULT 0.00,

  -- Prevision
  CREG_PENS    VARCHAR(16)  DEFAULT '',
  CUSSP        VARCHAR(32)  DEFAULT '',

  PRIMARY KEY (id),

  -- Indices para filtros exactos
  KEY idx_dni       (NDOCUMENTO),
  KEY idx_oficina   (OFICINA),
  KEY idx_paterno   (PATERNO),
  KEY idx_materno   (MATERNO),
  KEY idx_nombres   (NOMBRES),
  KEY idx_cargo     (DES_CARGO),
  KEY idx_reglab    (REGLAB),
  KEY idx_tservidor (DTSERVIDOR),
  KEY idx_ie        (NOMBRE_IE(64)),
  KEY idx_liq       (TLIQUIDO)

  -- Los indices FULLTEXT se crean DESPUES de la carga (mucho mas rapido):
  --   ft_search -> todos los campos buscables
  --   ft_names  -> solo PATERNO, MATERNO, NOMBRES, NDOCUMENTO (para ordenar
  --                por relevancia real; ver comentario en scripts/load-data.sh)
  -- Ver scripts/load-data.sh
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  ROW_FORMAT=DYNAMIC;