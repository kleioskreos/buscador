import React, { useEffect } from 'react'

// Formatea montos en soles (es-PE).
const fmtMoney = (v) => {
  if (v == null || v === '') return '—'
  const n = Number(v)
  if (Number.isNaN(n)) return String(v)
  return 'S/ ' + n.toLocaleString('es-PE', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}

// Convierte un nmero serial de Excel (das desde 1899-12-30) a fecha legible.
const serialToDate = (v) => {
  const n = Number(v)
  if (Number.isNaN(n) || n < 20000 || n > 70000) return null
  const d = new Date((n - 25569) * 86400000)
  if (Number.isNaN(d.getTime())) return null
  return d.toLocaleDateString('es-PE', {
    year: 'numeric', month: 'long', day: 'numeric',
  })
}

// Metadatos de las 27 columnas, agrupadas para el detalle.
const GROUPS = [
  {
    title: 'Identificación',
    fields: [
      { k: 'PERPAGO', label: 'Periodo de pago' },
      { k: 'MODULAR', label: 'Código modular' },
      { k: 'SECUENCIAL', label: 'Secuencial' },
      { k: 'NDOCUMENTO', label: 'DNI / Documento' },
      { k: 'COD_IE', label: 'Código IE' },
      { k: 'COD_CARGO', label: 'Código de cargo' },
      { k: 'CREG_PENS', label: 'Régimen pensión' },
      { k: 'CUSSP', label: 'CUSSP' },
    ],
  },
  {
    title: 'Datos personales',
    fields: [
      { k: 'PATERNO', label: 'Apellido paterno' },
      { k: 'MATERNO', label: 'Apellido materno' },
      { k: 'NOMBRES', label: 'Nombres' },
      { k: 'SEXO', label: 'Sexo' },
      { k: 'DSITUACION', label: 'Situación' },
      { k: 'FNACIMIENT', label: 'Fecha de nacimiento', date: true },
    ],
  },
  {
    title: 'Datos laborales',
    fields: [
      { k: 'OFICINA', label: 'UGEL / Oficina' },
      { k: 'NOMBRE_IE', label: 'Institución Educativa' },
      { k: 'DES_CARGO', label: 'Cargo' },
      { k: 'REGLAB', label: 'Régimen laboral' },
      { k: 'DTSERVIDOR', label: 'Tipo de servidor' },
      { k: 'JORLABORAL', label: 'Jornada laboral' },
      { k: 'PLAZA', label: 'Plaza' },
      { k: 'DESC_PLAZA', label: 'Desc. plaza' },
      { k: 'DPLANILLA', label: 'Planilla' },
      { k: 'FINGRESO', label: 'Fecha de ingreso', date: true },
    ],
  },
  {
    title: 'Remuneraciones',
    fields: [
      { k: 'THABER', label: 'Haber', money: true },
      { k: 'TDESCUENTO', label: 'Descuento', money: true },
      { k: 'TLIQUIDO', label: 'Líquido', money: true },
    ],
  },
]

function Field({ f, r }) {
  let value = r[f.k]
  let display = (value === '' || value == null) ? '—' : String(value)
  if (f.money) display = fmtMoney(value)
  else if (f.date) {
    const d = serialToDate(value)
    if (d) display = d
  }
  return (
    <div className="detail-field">
      <span className="detail-k">{f.label}</span>
      <span className="detail-v">{display}</span>
    </div>
  )
}

export default function PersonDetailModal({ record, onClose }) {
  useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose])

  if (!record) return null

  const fullName = `${record.PATERNO} ${record.MATERNO}, ${record.NOMBRES}`.trim()

  return (
    <div
      className="modal-overlay"
      onMouseDown={(e) => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="modal detail-modal" role="dialog" aria-modal="true">
        <div className="modal-head">
          <h3>Detalle del servidor</h3>
          <button className="modal-close" onClick={onClose} aria-label="Cerrar">×</button>
        </div>

        <div className="detail-id">
          <div className="detail-name">{fullName}</div>
          <div className="detail-dni">DNI {record.NDOCUMENTO || '—'}</div>
        </div>

        <div className="detail-body">
          {GROUPS.map((g) => (
            <section className="detail-group" key={g.title}>
              <h4 className="detail-group-title">{g.title}</h4>
              <div className="detail-grid">
                {g.fields.map((f) => <Field key={f.k} f={f} r={record} />)}
              </div>
            </section>
          ))}
        </div>

        <div className="modal-actions">
          <button className="btn-primary" onClick={onClose}>Cerrar</button>
        </div>
      </div>
    </div>
  )
}
