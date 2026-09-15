import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react'
import api from './api.js'
import Header from './components/Header.jsx'
import SearchPanel from './components/SearchPanel.jsx'
import ResultsGrid from './components/ResultsGrid.jsx'
import Pagination from './components/Pagination.jsx'
import ImportModal from './components/ImportModal.jsx'
import PersonDetailModal from './components/PersonDetailModal.jsx'

const PAGE_SIZE = 30

const EMPTY_FILTERS = {
  dni: '', paterno: '', materno: '', nombres: '',
  oficina: '', tservidor: '', cargo: '', reglab: '',
  secuencial: '', modular: '', cod_ie: '', cod_cargo: '',
  cussp: '', plaza: '', sexo: '', situacion: '',
}

export default function App() {
  const [query, setQuery] = useState('')
  const [filters, setFilters] = useState(EMPTY_FILTERS)
  const [options, setOptions] = useState({ oficina: [], tservidor: [], cargo: [], reglab: [], sexo: [], situacion: [] })
  const [stats, setStats] = useState(null)
  const [statsLoading, setStatsLoading] = useState(true)
  const [importOpen, setImportOpen] = useState(false)
  const [selected, setSelected] = useState(null)

  const [rows, setRows] = useState([])
  const [total, setTotal] = useState(0)
  const [elapsed, setElapsed] = useState(null)
  const [page, setPage] = useState(1)
  const [view, setView] = useState('cards')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const abortRef = useRef(null)
  const inputRef = useRef(null)

  // ---------- Carga inicial: stats + opciones de filtros ----------
  const loadMeta = useCallback(async (signal) => {
    try {
      const [s, ofi, tse, car, reg, sex, sit] = await Promise.all([
        api.stats(signal),
        api.distinct('OFICINA', signal),
        api.distinct('DTSERVIDOR', signal),
        api.distinct('DES_CARGO', signal),
        api.distinct('REGLAB', signal),
        api.distinct('SEXO', signal),
        api.distinct('DSITUACION', signal),
      ])
      setStats(s)
      setOptions({
        oficina: ofi.values || [],
        tservidor: tse.values || [],
        cargo: car.values || [],
        reglab: reg.values || [],
        sexo: sex.values || [],
        situacion: sit.values || [],
      })
      setError(null)
    } catch (e) {
      if (e.name !== 'AbortError') {
        setError('No se pudo conectar al backend. Verifica que los contenedores estén activos.')
      }
    } finally {
      setStatsLoading(false)
    }
  }, [])

  useEffect(() => {
    const ctrl = new AbortController()
    loadMeta(ctrl.signal)
    return () => ctrl.abort()
  }, [loadMeta])

  // ---------- Busqueda (debounce 180ms) ----------
  const runSearch = useCallback(async (pageOverride) => {
    const currentPage = pageOverride ?? page

    // Cancelar peticion anterior
    if (abortRef.current) abortRef.current.abort()
    const ctrl = new AbortController()
    abortRef.current = ctrl

    const params = { q: query.trim(), ...filters }
    Object.keys(params).forEach((k) => { if (!params[k]) delete params[k] })

    setLoading(true)
    try {
      const data = await api.search(params, PAGE_SIZE, (currentPage - 1) * PAGE_SIZE, ctrl.signal)
      setRows(data.results || [])
      setTotal(data.total || 0)
      setElapsed(data.elapsed_ms ?? null)
      setError(null)
    } catch (e) {
      if (e.name !== 'AbortError') {
        setError(`Error al buscar: ${e.message}`)
        setRows([])
        setTotal(0)
      }
    } finally {
      if (!ctrl.signal.aborted) setLoading(false)
    }
  }, [query, filters, page])

  // Debounce + reset a pagina 1 cuando cambian los criterios
  useEffect(() => {
    const t = setTimeout(() => { runSearch(1) }, 180)
    return () => clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query, filters])

  // Cuando cambia la pagina (por accion del usuario), recargar
  const goToPage = useCallback((p) => {
    setPage(p)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }, [])

  useEffect(() => {
    if (page === 1) return
    runSearch(page)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page])

  // ---------- Handlers ----------
  const setFilter = useCallback((key, value) => {
    setFilters((f) => ({ ...f, [key]: value }))
    setPage(1)
  }, [])

  const handleQueryChange = (v) => {
    setQuery(v)
    setPage(1)
  }

  const handleReset = useCallback(() => {
    setQuery('')
    setFilters(EMPTY_FILTERS)
    setPage(1)
  }, [])

  // Atajos de teclado
  useEffect(() => {
    const onKey = (e) => {
      const tag = document.activeElement?.tagName
      const isField = tag === 'INPUT' || tag === 'SELECT' || tag === 'TEXTAREA'

      if (e.key === '/' && !isField) {
        e.preventDefault()
        inputRef.current?.focus()
      }
      if (e.key === 'Escape' && document.activeElement?.id === 'globalQuery') {
        setQuery('')
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])

  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / PAGE_SIZE)), [total])

  const activeFilters = useMemo(
    () => Object.entries(filters).filter(([, v]) => v).length,
    [filters]
  )

  return (
    <>
      <Header stats={stats} loading={statsLoading} onImport={() => setImportOpen(true)} />

      <ImportModal
        open={importOpen}
        onClose={() => setImportOpen(false)}
        onImported={() => { const c = new AbortController(); loadMeta(c.signal) }}
      />

      <PersonDetailModal record={selected} onClose={() => setSelected(null)} />

      <main className="container">
        <SearchPanel
          query={query}
          setQuery={handleQueryChange}
          filters={filters}
          setFilter={setFilter}
          options={options}
          total={total}
          elapsed={elapsed}
          loading={loading}
          onReset={handleReset}
        />

        <div className="results-header">
          <h2>
            Resultados
            {activeFilters > 0 && (
              <span className="badge badge-sky" style={{ marginLeft: 10 }}>
                {activeFilters} filtro{activeFilters > 1 ? 's' : ''}
              </span>
            )}
          </h2>

          <div className="view-toggle" role="tablist">
            <button
              className={view === 'cards' ? 'active' : ''}
              onClick={() => setView('cards')}
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none"
                   stroke="currentColor" strokeWidth="2">
                <rect x="3" y="3" width="7" height="7" rx="1" />
                <rect x="14" y="3" width="7" height="7" rx="1" />
                <rect x="3" y="14" width="7" height="7" rx="1" />
                <rect x="14" y="14" width="7" height="7" rx="1" />
              </svg>
              Tarjetas
            </button>
            <button
              className={view === 'table' ? 'active' : ''}
              onClick={() => setView('table')}
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none"
                   stroke="currentColor" strokeWidth="2">
                <path d="M3 6h18M3 12h18M3 18h18" />
              </svg>
              Tabla
            </button>
          </div>
        </div>

        <ResultsGrid
          rows={rows}
          view={view}
          loading={loading}
          error={error}
          page={page}
          pageSize={PAGE_SIZE}
          onSelect={setSelected}
        />

        <Pagination page={page} totalPages={totalPages} onChange={goToPage} />
      </main>

      <footer className="app-footer">
        <span>Datos: Planilla Única de Pagos · MINEDU · BTR 202604</span>
        <span className="stack-tag">React</span>
        <span className="stack-tag">Go</span>
        <span className="stack-tag">MySQL</span>
        <span className="stack-tag">Docker</span>
      </footer>
    </>
  )
}