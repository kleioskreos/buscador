import React from 'react'

const fmtMoney = (v) => {
  if (v == null || v === '') return '—'
  const n = Number(v)
  if (Number.isNaN(n)) return String(v)
  return 'S/ ' + n.toLocaleString('es-PE', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}

const tipoBadge = (t) => {
  if (t === 'Docente Nombrado') return <span className="badge badge-blue">{t}</span>
  if (t === 'Docente Contratado') return <span className="badge badge-cyan">{t}</span>
  if (t) return <span className="badge badge-sky">{t}</span>
  return <span className="badge badge-slate">—</span>
}

function PersonCard({ r, onSelect }) {
  const fullName = `${r.PATERNO} ${r.MATERNO}, ${r.NOMBRES}`.trim()
  const open = () => onSelect && onSelect(r)
  return (
    <div
      className="person-card clickable"
      role="button"
      tabIndex={0}
      onClick={open}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); open() }
      }}
      title="Haz clic para ver el detalle completo"
    >
      <div className="pc-head">
        <h4 className="pc-name">{fullName}</h4>
        <span className="pc-dni">DNI {r.NDOCUMENTO || '—'}</span>
      </div>

      <div className="pc-row">
        <span className="k">Tipo</span>
        <span className="v">{tipoBadge(r.DTSERVIDOR)}</span>
      </div>
      <div className="pc-row">
        <span className="k">UGEL</span>
        <span className="v">{r.OFICINA || '—'}</span>
      </div>
      <div className="pc-row">
        <span className="k">Cargo</span>
        <span className="v">{r.DES_CARGO || '—'}</span>
      </div>
      <div className="pc-row">
        <span className="k">Institución</span>
        <span className="v">{r.NOMBRE_IE || '—'}</span>
      </div>
      <div className="pc-row">
        <span className="k">Régimen</span>
        <span className="v">{r.REGLAB || '—'}</span>
      </div>
      <div className="pc-row">
        <span className="k">Jornada</span>
        <span className="v">{r.JORLABORAL ? `${r.JORLABORAL} hrs` : '—'}</span>
      </div>
      <div className="pc-row">
        <span className="k">Plaza</span>
        <span className="v">
          {r.PLAZA || '—'}{r.DESC_PLAZA ? ` · ${r.DESC_PLAZA}` : ''}
        </span>
      </div>

      <div className="pc-money">
        <span className="lbl">Líquido</span>
        <span className="amt">{fmtMoney(r.TLIQUIDO)}</span>
      </div>

      <div className="pc-hint">Ver detalle completo →</div>
    </div>
  )
}

function ResultsTable({ rows, page, pageSize, onSelect }) {
  return (
    <div className="table-wrap">
      <div className="table-scroll">
        <table className="results-table">
          <thead>
            <tr>
              <th>#</th>
              <th>Persona</th>
              <th>DNI</th>
              <th>Tipo</th>
              <th>UGEL</th>
              <th>Cargo</th>
              <th>Régimen</th>
              <th>Ingresos</th>
              <th>Desc.</th>
              <th>Líquido</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr
                key={r.id ?? i}
                className="clickable-row"
                onClick={() => onSelect && onSelect(r)}
                title="Haz clic para ver el detalle completo"
              >
                <td className="num">{(page - 1) * pageSize + i + 1}</td>
                <td>
                  <div className="person">
                    <span className="name">
                      {`${r.PATERNO} ${r.MATERNO}, ${r.NOMBRES}`.trim()}
                    </span>
                    <span className="sub" title={r.NOMBRE_IE}>
                      {r.NOMBRE_IE || '—'}
                    </span>
                  </div>
                </td>
                <td className="num">{r.NDOCUMENTO || '—'}</td>
                <td>{r.DTSERVIDOR || '—'}</td>
                <td>{r.OFICINA || '—'}</td>
                <td>{r.DES_CARGO || '—'}</td>
                <td>{r.REGLAB || '—'}</td>
                <td className="money">{fmtMoney(r.THABER)}</td>
                <td className="money">{fmtMoney(r.TDESCUENTO)}</td>
                <td className="money">{fmtMoney(r.TLIQUIDO)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

export default function ResultsGrid({
  rows, view, loading, error, page, pageSize, onSelect,
}) {
  if (loading && rows.length === 0) {
    return (
      <div className="empty-state">
        <div className="loader" />
        <h3>Buscando…</h3>
        <p>Consultando la planilla</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="empty-state">
        <h3>⚠ Error de conexión</h3>
        <p>{error}</p>
      </div>
    )
  }

  if (!loading && rows.length === 0) {
    return (
      <div className="empty-state">
        <div className="ico">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none"
               stroke="currentColor" strokeWidth="1.8"
               strokeLinecap="round" strokeLinejoin="round">
            <circle cx="11" cy="11" r="7" />
            <path d="m21 21-4.3-4.3" />
          </svg>
        </div>
        <h3>Sin resultados</h3>
        <p>Intenta con otros criterios o limpia los filtros.</p>
      </div>
    )
  }

  return view === 'cards'
    ? <div className="cards-grid">{rows.map((r, i) => <PersonCard key={r.id ?? i} r={r} onSelect={onSelect} />)}</div>
    : <ResultsTable rows={rows} page={page} pageSize={pageSize} onSelect={onSelect} />
}