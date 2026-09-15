package main

// db.go — Conexion a MySQL y construccion de consultas

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DBPool es el pool global de conexiones
var DBPool *sql.DB

// ---------- Configuracion ----------

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	PortHTTP string
}

func LoadConfig() *Config {
	c := &Config{
		Host:     getEnv("MYSQL_HOST", "mysql"),
		Port:     getEnv("MYSQL_PORT", "3306"),
		User:     getEnv("MYSQL_USER", "btr_user"),
		Password: getEnv("MYSQL_PASSWORD", "btr_pass_2026"),
		DBName:   getEnv("MYSQL_DATABASE", "btr_db"),
		PortHTTP: getEnv("PORT", "8080"),
	}
	return c
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (c *Config) DSN() string {
	// allowAllFiles: necesario para LOAD DATA LOCAL INFILE (el backend lee el
	// archivo y lo transmite al servidor). Ver backend/import.go.
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_unicode_ci&allowAllFiles=true",
		c.User, c.Password, c.Host, c.Port, c.DBName)
}

// InitDB inicializa el pool y espera a que MySQL este disponible
func InitDB(c *Config) error {
	var err error
	DBPool, err = sql.Open("mysql", c.DSN())
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}

	// Pool settings
	DBPool.SetMaxOpenConns(50)
	DBPool.SetMaxIdleConns(25)
	DBPool.SetConnMaxLifetime(5 * time.Minute)
	DBPool.SetConnMaxIdleTime(2 * time.Minute)

	// Reintentos: MySQL tarda en arrancar dentro de Docker
	maxRetries := 60
	for i := 1; i <= maxRetries; i++ {
		if err = DBPool.Ping(); err == nil {
			log.Printf("[db] conectado a MySQL %s:%s/%s", c.Host, c.Port, c.DBName)
			return nil
		}
		if i == 1 || i%10 == 0 {
			log.Printf("[db] esperando MySQL (intento %d/%d): %v", i, maxRetries, err)
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("no se pudo conectar a MySQL tras %d intentos: %w", maxRetries, err)
}

// ---------- Modelos ----------

// Record es una fila de la planilla
type Record struct {
	ID         int64   `json:"id"`
	PERPAGO    string  `json:"PERPAGO"`
	MODULAR    string  `json:"MODULAR"`
	SECUENCIAL string  `json:"SECUENCIAL"`
	NDOCUMENTO string  `json:"NDOCUMENTO"`
	PATERNO    string  `json:"PATERNO"`
	MATERNO    string  `json:"MATERNO"`
	NOMBRES    string  `json:"NOMBRES"`
	OFICINA    string  `json:"OFICINA"`
	DTSERVIDOR string  `json:"DTSERVIDOR"`
	NOMBREIE   string  `json:"NOMBRE_IE"`
	DESCARGO   string  `json:"DES_CARGO"`
	REGLAB     string  `json:"REGLAB"`
	JORLABORAL string  `json:"JORLABORAL"`
	SEXO       string  `json:"SEXO"`
	DSITUACION string  `json:"DSITUACION"`
	THABER     float64 `json:"THABER"`
	TDESCUENTO float64 `json:"TDESCUENTO"`
	TLIQUIDO   float64 `json:"TLIQUIDO"`
	PLAZA      string  `json:"PLAZA"`
	DESCPLAZA  string  `json:"DESC_PLAZA"`
	DPLANILLA  string  `json:"DPLANILLA"`
	FNACIMIENT string  `json:"FNACIMIENT"`
	FINGRESO   string  `json:"FINGRESO"`
	CODIE      string  `json:"COD_IE"`
	CODCARGO   string  `json:"COD_CARGO"`
	CREGPENS   string  `json:"CREG_PENS"`
	CUSSP      string  `json:"CUSSP"`
	Relevance  float64 `json:"_score,omitempty"`
}

// SearchFilters agrupa todos los criterios de busqueda
type SearchFilters struct {
	Q         string
	DNI       string
	Paterno   string
	Materno   string
	Nombres   string
	Oficina   string
	TServidor string
	Cargo     string
	Reglab    string
	Secuencial string
	Modular    string
	CodIE      string
	CodCargo   string
	Cussp      string
	Plaza      string
	Sexo       string
	Situacion  string
	Limit     int
	Offset    int
}

// Stats son las metricas globales
type Stats struct {
	Total         int64 `json:"total"`
	Ugels         int64 `json:"ugels"`
	Instituciones int64 `json:"instituciones"`
	Cargos        int64 `json:"cargos"`
}

// ---------- Construccion de consulta ----------

const selectCols = `id, PERPAGO, MODULAR, SECUENCIAL, NDOCUMENTO, PATERNO, MATERNO, NOMBRES,
	OFICINA, DTSERVIDOR, NOMBRE_IE, DES_CARGO, REGLAB, JORLABORAL, SEXO, DSITUACION,
	THABER, TDESCUENTO, TLIQUIDO, PLAZA, DESC_PLAZA, DPLANILLA, FNACIMIENT, FINGRESO,
	COD_IE, COD_CARGO, CREG_PENS, CUSSP`

// Columnas del indice FULLTEXT amplio (debe coincidir con el ALTER TABLE en
// scripts/load-data.sh). Es el respaldo cuando el termino buscado no esta al
// principio del campo.
const ftsColumns = `NDOCUMENTO, PATERNO, MATERNO, NOMBRES, OFICINA, NOMBRE_IE, DES_CARGO, REGLAB, DTSERVIDOR, PLAZA, DESC_PLAZA`

// minBranchHits: por debajo de esta cantidad de coincidencias en la via rapida
// (ramas por prefijo) se prueba tambien FULLTEXT y gana el que devuelva mas filas.
// Hace falta porque las ramas solo encuentran el termino al PRINCIPIO del campo:
// "LA MAR" (OFICINA = "UGEL LA MAR") o "DE LA CRUZ" no las encuentra.
const minBranchHits = 25

// ftsSortMaxRows: por encima de este numero de coincidencias se renuncia a
// ordenar por relevancia en el respaldo FULLTEXT (ver searchByFullText).
const ftsSortMaxRows = 20000

// buildTokens normaliza la consulta y la divide en tokens (sin signos ni acentos
// que FULLTEXT no indexaria). "HUAMAN, Perez" -> ["HUAMAN", "Perez"]
func buildTokens(q string) []string {
	var sb strings.Builder
	for _, r := range q {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else {
			sb.WriteRune(' ')
		}
	}
	return strings.Fields(sb.String())
}

// fullTextArg convierte los tokens en la expresion AGAINST de modo booleano:
// ["HUAMAN","Perez"] -> "+HUAMAN* +Perez*"
// Devuelve false si ningun token llega a 2 caracteres: innodb_ft_min_token_size=2
// impide indexarlos, asi que no tiene sentido usar FULLTEXT.
func fullTextArg(tokens []string) (string, bool) {
	hasUsable := false
	for _, t := range tokens {
		if len(t) > 1 {
			hasUsable = true
			break
		}
	}
	if !hasUsable {
		return "", false
	}
	parts := make([]string, 0, len(tokens))
	for _, t := range tokens {
		parts = append(parts, "+"+t+"*")
	}
	return strings.Join(parts, " "), true
}

// matchExpr arma "MATCH(cols) AGAINST (? IN BOOLEAN MODE)" para un indice dado
func matchExpr(cols string) string {
	return "MATCH(" + cols + ") AGAINST (? IN BOOLEAN MODE)"
}

// ---------- Armado de condiciones ----------

// buildFilterConds arma las condiciones de los filtros de campo.
// La busqueda global (q) se trata aparte porque puede usar distintos indices.
//
// Los campos de texto usan LIKE 'texto%' (prefijo) y no '%texto%': con prefijo
// MySQL puede recorrer el indice compuesto ya ordenado, mientras que '%texto%'
// obliga a leer la tabla completa (686k filas, ~1,5 s).
func buildFilterConds(f SearchFilters) ([]string, []interface{}) {
	var conds []string
	var args []interface{}

	if f.DNI != "" {
		conds = append(conds, "NDOCUMENTO LIKE ?")
		args = append(args, f.DNI+"%")
	}
	if f.Paterno != "" {
		conds = append(conds, "PATERNO LIKE ?")
		args = append(args, f.Paterno+"%")
	}
	if f.Materno != "" {
		conds = append(conds, "MATERNO LIKE ?")
		args = append(args, f.Materno+"%")
	}
	if f.Nombres != "" {
		conds = append(conds, "NOMBRES LIKE ?")
		args = append(args, f.Nombres+"%")
	}
	if f.Oficina != "" {
		conds = append(conds, "OFICINA = ?")
		args = append(args, f.Oficina)
	}
	if f.TServidor != "" {
		conds = append(conds, "DTSERVIDOR = ?")
		args = append(args, f.TServidor)
	}
	if f.Cargo != "" {
		conds = append(conds, "DES_CARGO = ?")
		args = append(args, f.Cargo)
	}
	if f.Reglab != "" {
		conds = append(conds, "REGLAB = ?")
		args = append(args, f.Reglab)
	}
	// Campos de codigo: coincidencia por prefijo (LIKE 'valor%') para poder
	// teclear solo parte del codigo (p. ej. los primeros digitos del modular).
	if f.Secuencial != "" {
		conds = append(conds, "SECUENCIAL LIKE ?")
		args = append(args, f.Secuencial+"%")
	}
	if f.Modular != "" {
		conds = append(conds, "MODULAR LIKE ?")
		args = append(args, f.Modular+"%")
	}
	if f.CodIE != "" {
		conds = append(conds, "COD_IE LIKE ?")
		args = append(args, f.CodIE+"%")
	}
	if f.CodCargo != "" {
		conds = append(conds, "COD_CARGO LIKE ?")
		args = append(args, f.CodCargo+"%")
	}
	if f.Cussp != "" {
		conds = append(conds, "CUSSP LIKE ?")
		args = append(args, f.Cussp+"%")
	}
	if f.Plaza != "" {
		conds = append(conds, "PLAZA LIKE ?")
		args = append(args, f.Plaza+"%")
	}
	// Campos categoricos: coincidencia exacta.
	if f.Sexo != "" {
		conds = append(conds, "SEXO = ?")
		args = append(args, f.Sexo)
	}
	if f.Situacion != "" {
		conds = append(conds, "DSITUACION = ?")
		args = append(args, f.Situacion)
	}
	return conds, args
}

// ---------- Ramas de busqueda por prefijo ----------

// branch es una "rama" de la busqueda: el primer token como prefijo sobre una
// columna apoyada en un indice compuesto que ya entrega las filas ordenadas.
// Recorrer el indice en orden permite a MySQL detenerse en cuanto tiene las 30
// filas de la pagina, en lugar de materializar y ordenar todas las coincidencias.
type branch struct {
	col     string
	orderBy string
	conds   []string      // condiciones propias de la rama (con placeholders)
	args    []interface{} // valores de esas condiciones, en el mismo orden
}

// branchCols define las ramas y su prioridad: primero la identidad de la
// persona, despues los datos institucionales. Cada una necesita un indice
// compuesto (columna, PATERNO, MATERNO) para poder ordenar sin filesort.
var branchCols = []struct {
	col     string
	orderBy string
	group   string // "name": los tokens restantes caen en cualquier campo de identidad
	//               "inst": los tokens restantes se buscan dentro de la misma columna
}{
	{"PATERNO", "PATERNO, MATERNO, NOMBRES", "name"},
	{"MATERNO", "MATERNO, PATERNO, NOMBRES", "name"},
	{"NOMBRES", "NOMBRES, PATERNO, MATERNO", "name"},
	{"NDOCUMENTO", "NDOCUMENTO, PATERNO, MATERNO", "name"},
	{"DES_CARGO", "DES_CARGO, PATERNO, MATERNO", "inst"},
	{"OFICINA", "OFICINA, PATERNO, MATERNO", "inst"},
	{"NOMBRE_IE", "NOMBRE_IE, PATERNO, MATERNO", "inst"},
}

// nameCols son las columnas de identidad. Los tokens adicionales de una
// busqueda de persona ("HUAMAN PEREZ", "MARIA HUAMAN") pueden caer en
// cualquiera de ellas.
var nameCols = []string{"PATERNO", "MATERNO", "NOMBRES", "NDOCUMENTO"}

// buildBranches arma las ramas. Son DISJUNTAS: cada rama excluye las columnas
// de las anteriores, de modo que
//   - el total es exacto (no se cuentan dos veces los HUACAC HUACAC), y
//   - al paginar no salen filas repetidas.
//
// El primer token fija el rango del indice (LIKE 'tok%'); los restantes se
// comprueban SOLO dentro de ese rango, asi el coste sigue acotado.
func buildBranches(tokens []string) []branch {
	first := tokens[0] + "%"
	rest := tokens[1:]

	out := make([]branch, 0, len(branchCols))
	for i, c := range branchCols {
		conds := []string{c.col + " LIKE ?"}
		args := []interface{}{first}

		// Exclusiones: lo ya cubierto por las ramas anteriores
		for j := 0; j < i; j++ {
			conds = append(conds, branchCols[j].col+" NOT LIKE ?")
			args = append(args, first)
		}

		// Tokens restantes
		targets := nameCols
		if c.group == "inst" {
			targets = []string{c.col}
		}
		for _, t := range rest {
			like := "%" + t + "%"
			parts := make([]string, 0, len(targets))
			for _, col := range targets {
				parts = append(parts, col+" LIKE ?")
				args = append(args, like)
			}
			conds = append(conds, "("+strings.Join(parts, " OR ")+")")
		}

		out = append(out, branch{col: c.col, orderBy: c.orderBy, conds: conds, args: args})
	}
	return out
}

func (b branch) condsWith(base []string) []string {
	return append(cloneConds(base), b.conds...)
}

func (b branch) argsWith(base []interface{}) []interface{} {
	return append(cloneArgs(base), b.args...)
}

func cloneConds(conds []string) []string {
	out := make([]string, len(conds), len(conds)+2)
	copy(out, conds)
	return out
}

func cloneArgs(args []interface{}) []interface{} {
	out := make([]interface{}, len(args), len(args)+4)
	copy(out, args)
	return out
}

func whereOf(conds []string) string {
	if len(conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(conds, " AND ")
}

func countRows(conds []string, args []interface{}) (int64, error) {
	var n int64
	if err := DBPool.QueryRow("SELECT COUNT(*) FROM records"+whereOf(conds), args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// ---------- Consultas ----------

// SearchRecords elige la estrategia segun la forma de la consulta.
//
// El problema de fondo: MySQL FULLTEXT no puede paginar rapido. Para "+hua*"
// devuelve 30.656 filas y hay que ordenarlas todas antes de saber cuales son
// las 30 primeras: COUNT 45 ms + SELECT 460 ms. Y como en modo booleano todas
// las filas que coinciden una sola vez reciben el MISMO score, esa ordenacion
// ni siquiera mejora la relevancia. Con 180 ms de debounce en el frontend,
// escribir "HUAMAN" letra a letra resultaba inusable (3,3 s con una sola letra).
//
// Tres estrategias, medidas sobre las 686.686 filas reales:
//
//   A) Via rapida (por defecto): ramas con LIKE 'tok%' apoyadas en indices
//      compuestos. MySQL recorre el indice ya en orden y se detiene al llegar
//      al LIMIT:
//         COUNT ~6 ms por rama | pagina ~0,6 ms   (740x mas rapido que FULLTEXT)
//      Cubre apellidos, nombre, DNI y tambien cargo / UGEL / institucion.
//      Con varios tokens, el primero fija el rango y el resto se filtra dentro.
//
//   B) Respaldo: si las ramas encuentran pocas filas (< minBranchHits) se prueba
//      FULLTEXT ft_search. Hace falta para los terminos que no estan al principio
//      del campo, p. ej. "LA MAR" (OFICINA = "UGEL LA MAR") o "DE LA CRUZ".
func SearchRecords(f SearchFilters) ([]Record, int64, error) {
	baseConds, baseArgs := buildFilterConds(f)
	tokens := buildTokens(f.Q)

	// Sin texto: solo filtros de campo
	if len(tokens) == 0 {
		return searchByFilters(f, baseConds, baseArgs)
	}

	// A) Via rapida
	recs, total, err := searchByBranches(f, baseConds, baseArgs, tokens)
	if err != nil {
		return nil, 0, err
	}
	if total >= minBranchHits {
		return recs, total, nil
	}

	// B) Respaldo FULLTEXT (solo si hay algun token de 2+ caracteres)
	if arg, ok := fullTextArg(tokens); ok {
		ftRecs, ftTotal, ferr := searchByFullText(f, baseConds, baseArgs, arg)
		if ferr != nil {
			return nil, 0, ferr
		}
		if ftTotal > total {
			return ftRecs, ftTotal, nil
		}
	}
	return recs, total, nil
}

// searchByBranches pagina concatenando ramas disjuntas ya ordenadas por indice.
func searchByBranches(f SearchFilters, baseConds []string, baseArgs []interface{}, tokens []string) ([]Record, int64, error) {
	branches := buildBranches(tokens)

	// 1) Cuantas filas aporta cada rama (cada COUNT recorre solo su rango de indice)
	counts := make([]int64, len(branches))
	var total int64
	for i, b := range branches {
		n, err := countRows(b.condsWith(baseConds), b.argsWith(baseArgs))
		if err != nil {
			return nil, 0, fmt.Errorf("count rama %s: %w", b.col, err)
		}
		counts[i] = n
		total += n
	}
	if total == 0 {
		return []Record{}, 0, nil
	}

	// 2) Repartir offset/limit entre las ramas, respetando su orden de prioridad
	records := make([]Record, 0, f.Limit)
	skip, need := int64(f.Offset), int64(f.Limit)

	for i, b := range branches {
		if need <= 0 {
			break
		}
		c := counts[i]
		if c == 0 {
			continue
		}
		if skip >= c {
			skip -= c
			continue
		}
		take := c - skip
		if take > need {
			take = need
		}

		q := "SELECT " + selectCols + " FROM records" +
			whereOf(b.condsWith(baseConds)) +
			" ORDER BY " + b.orderBy + " LIMIT ? OFFSET ?"
		args := append(b.argsWith(baseArgs), take, skip)

		rows, err := DBPool.Query(q, args...)
		if err != nil {
			return nil, 0, fmt.Errorf("query rama %s: %w", b.col, err)
		}
		page, err := scanRecords(rows, false)
		rows.Close()
		if err != nil {
			return nil, 0, err
		}

		records = append(records, page...)
		need -= int64(len(page))
		skip = 0
	}
	return records, total, nil
}

// searchByFullText es el respaldo con MATCH...AGAINST sobre el indice amplio,
// necesario cuando el termino buscado no esta al principio del campo.
//
// Ordenar por relevancia aqui si que aporta (con varios tokens el score ya no
// es constante), pero cuesta: ordenar 570.000 filas tarda 4 s frente a 0,7 s sin
// ordenar. Por eso por encima de ftsSortMaxRows se renuncia al ORDER BY y las
// filas salen en el orden natural del indice.
func searchByFullText(f SearchFilters, baseConds []string, baseArgs []interface{}, arg string) ([]Record, int64, error) {
	total, err := countRows(append(cloneConds(baseConds), matchExpr(ftsColumns)),
		append(cloneArgs(baseArgs), arg))
	if err != nil {
		return nil, 0, fmt.Errorf("count fulltext: %w", err)
	}
	if total == 0 {
		return []Record{}, 0, nil
	}

	sel, order := selectCols, ""
	withScore := false
	var pre []interface{}

	if total <= ftsSortMaxRows {
		sel = matchExpr(ftsColumns) + " AS _score, " + selectCols
		order = " ORDER BY " + matchExpr(ftsColumns) + " DESC, PATERNO, MATERNO, NOMBRES"
		withScore = true
		pre = []interface{}{arg} // placeholder del SELECT
	}

	q := "SELECT " + sel + " FROM records" +
		whereOf(append(cloneConds(baseConds), matchExpr(ftsColumns))) + order +
		" LIMIT ? OFFSET ?"

	// Placeholders: [SELECT] WHERE(filtros + MATCH) [ORDER BY] LIMIT OFFSET
	args := append(pre, cloneArgs(baseArgs)...)
	args = append(args, arg)
	if withScore {
		args = append(args, arg)
	}
	args = append(args, f.Limit, f.Offset)

	rows, err := DBPool.Query(q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query fulltext: %w", err)
	}
	records, err := scanRecords(rows, withScore)
	rows.Close()
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

// searchByFilters se usa cuando no hay texto de busqueda: solo filtros de campo.
// ORDER BY PATERNO, MATERNO, NOMBRES esta cubierto por idx_pmn, asi que la
// primera pagina sale casi gratis incluso sin WHERE.
func searchByFilters(f SearchFilters, baseConds []string, baseArgs []interface{}) ([]Record, int64, error) {
	total, err := countRows(baseConds, baseArgs)
	if err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	if total == 0 {
		return []Record{}, 0, nil
	}
	q := "SELECT " + selectCols + " FROM records" + whereOf(baseConds) +
		" ORDER BY PATERNO, MATERNO, NOMBRES LIMIT ? OFFSET ?"
	rows, err := DBPool.Query(q, append(cloneArgs(baseArgs), f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("query: %w", err)
	}
	records, err := scanRecords(rows, false)
	rows.Close()
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

// scanRecords convierte las filas en Records. withScore indica si la consulta
// trae una columna _score delante del resto.
func scanRecords(rows *sql.Rows, withScore bool) ([]Record, error) {
	records := make([]Record, 0, 32)
	for rows.Next() {
		var r Record
		var score *float64
		var err error

		if withScore {
			err = rows.Scan(&score,
				&r.ID, &r.PERPAGO, &r.MODULAR, &r.SECUENCIAL, &r.NDOCUMENTO,
				&r.PATERNO, &r.MATERNO, &r.NOMBRES, &r.OFICINA, &r.DTSERVIDOR, &r.NOMBREIE,
				&r.DESCARGO, &r.REGLAB, &r.JORLABORAL, &r.SEXO, &r.DSITUACION,
				&r.THABER, &r.TDESCUENTO, &r.TLIQUIDO, &r.PLAZA, &r.DESCPLAZA, &r.DPLANILLA,
				&r.FNACIMIENT, &r.FINGRESO, &r.CODIE, &r.CODCARGO, &r.CREGPENS, &r.CUSSP)
		} else {
			err = rows.Scan(
				&r.ID, &r.PERPAGO, &r.MODULAR, &r.SECUENCIAL, &r.NDOCUMENTO,
				&r.PATERNO, &r.MATERNO, &r.NOMBRES, &r.OFICINA, &r.DTSERVIDOR, &r.NOMBREIE,
				&r.DESCARGO, &r.REGLAB, &r.JORLABORAL, &r.SEXO, &r.DSITUACION,
				&r.THABER, &r.TDESCUENTO, &r.TLIQUIDO, &r.PLAZA, &r.DESCPLAZA, &r.DPLANILLA,
				&r.FNACIMIENT, &r.FINGRESO, &r.CODIE, &r.CODCARGO, &r.CREGPENS, &r.CUSSP)
		}
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if score != nil {
			r.Relevance = *score
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func GetStats() (*Stats, error) {
	var s Stats
	err := DBPool.QueryRow(`
		SELECT
			COUNT(*),
			COUNT(DISTINCT OFICINA),
			COUNT(DISTINCT NOMBRE_IE),
			COUNT(DISTINCT DES_CARGO)
		FROM records`).Scan(&s.Total, &s.Ugels, &s.Instituciones, &s.Cargos)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// allowedDistinct evita inyeccion SQL: solo columnas conocidas
var allowedDistinct = map[string]bool{
	"OFICINA":    true,
	"DTSERVIDOR": true,
	"DES_CARGO":  true,
	"REGLAB":     true,
	"NOMBRE_IE":  true,
	"DPLANILLA":  true,
	"SEXO":       true,
	"DSITUACION": true,
}

func GetDistinct(field string) ([]string, error) {
	if !allowedDistinct[field] {
		return nil, fmt.Errorf("campo no permitido: %s", field)
	}
	rows, err := DBPool.Query(
		"SELECT DISTINCT "+field+" FROM records WHERE "+field+" <> '' ORDER BY "+field)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vals := make([]string, 0, 256)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, rows.Err()
}