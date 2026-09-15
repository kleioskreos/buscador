import React from 'react'

export default function SearchPanel({
  query, setQuery,
  filters, setFilter,
  options,
  total, elapsed, loading,
  onReset,
}) {
  return (
    <section className="search-card">
      <h3 className="card-title">Criterios de Búsqueda</h3>

      {/* Barra global */}
      <div className="main-search">
        <svg className="search-icon" viewBox="0 0 24 24" fill="none"
             stroke="currentColor" strokeWidth="2.2"
             strokeLinecap="round" strokeLinejoin="round">
          <circle cx="11" cy="11" r="7" />
          <path d="m21 21-4.3-4.3" />
        </svg>

        <input
          id="globalQuery"
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Buscar por DNI, apellido, nombre, cargo, UGEL, colegio…"
          autoComplete="off"
          spellCheck="false"
        />

        {query && (
          <button className="clear-btn" onClick={() => setQuery('')} title="Limpiar">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none"
                 stroke="currentColor" strokeWidth="2.5" strokeLinecap="round">
              <path d="M18 6L6 18M6 6l12 12" />
            </svg>
          </button>
        )}
      </div>

      {/* Filtros especificos */}
      <div className="filter-grid">
        <div className="filter-field">
          <label htmlFor="fDni">DNI</label>
          <input id="fDni" type="text" inputMode="numeric" maxLength={12}
                 placeholder="Ej. 28293046"
                 value={filters.dni}
                 onChange={(e) => setFilter('dni', e.target.value)} />
        </div>

        <div className="filter-field">
          <label htmlFor="fPaterno">Apellido Paterno</label>
          <input id="fPaterno" type="text" placeholder="Ej. HUAMAN"
                 value={filters.paterno}
                 onChange={(e) => setFilter('paterno', e.target.value)} />
        </div>

        <div className="filter-field">
          <label htmlFor="fMaterno">Apellido Materno</label>
          <input id="fMaterno" type="text" placeholder="Ej. SANCHEZ"
                 value={filters.materno}
                 onChange={(e) => setFilter('materno', e.target.value)} />
        </div>

        <div className="filter-field">
          <label htmlFor="fNombres">Nombres</label>
          <input id="fNombres" type="text" placeholder="Ej. APOLICARPIO"
                 value={filters.nombres}
                 onChange={(e) => setFilter('nombres', e.target.value)} />
        </div>

        <div className="filter-field">
          <label htmlFor="fOficina">UGEL / Oficina</label>
          <select id="fOficina" value={filters.oficina}
                  onChange={(e) => setFilter('oficina', e.target.value)}>
            <option value="">— Todas —</option>
            {options.oficina.map((v) => <option key={v} value={v}>{v}</option>)}
          </select>
        </div>

        <div className="filter-field">
          <label htmlFor="fTservidor">Tipo de Servidor</label>
          <select id="fTservidor" value={filters.tservidor}
                  onChange={(e) => setFilter('tservidor', e.target.value)}>
            <option value="">— Todos —</option>
            {options.tservidor.map((v) => <option key={v} value={v}>{v}</option>)}
          </select>
        </div>

        <div className="filter-field">
          <label htmlFor="fCargo">Cargo</label>
          <select id="fCargo" value={filters.cargo}
                  onChange={(e) => setFilter('cargo', e.target.value)}>
            <option value="">— Todos —</option>
            {options.cargo.map((v) => <option key={v} value={v}>{v}</option>)}
          </select>
        </div>

        <div className="filter-field">
          <label htmlFor="fReglab">Ley / Régimen</label>
          <select id="fReglab" value={filters.reglab}
                  onChange={(e) => setFilter('reglab', e.target.value)}>
            <option value="">— Todos —</option>
            {options.reglab.map((v) => <option key={v} value={v}>{v}</option>)}
          </select>
        </div>

        <div className="filter-field">
          <label htmlFor="fSecuencial">Secuencial</label>
          <input id="fSecuencial" type="text" inputMode="numeric" maxLength={12}
                 placeholder="Ej. 279004"
                 value={filters.secuencial}
                 onChange={(e) => setFilter('secuencial', e.target.value)} />
        </div>

        <div className="filter-field">
          <label htmlFor="fModular">Código Modular</label>
          <input id="fModular" type="text" maxLength={20}
                 placeholder="Ej. 5000002084"
                 value={filters.modular}
                 onChange={(e) => setFilter('modular', e.target.value)} />
        </div>

        <div className="filter-field">
          <label htmlFor="fCodIe">Código IE</label>
          <input id="fCodIe" type="text" maxLength={20}
                 placeholder="Ej. DT60V411"
                 value={filters.cod_ie}
                 onChange={(e) => setFilter('cod_ie', e.target.value)} />
        </div>

        <div className="filter-field">
          <label htmlFor="fCodCargo">Código Cargo</label>
          <input id="fCodCargo" type="text" maxLength={12}
                 placeholder="Ej. 4006"
                 value={filters.cod_cargo}
                 onChange={(e) => setFilter('cod_cargo', e.target.value)} />
        </div>

        <div className="filter-field">
          <label htmlFor="fCussp">CUSSP</label>
          <input id="fCussp" type="text" maxLength={20}
                 placeholder="Ej. 616970OGXCX2"
                 value={filters.cussp}
                 onChange={(e) => setFilter('cussp', e.target.value)} />
        </div>

        <div className="filter-field">
          <label htmlFor="fPlaza">Plaza</label>
          <input id="fPlaza" type="text" maxLength={12}
                 placeholder="Ej. 0000"
                 value={filters.plaza}
                 onChange={(e) => setFilter('plaza', e.target.value)} />
        </div>

        <div className="filter-field">
          <label htmlFor="fSexo">Sexo</label>
          <select id="fSexo" value={filters.sexo}
                  onChange={(e) => setFilter('sexo', e.target.value)}>
            <option value="">— Todos —</option>
            {options.sexo.map((v) => <option key={v} value={v}>{v}</option>)}
          </select>
        </div>

        <div className="filter-field">
          <label htmlFor="fSituacion">Situación</label>
          <select id="fSituacion" value={filters.situacion}
                  onChange={(e) => setFilter('situacion', e.target.value)}>
            <option value="">— Todas —</option>
            {options.situacion.map((v) => <option key={v} value={v}>{v}</option>)}
          </select>
        </div>
      </div>

      {/* Acciones */}
      <div className="action-row">
        <div className="search-meta">
          <span>
            Mostrando <strong>{Number(total || 0).toLocaleString('es-PE')}</strong> resultados
          </span>
          <span className="timer-chip">
            {loading ? '…' : `${elapsed != null ? elapsed.toFixed(1) : '0.0'} ms`}
          </span>
          <span className="kbd-hint">
            <span className="kbd">/</span> enfocar
            <span className="kbd">Esc</span> limpiar
          </span>
        </div>

        <button className="btn btn-ghost" onClick={onReset} disabled={loading}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none"
               stroke="currentColor" strokeWidth="2.2"
               strokeLinecap="round" strokeLinejoin="round">
            <path d="M3 12a9 9 0 1 0 3-6.7L3 8" />
            <path d="M3 3v5h5" />
          </svg>
          Limpiar
        </button>
      </div>
    </section>
  )
}