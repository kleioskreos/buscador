"""Export BTR records from SQLite to TSV for MySQL LOAD DATA INFILE."""
import sqlite3, os, sys

DB = r"C:\github\workbuddy\btr_search\btr.db"
OUT = r"C:\github\workbuddy\btr_docker\db\seed\btr_data.tsv"

COLS = ['PERPAGO','MODULAR','SECUENCIAL','NDOCUMENTO','PATERNO','MATERNO','NOMBRES',
        'OFICINA','DTSERVIDOR','NOMBRE_IE','DES_CARGO','REGLAB','JORLABORAL','SEXO',
        'DSITUACION','THABER','TDESCUENTO','TLIQUIDO','PLAZA','DESC_PLAZA','DPLANILLA',
        'FNACIMIENT','FINGRESO','COD_IE','COD_CARGO','CREG_PENS','CUSSP']

os.makedirs(os.path.dirname(OUT), exist_ok=True)

def sanitize(v):
    if v is None:
        return ''
    s = str(v).strip()
    # TSV-safe: remove tabs, CR, LF, and backslash (MySQL escape char)
    s = s.replace('\\', ' ').replace('\t', ' ').replace('\r', ' ').replace('\n', ' ')
    # collapse multiple spaces
    while '  ' in s:
        s = s.replace('  ', ' ')
    return s

conn = sqlite3.connect(DB)
cur = conn.cursor()

count = 0
t0 = __import__('time').time()
with open(OUT, 'w', encoding='utf-8', newline='') as f:
    f.write('\t'.join(COLS) + '\n')
    for row in cur.execute("SELECT %s FROM records" % ','.join(COLS)):
        f.write('\t'.join(sanitize(v) for v in row) + '\n')
        count += 1
        if count % 100000 == 0:
            print("%d rows (%.0fs)" % (count, __import__('time').time()-t0), file=sys.stderr)

conn.close()
print("Done: %d rows -> %s" % (count, OUT), file=sys.stderr)