package main

// import.go — Importacion de datos (reemplaza la tabla records)
//
// El endpoint POST /api/import recibe un archivo (.xlsb o .tsv) por multipart,
// lo guarda en temporal y lanza una importacion en segundo plano para no
// bloquear la respuesta HTTP (cargar 686k filas + reconstruir indices tarda
// minutos). El frontend sondea GET /api/import para conocer el progreso.
//
// Flujo de la importacion:
//   1. Si es .xlsb  -> python3 /app/xlsb_to_tsv.py lo convierte a TSV canonico.
//   2. Se quitan los 8 indices pesados (ft_search + 7 compuestos).
//   3. TRUNCATE records.
//   4. LOAD DATA LOCAL INFILE del TSV (el backend lee el archivo y lo envia).
//   5. Se recrean los 8 indices (FULLTEXT amplio + 7 compuestos por prefijo).
//
// Es una operacion destructiva: vacia y reemplaza todos los registros.

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ---------- Estado global de la importacion ----------

type ImportJob struct {
	mu           sync.Mutex
	Running      bool   `json:"running"`
	Stage        string `json:"stage"`
	RowsDone     int64  `json:"rows_done"`
	RowsTotal    int64  `json:"rows_total"`
	Filename     string `json:"filename"`
	Format       string `json:"format"`
	StartedAt    string `json:"started_at"`
	FinishedAt   string `json:"finished_at"`
	Error        string `json:"error"`
	Done         bool   `json:"done"`
	RowsImported int64  `json:"rows_imported"`
	ElapsedMs    float64 `json:"elapsed_ms"`
}

var importJob = &ImportJob{}

func (j *ImportJob) snapshot() map[string]interface{} {
	j.mu.Lock()
	defer j.mu.Unlock()
	return map[string]interface{}{
		"running":       j.Running,
		"stage":         j.Stage,
		"rows_done":     j.RowsDone,
		"rows_total":    j.RowsTotal,
		"filename":      j.Filename,
		"format":        j.Format,
		"started_at":    j.StartedAt,
		"finished_at":   j.FinishedAt,
		"error":         j.Error,
		"done":          j.Done,
		"rows_imported": j.RowsImported,
		"elapsed_ms":    j.ElapsedMs,
	}
}

// update aplica fn bajo el cerrojo del job.
func (j *ImportJob) update(fn func()) {
	j.mu.Lock()
	defer j.mu.Unlock()
	fn()
}

func failImport(msg string) {
	log.Printf("[import] ERROR: %s", msg)
	importJob.update(func() {
		importJob.Error = msg
		importJob.Stage = "error"
	})
}

// ---------- Handlers ----------

// handleImport: GET -> estado; POST -> iniciar importacion.
func handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, importJob.snapshot())
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "metodo no permitido")
		return
	}
	if importJob.Running {
		writeError(w, http.StatusConflict, "ya hay una importacion en curso; espera a que termine")
		return
	}

	const maxBytes = 512 << 20 // 512 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		writeError(w, http.StatusBadRequest, "archivo demasiado grande o invalido (max 512 MB)")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "falta el campo 'file' (archivo .tsv o .xlsb)")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".tsv" && ext != ".xlsb" {
		writeError(w, http.StatusBadRequest, "formato no soportado: usa .tsv o .xlsb")
		return
	}

	// Guardar el archivo subido en temporal
	tmp, err := os.CreateTemp("", "btr-import-*"+ext)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo crear el archivo temporal")
		return
	}
	tmpPath := tmp.Name()
	if _, err := io.Copy(tmp, file); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		writeError(w, http.StatusInternalServerError, "error guardando el archivo subido")
		return
	}
	tmp.Close()

	// Lanzar la importacion en segundo plano y responder 202 en seco.
	go runImport(tmpPath, ext, header.Filename)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":   "queued",
		"filename": header.Filename,
		"format":   ext,
	})
}

// ---------- Logica de importacion ----------

