"""Convierte el .xlsb (en realidad .xlsx) del BTR a un TSV canonico de 27 columnas.

Uso:
    python scripts/xlsx_to_tsv.py [entrada] [salida]

Por defecto:
    entrada = C:\github\workbuddy\BTR_202604.xlsb
    salida  = C:\github\workbuddy\Buscador de apis\db\seed\btr_data.tsv

NOTA: este script usa openpyxl (lee .xlsx nativo, no requiere Excel).
El archivo BTR_202604.xlsb tiene magic bytes ZIP, asi que en realidad es .xlsx.
pyxlsb (que usa xlsb_to_tsv.py del backend) no puede leerlo porque busca firma OLE.
"""
import sys
import os
import time

try:
    from openpyxl import load_workbook
except ImportError:
    sys.stderr.write("ERROR: falta openpyxl. Instalar con: pip install openpyxl\n")
    sys.exit(3)

INP  = sys.argv[1] if len(sys.argv) > 1 else r"C:\github\workbuddy\BTR_202604.xlsb"
OUT  = sys.argv[2] if len(sys.argv) > 2 else r"C:\github\workbuddy\Buscador de apis\db\seed\btr_data.tsv"

# Orden canonico que espera el LOAD DATA del backend.
COLS = [
    'PERPAGO', 'MODULAR', 'SECUENCIAL', 'NDOCUMENTO', 'PATERNO', 'MATERNO',
    'NOMBRES', 'OFICINA', 'DTSERVIDOR', 'NOMBRE_IE', 'DES_CARGO', 'REGLAB',
    'JORLABORAL', 'SEXO', 'DSITUACION', 'THABER', 'TDESCUENTO', 'TLIQUIDO',
    'PLAZA', 'DESC_PLAZA', 'DPLANILLA', 'FNACIMIENT', 'FINGRESO', 'COD_IE',
    'COD_CARGO', 'CREG_PENS', 'CUSSP',
]


def norm(s):
    """Normaliza un nombre de columna (sin case/spaces/accents) para comparar."""
    if s is None:
        return ''
    return ''.join(c for c in str(s).upper() if c.isalnum())


def sanitize(v):
    """Deja el valor TSV-seguro."""
    if v is None:
        return ''
    # openpyxl entrega fechas como datetime; las pasamos a string ISO si lo son.
    if hasattr(v, 'isoformat'):
        v = v.isoformat(sep=' ')[:10]
    # Enteros sin ".0"
    if isinstance(v, float) and v.is_integer():
        v = int(v)
    s = str(v).strip()
    s = s.replace('\\', ' ').replace('\t', ' ').replace('\r', ' ').replace('\n', ' ')
    while '  ' in s:
        s = s.replace('  ', ' ')
    return s


def main():
    if not os.path.exists(INP):
        sys.stderr.write(f"ERROR: no existe el archivo de entrada: {INP}\n")
        sys.exit(2)

    print(f"[xlsx_to_tsv] leyendo {INP} ...", file=sys.stderr)
    # read_only=True evita cargar todo en memoria (clave para archivos de 137 MB)
    wb = load_workbook(INP, read_only=True, data_only=True)
    sh = wb.active
    if sh is None:
        sys.stderr.write("ERROR: el archivo no tiene hojas activas.\n")
        sys.exit(4)

    print(f"[xlsx_to_tsv] hoja activa: {sh.title}", file=sys.stderr)

    os.makedirs(os.path.dirname(OUT), exist_ok=True)
    t0 = time.time()
    count = 0

    with open(OUT, 'w', encoding='utf-8', newline='') as f:
        f.write('\t'.join(COLS) + '\n')

        # Primera fila = cabecera
        rows = sh.iter_rows(values_only=True)
        try:
            header_row = next(rows)
        except StopIteration:
            sys.stderr.write("ERROR: el archivo esta vacio.\n")
            sys.exit(5)

        hnames = list(header_row)
        hmap = {}
        for i, h in enumerate(hnames):
            hmap.setdefault(norm(h), i)

        idx = []
        unmapped = []
        for col in COLS:
            j = hmap.get(norm(col))
            if j is None:
                # fallback posicional (por si la cabecera cambia de nombre)
                j = COLS.index(col)
                unmapped.append(col)
            idx.append(j)

        if unmapped:
            print(
                f"[xlsx_to_tsv] AVISO: sin coincidencia de nombre, se usara "
                f"posicion para: {', '.join(unmapped)}",
                file=sys.stderr,
            )

        for row in rows:
            vals = list(row)
            if not any(v is not None and str(v).strip() != '' for v in vals):
                continue  # fila vacia
            out = []
            for j in idx:
                v = vals[j] if j < len(vals) else ''
                out.append(sanitize(v))
            f.write('\t'.join(out) + '\n')
            count += 1
            if count % 100000 == 0:
                print(f"[xlsx_to_tsv] {count:,} filas ({time.time()-t0:.0f}s)", file=sys.stderr)

    elapsed = time.time() - t0
    print(
        f"[xlsx_to_tsv] OK: {count:,} filas -> {OUT} ({elapsed:.0f}s)",
        file=sys.stderr,
    )


if __name__ == '__main__':
    main()
