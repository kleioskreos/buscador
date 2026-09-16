"""Convierte el .xlsb del BTR a un TSV canonico de 27 columnas usando pyxlsb.

Este script es la version local de /app/xlsb_to_tsv.py del backend,
con el fix de get_sheet(0) -> get_sheet(wb.sheets[0]) aplicado.

Uso:
    python scripts/xlsb_to_tsv_local.py [entrada] [salida]

Por defecto:
    entrada = C:\github\workbuddy\BTR_202604.xlsb
    salida  = C:\github\workbuddy\Buscador de apis\db\seed\btr_data.tsv
"""
import os
import sys
import time

try:
    from pyxlsb import open_workbook
except ImportError:
    sys.stderr.write("ERROR: falta pyxlsb. Instalar con: pip install pyxlsb\n")
    sys.exit(3)

INP = sys.argv[1] if len(sys.argv) > 1 else r"C:\github\workbuddy\BTR_202604.xlsb"
OUT = sys.argv[2] if len(sys.argv) > 2 else r"C:\github\workbuddy\Buscador de apis\db\seed\btr_data.tsv"

COLS = [
    'PERPAGO', 'MODULAR', 'SECUENCIAL', 'NDOCUMENTO', 'PATERNO', 'MATERNO',
    'NOMBRES', 'OFICINA', 'DTSERVIDOR', 'NOMBRE_IE', 'DES_CARGO', 'REGLAB',
    'JORLABORAL', 'SEXO', 'DSITUACION', 'THABER', 'TDESCUENTO', 'TLIQUIDO',
    'PLAZA', 'DESC_PLAZA', 'DPLANILLA', 'FNACIMIENT', 'FINGRESO', 'COD_IE',
    'COD_CARGO', 'CREG_PENS', 'CUSSP',
]


def norm(s):
    if s is None:
        return ''
    return ''.join(c for c in str(s).upper() if c.isalnum())


def sanitize(v):
    if v is None:
        return ''
    if isinstance(v, float) and v.is_integer():
        v = int(v)
    s = str(v).strip()
    s = s.replace('\\', ' ').replace('\t', ' ').replace('\r', ' ').replace('\n', ' ')
    while '  ' in s:
        s = s.replace('  ', ' ')
    return s


def main():
    if not os.path.exists(INP):
        sys.stderr.write(f"ERROR: no existe {INP}\n")
        sys.exit(2)

    print(f"[xlsb_to_tsv_local] leyendo {INP} ...", file=sys.stderr)
    t0 = time.time()
    with open_workbook(INP) as wb:
        if not wb.sheets:
            sys.stderr.write("ERROR: el archivo no tiene hojas.\n")
            sys.exit(4)
        sheet_name = wb.sheets[0]
        sh = wb.get_sheet(sheet_name)
        print(f"[xlsb_to_tsv_local] hoja: {sheet_name}", file=sys.stderr)
        rows = sh.rows()

        try:
            header_row = next(rows)
        except StopIteration:
            sys.stderr.write("ERROR: archivo vacio.\n")
            sys.exit(5)

        hnames = [c.v for c in header_row]
        hmap = {}
        for i, h in enumerate(hnames):
            hmap.setdefault(norm(h), i)

        idx = []
        unmapped = []
        for col in COLS:
            j = hmap.get(norm(col))
            if j is None:
                j = COLS.index(col)
                unmapped.append(col)
            idx.append(j)
        if unmapped:
            print(f"[xlsb_to_tsv_local] AVISO sin nombre, usando posicion: {', '.join(unmapped)}", file=sys.stderr)

        os.makedirs(os.path.dirname(OUT), exist_ok=True)
        count = 0
        with open(OUT, 'w', encoding='utf-8', newline='') as f:
            f.write('\t'.join(COLS) + '\n')
            for row in rows:
                vals = [c.v for c in row]
                if not any(v is not None and str(v).strip() != '' for v in vals):
                    continue
                out = []
                for j in idx:
                    v = vals[j] if j < len(vals) else ''
                    out.append(sanitize(v))
                f.write('\t'.join(out) + '\n')
                count += 1
                if count % 100000 == 0:
                    print(f"[xlsb_to_tsv_local] {count:,} filas ({time.time()-t0:.0f}s)", file=sys.stderr)

    elapsed = time.time() - t0
    print(f"[xlsb_to_tsv_local] OK: {count:,} filas -> {OUT} ({elapsed:.0f}s)", file=sys.stderr)


if __name__ == '__main__':
    main()