func runImport(inPath, ext, origName string) {
	start := time.Now()
	importJob.update(func() {
		importJob.Running = true
		importJob.Done = false
		importJob.Error = ""
		importJob.Stage = "iniciando"
		importJob.RowsDone = 0
		importJob.RowsTotal = 0
		importJob.RowsImported = 0
		importJob.Filename = origName
		importJob.Format = ext
		importJob.StartedAt = time.Now().Format(time.RFC3339)
		importJob.FinishedAt = ""
		importJob.ElapsedMs = 0
	})
	defer func() {
		importJob.update(func() {
			importJob.Running = false
			importJob.Done = true
			importJob.FinishedAt = time.Now().Format(time.RFC3339)
			importJob.ElapsedMs = float64(time.Since(start).Microseconds()) / 1000.0
		})
		os.Remove(inPath)
	}()

	tsvPath := inPath

	// Paso 1: convertir .xlsb -> .tsv (preserva el orden canonico de columnas)
	if ext == ".xlsb" {
		importJob.update(func() { importJob.Stage = "convirtiendo xlsb -> tsv (puede tardar)" })
		out, err := os.CreateTemp("", "btr-converted-*.tsv")
		if err != nil {
			failImport(fmt.Sprintf("no se pudo crear el TSV temporal: %v", err))
			return
		}
		tsvPath = out.Name()
		out.Close()
		defer os.Remove(tsvPath)

		cmd := exec.Command("python3", "/app/xlsb_to_tsv.py", inPath, tsvPath)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			failImport(fmt.Sprintf("fallo la conversion xlsb: %v | %s", err, stderr.String()))
			return
		}
		log.Printf("[import] conversion xlsb -> tsv: %s", strings.TrimSpace(stderr.String()))
	}

	// Paso 2: contar filas para el indicador de progreso (menos la cabecera)
	total, err := countLines(tsvPath)
	if err != nil {
		failImport(fmt.Sprintf("no se pudo leer el TSV: %v", err))
		return
	}
	if total > 0 {
		total-- // la primera linea es la cabecera
	}
	importJob.update(func() { importJob.RowsTotal = total })

	// Paso 3: preparar la tabla (quitar indices pesados + vaciar)
	importJob.update(func() { importJob.Stage = "preparando tabla (quitando indices)" })
	if err := prepareTableForLoad(); err != nil {
		failImport(fmt.Sprintf("error preparando la tabla: %v", err))
		return
	}

	// Paso 4: cargar datos
	importJob.update(func() { importJob.Stage = "cargando datos (LOAD DATA)" })
	if err := loadDataTSV(tsvPath); err != nil {
		failImport(fmt.Sprintf("error cargando datos: %v", err))
		return
	}
	importJob.update(func() { importJob.RowsDone = total })

	// Paso 5: reconstruir indices (FULLTEXT amplio + 7 compuestos por prefijo)
	importJob.update(func() { importJob.Stage = "creando indices (FULLTEXT + 7 compuestos)" })
	if err := rebuildIndexes(); err != nil {
		failImport(fmt.Sprintf("error creando indices: %v", err))
		return
	}

	// Conteo final
	var imported int64
	if err := DBPool.QueryRow("SELECT COUNT(*) FROM records").Scan(&imported); err != nil {
		failImport(fmt.Sprintf("error contando registros: %v", err))
		return
	}
	importJob.update(func() {
		importJob.RowsImported = imported
		importJob.Stage = "completado"
	})
	log.Printf("[import] importacion completada: %d filas en %s", imported, time.Since(start))
}

// countLines cuenta los '\n' de un archivo (rapido incluso para 167 MB).
func countLines(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	buf := make([]byte, 1<<20)
	var count int64
	for {
		n, rerr := f.Read(buf)
		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				count++
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			return 0, rerr
		}
	}
	return count, nil
}

// prepareTableForLoad quita los indices pesados (los que ralentizan el LOAD) y
// vacia la tabla. Los indices simples creados en el esquema inicial se mantienen.
//
// No usamos "DROP INDEX IF EXISTS" porque no esta disponible en todas las
// versiones/paquetes de MySQL; en su lugar consultamos information_schema y solo
// eliminamos los indices que de verdad existen (es idempotente entre reintentos).
func prepareTableForLoad() error {
	drops := []string{
		"ft_search", "idx_pmn", "idx_mpn", "idx_npm", "idx_dni_n",
		"idx_cargo_n", "idx_ofic_n", "idx_ie_n",
	}
	ph := strings.Repeat("?,", len(drops))
	ph = ph[:len(ph)-1]
	q := "SELECT INDEX_NAME FROM information_schema.STATISTICS " +
		"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'records' " +
		"AND INDEX_NAME IN (" + ph + ")"
	args := make([]interface{}, 0, len(drops))
	for _, d := range drops {
		args = append(args, d)
	}
	rows, err := DBPool.Query(q, args...)
	if err != nil {
		return fmt.Errorf("consultando indices: %w", err)
	}
	existing := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("scan indices: %w", err)
		}
		existing[name] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows indices: %w", err)
	}

	for _, idx := range drops {
		if !existing[idx] {
			continue
		}
		if _, err := DBPool.Exec("DROP INDEX " + idx + " ON records"); err != nil {
			return fmt.Errorf("drop %s: %w", idx, err)
		}
	}
	if _, err := DBPool.Exec("TRUNCATE TABLE records"); err != nil {
		return fmt.Errorf("truncate: %w", err)
	}
	return nil
}

