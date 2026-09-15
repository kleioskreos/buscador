import React from 'react'

// Construye la lista de numeros de pagina con elipsis
function pageList(current, total) {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }
  const pages = new Set([1, total, current, current - 1, current + 1])
  if (current <= 3) { pages.add(2); pages.add(3); pages.add(4) }
  if (current >= total - 2) { pages.add(total - 1); pages.add(total - 2); pages.add(total - 3) }

  const sorted = [...pages].filter((p) => p >= 1 && p <= total).sort((a, b) => a - b)

  const out = []
  let prev = 0
  for (const p of sorted) {
    if (prev && p - prev > 1) out.push('…')
    out.push(p)
    prev = p
  }
  return out
}

export default function Pagination({ page, totalPages, onChange }) {
  if (totalPages <= 1) return null

  return (
    <nav className="pagination" aria-label="Paginación">
      <button onClick={() => onChange(1)} disabled={page === 1} title="Primera">
        &laquo;
      </button>
      <button onClick={() => onChange(page - 1)} disabled={page === 1} title="Anterior">
        &lsaquo;
      </button>

      {pageList(page, totalPages).map((p, i) =>
        p === '…' ? (
          <span key={`e${i}`} className="page-info" style={{ margin: 0 }}>…</span>
        ) : (
          <button
            key={p}
            className={p === page ? 'active' : ''}
            onClick={() => onChange(p)}
          >
            {p}
          </button>
        )
      )}

      <button onClick={() => onChange(page + 1)} disabled={page === totalPages} title="Siguiente">
        &rsaquo;
      </button>
      <button onClick={() => onChange(totalPages)} disabled={page === totalPages} title="Última">
        &raquo;
      </button>
    </nav>
  )
}