#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
xlsb_to_tsv.py — Convierte el archivo .xlsb del BTR (MINEDU, Peru) a un TSV
canonico de 27 columnas que el backend Go puede cargar con LOAD DATA INFILE.

Este script es la version para usar EN TU MAQUINA LOCAL (Windows/Mac/Linux)
para regenerar la semilla antes de hacer import. La version para correr
adentro del container esta en backend/xlsb_to_tsv.py y se invoca
automaticamente cuando subis un .xlsb al buscador.

USO:
    # Desde la raiz del proyecto (Windows):
    python scripts/xlsb_to_tsv.py

    # Pasando paths explicitos:
    python scripts/xlsb_to_tsv.py ruta/al/archivo.xlsb ruta/al/output.tsv

    # Con Python 3 desde CUALQUIER directorio (usando forward slashes):
    python /c/github/workbuddy/Buscador\ de\ apis/scripts/xlsb_to_tsv.py

DEPENDENCIAS:
    pip install pyxlsb

OUTPUT:
    Escribe un TSV con la cabecera canonica de 27 columnas (la primera linea)
    seguida de una fila por cada registro del .xlsb. El archivo esta listo
    para subir al buscador desde el boton "Importar datos" o para copiar
    a db/seed/btr_data.tsv (no recomendado: 160 MB > limite de git).

NOTAS TECNICAS:
    - Por que pyxlsb a veces falla con "IndexError: sheet index out of range"
      y como lo evitamos: ver bloque mas abajo.
    - Las 27 columnas canonicas las sacamos del schema MySQL (db/init/01-schema.sql)
      y del orden usado por el LOAD DATA (backend/db.go).
    - Mapeamos por NOMBRE de columna (no por posicion), asi si el .xlsb
      original agrega columnas nuevas o reordena, el script sigue funcionando.

HISTORIAL:
    - 2026-09-15: Creado para evitar el bug "IndexError: sheet index out of
      range" que aparecia al cargar el archivo BTR_202604.xlsb del 2026 desde
      el buscador desplegado en Dokploy.
