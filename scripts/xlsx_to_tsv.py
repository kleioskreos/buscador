"""DEPRECATED: este script NO funciona para .xlsb.

Usaba openpyxl, que NO soporta el formato binario .xlsb (OLE Compound File).
El archivo BTR_202604.xlsb parecia .xlsx (magic bytes ZIP) pero openpyxl igual
lo rechaza con 'InvalidFileException' por la extension.

La version correcta para .xlsb esta en scripts/xlsb_to_tsv.py.

Si queres convertir un .xlsx (no .xlsb) a TSV, podes hacerlo directamente
desde Excel (Guardar como > Texto separado por tabuladores).
"""
import sys
sys.stderr.write(
    "DEPRECATED: este script no sirve para archivos .xlsb. "
    "Usa scripts/xlsb_to_tsv.py (que SI funciona) o converti el archivo "
    "desde Excel a Texto separado por tabuladores.\n"
)
sys.exit(1)
