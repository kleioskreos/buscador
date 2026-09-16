#!/usr/bin/env python3
# ============================================================
# xlsb_to_tsv.py — Convierte BTR .xlsb -> TSV canonico (27 cols)
#
# El TSV resultante tiene la cabecera con los 27 nombres canoncos y una fila
# por registro, separado por TAB, listo para el LOAD DATA de MySQL.
# Los valores se escriben como texto plano (sin reformatear) para respetar el
# esquema (THABER/TDESCUENTO/TLIQUIDO son DECIMAL, FNACIMIENT/FINGRESO VARCHAR).
#
# Uso:  python3 xlsb_to_tsv.py <entrada.xlsb> <salida.tsv>
# ============================================================
import sys

try:
    from pyxlsb import open_workbook
except ImportError:
    sys.stderr.write("ERROR: falta pyxlsb (pip install pyxlsb)\n")
    sys.exit(3)

# Orden canonico esperado por el LOAD DATA del backend.
COLS = [
    'PERPAGO', 'MODULAR', 'SECUENCIAL', 'NDOCUMENTO', 'PATERNO', 'MATERNO',
    'NOMBRES', 'OFICINA', 'DTSERVIDOR', 'NOMBRE_IE', 'DES_CARGO', 'REGLAB',
    'JORLABORAL', 'SEXO', 'DSITUACION', 'THABER', 'TDESCUENTO', 'TLIQUIDO',
    'PLAZA', 'DESC_PLAZA', 'DPLANILLA', 'FNACIMIENT', 'FINGRESO', 'COD_IE',
    'COD_CARGO', 'CREG_PENS', 'CUSSP',
]


def norm(s):
    """Normaliza un nombre de columna para comparar ignoring case/spaces/accents."""
    if s is None:
        return ''
    out = []
    for ch in str(s).upper():
        if ch.isalnum():
            out.append(ch)
    return ''.join(out)


def sanitize(v):
    """Deja el valor TSV-seguro (sin tab/CR/LF/backslash y sin espacios dobles)."""
    if v is None:
        return ''
    # pyxlsb entrega numeros como float; los enteros se escriben sin ".0".
    if isinstance(v, float) and v.is_integer():
        v = int(v)
    s = str(v).strip()
    s = s.replace('\\', ' ').replace('\t', ' ').replace('\r', ' ').replace('\n', ' ')
    while '  ' in s:
        s = s.replace('  ', ' ')
    return s


def main():
    if len(sys.argv) < 3:
        sys.stderr.write("uso: xlsb_to_tsv.py <entrada.xlsb> <salida.tsv>\n")
        sys.exit(2)

    inp, outp = sys.argv[1], sys.argv[2]

    with open_workbook(inp) as wb:
        # NOTA: usamos wb.sheets[0] (el nombre) en lugar de get_sheet(0) (el indice).
        # En archivos .xlsb reales con formato OLE funciona ambos, pero en archivos
        # donde el workbook viene envuelto en un ZIP no estandar (firma 50 4B 03 04
        # en lugar de D0 CF 11 E0), get_sheet(0) lanza IndexError aunque la hoja
        # exista. get_sheet(<nombre>) funciona siempre.
        if not wb.sheets:
            sys.stderr.write("ERROR: el archivo no tiene hojas. Posiblemente corrupto.\n")
            sys.exit(4)
        sheet_name = wb.sheets[0]
        sh = wb.get_sheet(sheet_name)
        sys.stderr.write("hoja activa: %s\n" % sheet_name)
        rows = sh.rows()

        # --- Cabecera: mapear cada columna canonica a su indice en el xlsb ---
        header = next(rows)
        hnames = [c.v for c in header]
        hmap = {}
        for i, h in enumerate(hnames):
            hmap.setdefault(norm(h), i)  # primera coincidencia gana

        idx = []
        unmapped = []
        for col in COLS:
            j = hmap.get(norm(col))
            if j is None:
                j = COLS.index(col)  # reverso posicional si no hay nombre
                unmapped.append(col)
            idx.append(j)

        if unmapped:
            sys.stderr.write(
                "AVISO: sin coincidencia de nombre, se usara posicion para: %s\n"
                % ', '.join(unmapped)
            )

        nsrc = len(hnames)
        count = 0
        with open(outp, 'w', encoding='utf-8', newline='') as f:
            f.write('\t'.join(COLS) + '\n')
            for row in rows:
                vals = [c.v for c in row]
                if not any(v is not None and str(v).strip() != '' for v in vals):
                    continue  # fila vacia
                out = []
                for j in idx:
                    v = vals[j] if j < len(vals) else ''
                    out.append(sanitize(v))
                f.write('\t'.join(out) + '\n')
                count += 1

        sys.stderr.write("convertido: %d filas -> %s\n" % (count, outp))


if __name__ == '__main__':
    main()
