import React, { useState, useEffect, useRef } from 'react'
import api from '../api.js'

const STAGE_LABELS = {
  'iniciando': 'Iniciando importación…',
  'convirtiendo xlsb -> tsv (puede tardar)': 'Convirtiendo .xlsb a datos (puede tardar)…',
  'preparando tabla (quitando indices)': 'Preparando la tabla (quitando índices)…',
  'cargando datos (LOAD DATA)': 'Cargando datos en MySQL…',
  'creando indices (FULLTEXT + 7 compuestos)': 'Creando índices de búsqueda…',
  'completado': '¡Importación completada!',
  'error': 'Error',
}

const fmtInt = (n) => Number(n || 0).toLocaleString('es-PE')
const fmtMs = (ms) => {
  const s = (ms || 0) / 1000
  if (s < 60) return `${s.toFixed(1)} s`
  const m = Math.floor(s / 60)
  return `${m} min ${Math.round(s % 60)} s`
}

export default function ImportModal({ open, onClose, onImported }) {
  const [file, setFile] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [uploadPct, setUploadPct] = useState(0)
  const [job, setJob] = useState(null)
  const [errorMsg, setErrorMsg] = useState('')
  const fileRef = useRef(null)
  const timerRef = useRef(null)

  // Reiniciar estado al abrir/cerrar
  useEffect(() => {
    if (!open) {
      setFile(null)
      setUploading(false)
      setUploadPct(0)
      setJob(null)
      setErrorMsg('')
      if (timerRef.current) clearInterval(timerRef.current)
    }
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  const startPolling = () => {
    if (timerRef.current) clearInterval(timerRef.current)
    timerRef.current = setInterval(async () => {
      try {
        const s = await api.importStatus()
        setJob(s)
        if (s.done) {
          clearInterval(timerRef.current)
          if (!s.error && onImported) onImported()
        }
      } catch {
        /* el backend aún puede estar arrancando la importación; reintentamos */
      }
    }, 1000)
  }

  const handleFile = (e) => {
    const f = e.target.files && e.target.files[0]
    setFile(f || null)
    setErrorMsg('')
  }

  const doImport = () => {
    if (!file) return
    const fd = new FormData()
    fd.append('file', file)
    setUploading(true)
    setUploadPct(0)
    setJob(null)
    setErrorMsg('')

    const xhr = new XMLHttpRequest()
    xhr.open('POST', (import.meta.env.VITE_API_URL || '') + '/api/import')
    xhr.upload.onprogress = (ev) => {
      if (ev.lengthComputable) setUploadPct(Math.round((ev.loaded / ev.total) * 100))
    }
    xhr.onload = () => {
      setUploading(false)
      if (xhr.status >= 200 && xhr.status < 300) {
        startPolling()
      } else {
        let msg = 'Error al subir el archivo'
        try { msg = JSON.parse(xhr.responseText).error || msg } catch {}
        setErrorMsg(msg)
      }
    }
    xhr.onerror = () => {
      setUploading(false)
      setErrorMsg('Error de red al subir el archivo')
    }
    xhr.send(fd)
  }

  if (!open) return null

  const busy = uploading || (job && job.running)
  const stageLabel = job ? (STAGE_LABELS[job.stage] || job.stage) : ''
  const showProgress = job && job.running && job.stage === 'cargando datos (LOAD DATA)' && job.rows_total > 0

  return (
    <div className="modal-overlay" onClick={!busy ? onClose : undefined}>
      <div className="modal import-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Importar datos</h3>
          {!busy && (
            <button className="modal-close" onClick={onClose} aria-label="Cerrar">×</button>
          )}
        </div>

        <p className="import-desc">
          Reemplaza la planilla completa (686.686 registros) cargando un archivo
          nuevo. Acepta <strong>.xlsb</strong> (el formato original de MINEDU) o
          un <strong>.tsv</strong> con las 27 columnas en el orden canónico.
        </p>

        {errorMsg && <div className="import-error">⚠ {errorMsg}</div>}

        {/* Fase 1: selección de archivo */}
        {!job && !uploading && (
          <div className="import-drop">
            <input
              ref={fileRef}
              type="file"
              accept=".xlsb,.tsv"
              onChange={handleFile}
              id="import-file"
              className="import-file-input"
            />
            <label htmlFor="import-file" className="import-file-label">
              {file ? file.name : 'Selecciona un archivo (.xlsb o .tsv)'}
            </label>
            {file && (
              <div className="import-file-meta">
                {(file.size / (1024 * 1024)).toFixed(1)} MB · {file.name.toLowerCase().endsWith('.xlsb') ? 'Excel binario' : 'TSV'}
              </div>
            )}
            <button
              className="btn btn-primary"
              disabled={!file}
              onClick={doImport}
            >
              Importar datos
            </button>
          </div>
        )}

        {/* Fase 2: subida */}
        {uploading && (
          <div className="import-progress">
            <div className="import-stage">Subiendo archivo…</div>
            <div className="progress-bar">
              <div className="progress-fill" style={{ width: `${uploadPct}%` }} />
            </div>
            <div className="import-pct">{uploadPct}%</div>
          </div>
        )}

        {/* Fase 3: procesamiento en el servidor */}
        {job && job.running && (
          <div className="import-progress">
            <div className="import-stage">{stageLabel}</div>
            {showProgress ? (
              <>
                <div className="progress-bar">
                  <div
                    className="progress-fill"
                    style={{ width: `${Math.min(100, Math.round((job.rows_done / job.rows_total) * 100))}%` }}
                  />
                </div>
                <div className="import-pct">
                  {fmtInt(job.rows_done)} / {fmtInt(job.rows_total)} filas
                </div>
              </>
            ) : (
              <div className="loader" style={{ margin: '18px auto' }} />
            )}
          </div>
        )}

        {/* Fase 4: resultado */}
        {job && job.done && (
          <>
            {job.error ? (
              <div className="import-error">⚠ {job.error}</div>
            ) : (
              <div className="import-success">
                <div className="import-success-ico">✓</div>
                <div>
                  <strong>{fmtInt(job.rows_imported)} registros importados</strong>
                  <div className="import-success-sub">
                    en {fmtMs(job.elapsed_ms)} · {job.filename}
                  </div>
                </div>
              </div>
            )}
            <div className="modal-actions">
              <button className="btn btn-primary" onClick={onClose}>
                {job.error ? 'Cerrar' : 'Actualizar y cerrar'}
              </button>
            </div>
          </>
        )}

        <div className="import-foot">
          Operación destructiva: vacía y reemplaza todos los registros actuales.
        </div>
      </div>
    </div>
  )
}