"""
import os
import sys
import time

try:
    from pyxlsb import open_workbook
except ImportError:
    sys.stderr.write(
        "ERROR: falta la dependencia 'pyxlsb'.\n"
        "Instalala con:  pip install pyxlsb\n"
    )
    sys.exit(3)


# Orden canonico que espera el LOAD DATA del backend Go.
# DEBE coincidir con `loadCols` en backend/db.go y con el header que genera
# backend/xlsb_to_tsv.py cuando convierte un .xlsb subido por la UI.
COLS = [
    'PERPAGO', 'MODULAR', 'SECUENCIAL', 'NDOCUMENTO', 'PATERNO', 'MATERNO',
    'NOMBRES', 'OFICINA', 'DTSERVIDOR', 'NOMBRE_IE', 'DES_CARGO', 'REGLAB',
    'JORLABORAL', 'SEXO', 'DSITUACION', 'THABER', 'TDESCUENTO', 'TLIQUIDO',
    'PLAZA', 'DESC_PLAZA', 'DPLANILLA', 'FNACIMIENT', 'FINGRESO', 'COD_IE',
    'COD_CARGO', 'CREG_PENS', 'CUSSP',
]

# Firma magica de un archivo OLE Compound File (formato real de .xlsb):
#   D0 CF 11 E0 A1 B1 1A E1
# Si los primeros 8 bytes NO coinciden, el archivo NO es .xlsb (puede ser
# .xlsx, .csv, .pdf renombrado, etc.) y pyxlsb va a fallar.
XLSB_MAGIC = b'\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1'

# Paths por defecto pensados para el setup tipico del proyecto en Windows.
# Se pueden override por linea de comandos.
DEFAULT_INP = r"C:\github\workbuddy\BTR_202604.xlsb"
DEFAULT_OUT = os.path.join(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
    'db', 'seed', 'btr_data.tsv',
)


def norm(s):
    """Normaliza un nombre de columna para comparar (sin case ni simbolos)."""
    if s is None:
        return ''
    return ''.join(c for c in str(s).upper() if c.isalnum())


def sanitize(v):
    """Deja el valor TSV-seguro (sin tab/CR/LF/backslash, sin espacios dobles)."""
    if v is None:
        return ''
    # pyxlsb entrega fechas como datetime; serial Excel como float
    if hasattr(v, 'isoformat'):
        v = v.isoformat(sep=' ')[:10]
    if isinstance(v, float) and v.is_integer():
        v = int(v)
    s = str(v).strip()
    s = s.replace('\\', ' ').replace('\t', ' ').replace('\r', ' ').replace('\n', ' ')
    while '  ' in s:
        s = s.replace('  ', ' ')
    return s


def usage():
    print(__doc__, file=sys.stderr)
    print(f"Uso:  python {sys.argv[0]} [entrada.xlsb] [salida.tsv]", file=sys.stderr)
    print(f"Por defecto:", file=sys.stderr)
    print(f"  entrada: {DEFAULT_INP}", file=sys.stderr)
    print(f"  salida:  {DEFAULT_OUT}", file=sys.stderr)


def main():
    if len(sys.argv) > 1 and sys.argv[1] in ('-h', '--help', '/?'):
        usage()
        sys.exit(0)

    inp = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_INP
    out = sys.argv[2] if len(sys.argv) > 2 else DEFAULT_OUT

    # --- Validacion de entrada ---
    if not os.path.exists(inp):
        sys.stderr.write(f"ERROR: no existe el archivo de entrada: {inp}\n")
        sys.exit(2)

    size_mb = os.path.getsize(inp) / (1024 * 1024)
    print(f"[xlsb_to_tsv] entrada: {inp} ({size_mb:.1f} MB)", file=sys.stderr)
    print(f"[xlsb_to_tsv] salida:  {out}", file=sys.stderr)

    # --- Validacion de magic bytes ---
    with open(inp, 'rb') as f:
        magic = f.read(8)
    if magic != XLSB_MAGIC:
        sys.stderr.write(
            f"\nERROR: '{inp}' NO es un .xlsb real.\n"
            f"  Magic bytes encontrados: {magic.hex(' ').upper()}\n"
            f"  Magic bytes esperados:   {XLSB_MAGIC.hex(' ').upper()} (OLE Compound File)\n\n"
            f"Posibles causas:\n"
            f"  - El archivo es .xlsx (formato ZIP/XML) con extension cambiada.\n"
            f"  - El archivo esta corrupto o es una descarga incompleta.\n"
            f"  - El archivo es otro formato (CSV, PDF, etc.) renombrado.\n\n"
            f"Solucion:\n"
            f"  - Abrilo con Excel y 'Guardar como > Libro binario de Excel (.xlsb)'.\n"
            f"  - O convertilo a .tsv directamente desde Excel (Texto separado por tabuladores).\n"
        )
        sys.exit(4)

    # --- Conversion ---
    print(f"[xlsb_to_tsv] abriendo workbook...", file=sys.stderr)
    t0 = time.time()
    with open_workbook(inp) as wb:
        if not wb.sheets:
            sys.stderr.write(
                f"\nERROR: '{inp}' no tiene hojas.\n"
                f"  El archivo es .xlsb valido pero esta vacio.\n"
            )
            sys.exit(5)

        # FIX IMPORTANTE: usar wb.get_sheet(NOMBRE) en lugar de wb.get_sheet(0).
        # pyxlsb lanza "IndexError: sheet index out of range" con algunos archivos
        # .xlsb (tipicamente los generados por herramientas que envuelven el
        # workbook en formatos no estandar) cuando se accede por indice.
        # Acceder por nombre (que es lo que devuelve wb.sheets[0]) funciona
        # siempre. Esta misma fix esta aplicada en backend/xlsb_to_tsv.py.
        sheet_name = wb.sheets[0]
        sh = wb.get_sheet(sheet_name)
        print(f"[xlsb_to_tsv] hoja activa: {sheet_name!r}", file=sys.stderr)

        rows = sh.rows()
        try:
            header_row = next(rows)
        except StopIteration:
            sys.stderr.write("ERROR: el archivo esta vacio (sin filas).\n")
            sys.exit(6)

        hnames = [c.v for c in header_row]
        print(
            f"[xlsb_to_tsv] header: {len(hnames)} columnas detectadas en el .xlsb",
            file=sys.stderr,
        )

        # Mapa nombre-normalizado -> indice de columna en el .xlsb.
        # Usamos setdefault para tomar la primera coincidencia si hay nombres duplicados.
        hmap = {}
        for i, h in enumerate(hnames):
            hmap.setdefault(norm(h), i)

        # Para cada columna canonica, resolver su indice en el .xlsb.
        idx = []
        unmapped = []
        for col in COLS:
            j = hmap.get(norm(col))
            if j is None:
                # Fallback posicional SOLO si el nombre no esta en el header.
                # Esto es un safety net, no el caso normal.
                j = COLS.index(col)
                unmapped.append(col)
            idx.append(j)

        if unmapped:
            print(
                f"[xlsb_to_tsv] AVISO: columnas canonicas sin nombre en el header, "
                f"se uso la posicion como fallback: {', '.join(unmapped)}",
                file=sys.stderr,
            )

        # --- Escritura del TSV ---
        os.makedirs(os.path.dirname(out), exist_ok=True)
        count = 0
        with open(out, 'w', encoding='utf-8', newline='') as f:
            f.write('\t'.join(COLS) + '\n')
            for row in rows:
                vals = [c.v for c in row]
                # Saltar filas totalmente vacias (comunes al final de los .xlsb)
                if not any(v is not None and str(v).strip() != '' for v in vals):
                    continue
                out_cells = []
                for j in idx:
                    v = vals[j] if j < len(vals) else ''
                    out_cells.append(sanitize(v))
                f.write('\t'.join(out_cells) + '\n')
                count += 1
                if count % 100000 == 0:
                    print(
                        f"[xlsb_to_tsv] {count:,} filas ({time.time()-t0:.0f}s)",
                        file=sys.stderr,
                    )

    elapsed = time.time() - t0
    size_out_mb = os.path.getsize(out) / (1024 * 1024)
    print(
        f"\n[xlsb_to_tsv] OK: {count:,} filas -> {out} ({size_out_mb:.1f} MB, {elapsed:.0f}s)",
        file=sys.stderr,
    )
    print(
        f"[xlsb_to_tsv] siguiente paso: importar el .tsv desde la UI del buscador "
        f"(boton 'Importar datos') o copiarlo a db/seed/ para auto-carga al levantar el stack.",
        file=sys.stderr,
    )


if __name__ == '__main__':
    main()
