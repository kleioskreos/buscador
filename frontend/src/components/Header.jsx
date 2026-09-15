import React from 'react'

const fmtInt = (n) =>
  Number(n || 0).toLocaleString('es-PE')

export default function Header({ stats, loading, onImport }) {
  return (
    <header className="app-header">
      <div className="header-inner">
        <div className="brand">
          <div className="logo">B</div>
          <div className="brand-text">
            <h1>BTR 202604 · Buscador Especializado</h1>
            <p>Boleta de Trabajo y Remuneraciones — MINEDU / UGEL</p>
          </div>
        </div>

        <div className="header-right">
          <button className="btn-import" onClick={onImport} title="Importar / reemplazar datos">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none"
                 stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="17 8 12 3 7 8" />
              <line x1="12" y1="3" x2="12" y2="15" />
            </svg>
            Importar datos
          </button>

          <div className="header-stats">
            <div className="stat-pill">
              <div className="num">{loading ? '—' : fmtInt(stats?.total)}</div>
              <div className="lbl">Servidores</div>
            </div>
            <div className="stat-pill">
              <div className="num">{loading ? '—' : fmtInt(stats?.ugels)}</div>
              <div className="lbl">UGELs</div>
            </div>
            <div className="stat-pill">
              <div className="num">{loading ? '—' : fmtInt(stats?.instituciones)}</div>
              <div className="lbl">Colegios</div>
            </div>
          </div>
        </div>
      </div>
    </header>
  )
}