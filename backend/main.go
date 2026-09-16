package main

// main.go — Servidor HTTP del buscador BTR (Go + MySQL)

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// ---------- Utilidades HTTP ----------

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[http] error encodeando respuesta: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func qInt(r *http.Request, key string, def, max int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func qStr(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// ---------- Handlers ----------

func handleSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	f := SearchFilters{
		Q:         qStr(r, "q"),
		DNI:       qStr(r, "dni"),
		Paterno:   qStr(r, "paterno"),
		Materno:   qStr(r, "materno"),
		Nombres:   qStr(r, "nombres"),
		Oficina:   qStr(r, "oficina"),
		TServidor: qStr(r, "tservidor"),
		Cargo:     qStr(r, "cargo"),
		Reglab:    qStr(r, "reglab"),
		Secuencial: qStr(r, "secuencial"),
		Modular:    qStr(r, "modular"),
		CodIE:      qStr(r, "cod_ie"),
		CodCargo:   qStr(r, "cod_cargo"),
		Cussp:      qStr(r, "cussp"),
		Plaza:      qStr(r, "plaza"),
		Sexo:       qStr(r, "sexo"),
		Situacion:  qStr(r, "situacion"),
		Limit:     qInt(r, "limit", 30, 200),
		Offset:    qInt(r, "offset", 0, 1_000_000),
	}

	records, total, err := SearchRecords(f)
	if err != nil {
		log.Printf("[search] %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":      total,
		"limit":      f.Limit,
		"offset":     f.Offset,
		"elapsed_ms": float64(time.Since(start).Microseconds()) / 1000.0,
		"results":    records,
	})
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	s, err := GetStats()
	if err != nil {
		log.Printf("[stats] %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func handleDistinct(w http.ResponseWriter, r *http.Request) {
	field := qStr(r, "field")
	if field == "" {
		writeError(w, http.StatusBadRequest, "parametro 'field' requerido")
		return
	}
	vals, err := GetDistinct(field)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"values": vals})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := DBPool.Ping(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "db_down", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------- Middleware ----------

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		if os.Getenv("DEBUG") == "1" {
			log.Printf("%s %s %v", r.Method, r.URL.String(), time.Since(start))
		}
	}
}

func chain(h http.HandlerFunc) http.HandlerFunc {
	return loggingMiddleware(corsMiddleware(h))
}

// ---------- Main ----------

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=========================================")
	log.Println("  BTR 202604 · Backend Go + MySQL")
	log.Println("=========================================")

	cfg := LoadConfig()

	if err := InitDB(cfg); err != nil {
		log.Fatalf("[fatal] %v", err)
	}
	defer DBPool.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/search", chain(handleSearch))
	mux.HandleFunc("/api/stats", chain(handleStats))
	mux.HandleFunc("/api/distinct", chain(handleDistinct))
	mux.HandleFunc("/api/health", chain(handleHealth))
	mux.HandleFunc("/health", chain(handleHealth))
	mux.HandleFunc("/api/import", chain(handleImport))

	addr := ":" + cfg.PortHTTP
	log.Printf("[server] escuchando en %s", addr)
	log.Printf("[server] endpoints: /api/search /api/stats /api/distinct /api/health /api/import")

	// NOTA: WriteTimeout arranca a contar desde el primer byte de la request,
	// no desde que el handler responde. Por eso el endpoint /api/import fallaba
	// con 502: subir 160 MB + io.Copy a disco + writeJSON ~= 60-70s, justo en
	// el limite del timeout viejo (60s). Subimos a 30 min para que la subida
	// de archivos grandes (xlsb/tsv hasta 512 MB) tenga margen. Las demas
	// rutas (search/stats/distinct/health) responden en ms y no se afectan.
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  1800 * time.Second, // 30 min (match con WriteTimeout)
		WriteTimeout: 1800 * time.Second, // 30 min
		IdleTimeout:  120 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("[fatal] servidor: %v", err)
	}
}