// loadCols es el orden canonico de las 27 columnas (debe coincidir con
// scripts/load-data.sh y con la cabecera del TSV que produce xlsb_to_tsv.py).
const loadCols = `PERPAGO, MODULAR, SECUENCIAL, NDOCUMENTO, PATERNO, MATERNO, NOMBRES,
	OFICINA, DTSERVIDOR, NOMBRE_IE, DES_CARGO, REGLAB, JORLABORAL, SEXO,
	DSITUACION, THABER, TDESCUENTO, TLIQUIDO, PLAZA, DESC_PLAZA, DPLANILLA,
	FNACIMIENT, FINGRESO, COD_IE, COD_CARGO, CREG_PENS, CUSSP`

// escapeSQLString escapa una cadena para usarla como literal SQL ('...').
// MySQL requiere escapar la barra invertida y la comilla simple.
func escapeSQLString(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return r.Replace(s)
}

// loadDataTSV usa LOAD DATA LOCAL INFILE: el backend lee el TSV y lo transmite
// al servidor MySQL (por eso el DSN lleva allowAllFiles=true).
//
// NOTA: MySQL NO acepta un marcador '?' para el nombre del archivo en LOAD DATA,
// por eso interpolamos la ruta directamente (es un archivo temporal que nosotros
// mismos creamos, asi que no hay riesgo de inyeccion).
func loadDataTSV(path string) error {
	prep := []string{
		"SET SESSION unique_checks=0",
		"SET SESSION foreign_key_checks=0",
		"SET SESSION autocommit=0",
	}
	for _, s := range prep {
		if _, err := DBPool.Exec(s); err != nil {
			return fmt.Errorf("%s: %w", s, err)
		}
	}

	q := `LOAD DATA LOCAL INFILE '` + escapeSQLString(path) + `' INTO TABLE records
		CHARACTER SET utf8mb4
		FIELDS TERMINATED BY '\t' ENCLOSED BY '' ESCAPED BY ''
		LINES TERMINATED BY '\n'
		IGNORE 1 LINES
		(` + loadCols + `)`
	if _, err := DBPool.Exec(q); err != nil {
		return fmt.Errorf("load: %w", err)
	}
	if _, err := DBPool.Exec("COMMIT"); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// rebuildIndexes recrea el indice FULLTEXT amplio y los 7 indices compuestos
// por prefijo que usa la "via rapida" de busqueda (ver db.go).
func rebuildIndexes() error {
	stmts := []string{
		`ALTER TABLE records ADD FULLTEXT INDEX ft_search
		 (NDOCUMENTO, PATERNO, MATERNO, NOMBRES, OFICINA, NOMBRE_IE,
		  DES_CARGO, REGLAB, DTSERVIDOR, PLAZA, DESC_PLAZA)`,
		`ALTER TABLE records
		 ADD INDEX idx_pmn   (PATERNO, MATERNO, NOMBRES),
		 ADD INDEX idx_mpn   (MATERNO, PATERNO, NOMBRES),
		 ADD INDEX idx_npm   (NOMBRES, PATERNO, MATERNO),
		 ADD INDEX idx_dni_n (NDOCUMENTO, PATERNO, MATERNO),
		 ADD INDEX idx_cargo_n (DES_CARGO, PATERNO, MATERNO),
		 ADD INDEX idx_ofic_n  (OFICINA, PATERNO, MATERNO),
		 ADD INDEX idx_ie_n    (NOMBRE_IE, PATERNO, MATERNO)`,
	}
	for _, s := range stmts {
		if _, err := DBPool.Exec(s); err != nil {
			return fmt.Errorf("index: %w", err)
		}
	}
	return nil
}
