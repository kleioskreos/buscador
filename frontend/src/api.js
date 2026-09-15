// api.js — Cliente HTTP hacia el backend Go

// En Docker, nginx hace proxy de /api al backend.
// En desarrollo local con Vite, el proxy de vite.config.js redirige igual.
const BASE = import.meta.env.VITE_API_URL || ''

async function getJSON(path, params = {}, signal) {
  const qs = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => {
    if (v !== '' && v !== null && v !== undefined) qs.append(k, v)
  })
  const url = `${BASE}${path}?${qs.toString()}`
  const res = await fetch(url, { signal, headers: { Accept: 'application/json' } })
  if (!res.ok) {
    throw new Error(`HTTP ${res.status} — ${res.statusText}`)
  }
  return res.json()
}

export const api = {
  search: (filters, limit, offset, signal) =>
    getJSON('/api/search', { ...filters, limit, offset }, signal),

  stats: (signal) => getJSON('/api/stats', {}, signal),

  distinct: (field, signal) => getJSON('/api/distinct', { field }, signal),

  health: (signal) => getJSON('/api/health', {}, signal),

  // Importacion: POST multipart (el progreso de subida va por XHR en el modal).
  importUpload: (formData) =>
    new Promise((resolve, reject) => {
      const url = `${BASE}/api/import`
      const xhr = new XMLHttpRequest()
      xhr.open('POST', url)
      xhr.onload = () => {
        let data = {}
        try { data = JSON.parse(xhr.responseText) } catch {}
        if (xhr.status >= 200 && xhr.status < 300) resolve(data)
        else reject(new Error(data.error || `HTTP ${xhr.status}`))
      }
      xhr.onerror = () => reject(new Error('Error de red al subir el archivo'))
      xhr.send(formData)
    }),

  importStatus: (signal) => getJSON('/api/import', {}, signal),
}

export default api