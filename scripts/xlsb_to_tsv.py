#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
xlsb_to_tsv.py — Convierte archivos .xlsb (formato binario OLE Compound File)
a TSV canonico de 27 columnas, pensado para planillas tipo MINEDU/BTR.

USO:

  # Doble click sobre el archivo, o sin argumentos: abre una GUI simple
  python scripts/xlsb_to_tsv.py

  # CLI: pasando el archivo de entrada (y opcionalmente el de salida)
  python scripts/xlsb_to_tsv.py ruta/al/archivo.xlsb
  python scripts/xlsb_to_tsv.py ruta/al/archivo.xlsb ruta/al/salida.tsv

DEPENDENCIAS:
    pip install pyxlsb
    # tkinter viene con Python en Windows y macOS.
    # En Linux: sudo apt install python3-tk

SALIDA (modo GUI):
    El TSV se genera en la MISMA CARPETA del archivo .xlsb, con la extension
    cambiada de .xlsb a .tsv. Ej: planilla.xlsb -> planilla.tsv

SALIDA (modo CLI):
    Si no se pasa output, se usa la misma convencion (misma carpeta, .tsv).
    Si se pasa, se usa ese path.

NOTAS TECNICAS:
    - Por que pyxlsb a veces falla con "IndexError: sheet index out of range":
      ver bloque "EL BUG" mas abajo.
    - Las 27 columnas canonicas se mapean por NOMBRE (no por posicion), asi si
      el archivo agrega o reordena columnas, el script sigue funcionando.
    - Si el archivo no es un .xlsb real (magic bytes incorrectos), el script
      se niega a procesarlo y da un mensaje claro. Esto evita perder tiempo
      con archivos renombrados (.xlsx, .csv, .pdf como .xlsb).

EL BUG (referencia historica, 2026-09):
    pyxlsb.open_workbook() puede lanzar "IndexError: sheet index out of range"
    al llamar wb.get_sheet(0), aunque el archivo tenga sheets. Esto pasa con
    archivos .xlsb no estandar (envoltorio ZIP raro, etc.). La solucion es
    acceder por NOMBRE en vez de por indice:
        sheet_name = wb.sheets[0]      # devuelve el nombre de la primera hoja
        sh = wb.get_sheet(sheet_name)  # funciona siempre
"""
import os
import sys
import time

# Orden canonico que espera el LOAD DATA del backend (backend/db.go) cuando se
# carga el TSV. 27 columnas. El script las busca por NOMBRE en el header del
# .xlsb; si alguna no esta, usa la posicion como fallback (con un aviso).
COLS = [
    'PERPAGO', 'MODULAR', 'SECUENCIAL', 'NDOCUMENTO', 'PATERNO', 'MATERNO',
    'NOMBRES', 'OFICINA', 'DTSERVIDOR', 'NOMBRE_IE', 'DES_CARGO', 'REGLAB',
    'JORLABORAL', 'SEXO', 'DSITUACION', 'THABER', 'TDESCUENTO', 'TLIQUIDO',
    'PLAZA', 'DESC_PLAZA', 'DPLANILLA', 'FNACIMIENT', 'FINGRESO', 'COD_IE',
    'COD_CARGO', 'CREG_PENS', 'CUSSP',
]

# Firma magica OLE Compound File (formato real de .xlsb):
#   D0 CF 11 E0 A1 B1 1A E1
# Si los primeros 8 bytes NO coinciden, NO es .xlsb.
XLSB_MAGIC = b'\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1'


def norm(s):
    """Normaliza un nombre de columna (sin case/simbolos) para comparar."""
    if s is None:
        return ''
    return ''.join(c for c in str(s).upper() if c.isalnum())


def sanitize(v):
    """Deja el valor TSV-seguro."""
    if v is None:
        return ''
    if hasattr(v, 'isoformat'):
        v = v.isoformat(sep=' ')[:10]
    if isinstance(v, float) and v.is_integer():
        v = int(v)
    s = str(v).strip()
    s = s.replace('\\', ' ').replace('\t', ' ').replace('\r', ' ').replace('\n', ' ')
    while '  ' in s:
        s = s.replace('  ', ' ')
    return s


def derive_output_path(inp):
    """Genera el path de salida: misma carpeta que el input, extension .tsv."""
    base, _ = os.path.splitext(inp)
    return base + '.tsv'


def convert(inp, out, log=None, progress=None):
    """
    Convierte inp.xlsb -> out.tsv.

    Args:
        inp: ruta al archivo .xlsb
        out: ruta al archivo .tsv de salida
        log: funcion opcional log(msg) para mostrar mensajes (GUI o CLI)
        progress: funcion opcional progress(count, elapsed_seconds) llamada
                  cada 100k filas. None = modo silencioso.

    Returns:
        dict con 'rows' (cantidad) y 'elapsed' (segundos)

    Raises:
        SystemExit con codigo 2..6 segun el tipo de error.
    """
    def say(msg):
        if log:
            log(msg)

    if not os.path.exists(inp):
        say(f"ERROR: no existe el archivo: {inp}")
        sys.exit(2)

    size_mb = os.path.getsize(inp) / (1024 * 1024)
    say(f"Entrada: {inp} ({size_mb:.1f} MB)")
    say(f"Salida:  {out}")

    # Validacion de magic bytes
    with open(inp, 'rb') as f:
        magic = f.read(8)
    if magic != XLSB_MAGIC:
        say("")
        say(f"ERROR: el archivo NO es un .xlsb real.")
        say(f"  Magic bytes encontrados: {magic.hex(' ').upper()}")
        say(f"  Magic bytes esperados:   {XLSB_MAGIC.hex(' ').upper()} (OLE Compound File)")
        say("")
        say("Posibles causas:")
        say("  - Es un .xlsx (formato ZIP/XML) con extension cambiada.")
        say("  - El archivo esta corrupto o es una descarga incompleta.")
        say("  - Es otro formato (CSV, PDF, etc.) renombrado a .xlsb.")
        say("")
        say("Sugerencias:")
        say("  - Abrilo con Excel y 'Guardar como > Libro binario de Excel (.xlsb)'.")
        say("  - O convertilo a .tsv desde Excel (Texto separado por tabuladores).")
        sys.exit(4)

    try:
        from pyxlsb import open_workbook
    except ImportError:
        say("")
        say("ERROR: falta la dependencia 'pyxlsb'.")
        say("Instalala con:  pip install pyxlsb")
        sys.exit(3)

    say("")
    say("Abriendo workbook...")
    t0 = time.time()
    with open_workbook(inp) as wb:
        if not wb.sheets:
            say("")
            say("ERROR: el archivo no tiene hojas (vacio o corrupto).")
            sys.exit(5)

        # FIX IMPORTANTE: usar wb.get_sheet(NOMBRE) en vez de wb.get_sheet(0).
        # pyxlsb lanza IndexError con algunos .xlsb no estandar cuando se
        # accede por indice. Acceder por nombre (devuelto por wb.sheets[0])
        # funciona siempre.
        sheet_name = wb.sheets[0]
        sh = wb.get_sheet(sheet_name)
        say(f"Hoja activa: {sheet_name!r}")
        rows = sh.rows()

        try:
            header_row = next(rows)
        except StopIteration:
            say("ERROR: el archivo esta vacio (sin filas).")
            sys.exit(6)

        hnames = [c.v for c in header_row]
        say(f"Header: {len(hnames)} columnas detectadas en el .xlsb")

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
            say(
                f"AVISO: columnas canonicas sin nombre en el header, "
                f"se uso la posicion como fallback: {', '.join(unmapped)}"
            )

        os.makedirs(os.path.dirname(out) or '.', exist_ok=True)
        count = 0
        with open(out, 'w', encoding='utf-8', newline='') as f:
            f.write('\t'.join(COLS) + '\n')
            for row in rows:
                vals = [c.v for c in row]
                if not any(v is not None and str(v).strip() != '' for v in vals):
                    continue
                cells = []
                for j in idx:
                    v = vals[j] if j < len(vals) else ''
                    cells.append(sanitize(v))
                f.write('\t'.join(cells) + '\n')
                count += 1
                if progress and count % 100000 == 0:
                    progress(count, time.time() - t0)

    elapsed = time.time() - t0
    size_out_mb = os.path.getsize(out) / (1024 * 1024)
    say("")
    say(f"OK: {count:,} filas -> {out} ({size_out_mb:.1f} MB, {elapsed:.0f}s)")
    return {'rows': count, 'elapsed': elapsed}


# ============================================================
# Modo CLI
# ============================================================

def run_cli(args):
    inp = args[0] if args else None
    if not inp:
        print("Uso: python xlsb_to_tsv.py <entrada.xlsb> [salida.tsv]", file=sys.stderr)
        print("Sin argumentos se abre la GUI. Use --help para mas info.", file=sys.stderr)
        sys.exit(1)

    out = args[1] if len(args) > 1 else derive_output_path(inp)
    convert(inp, out, log=lambda m: print(m, file=sys.stderr))


# ============================================================
# Modo GUI (tkinter)
# ============================================================

def run_gui():
    try:
        import tkinter as tk
        from tkinter import filedialog, messagebox, scrolledtext
    except ImportError:
        print(
            "ERROR: tkinter no esta disponible en este Python.\n"
            "  - En Windows/macOS viene con Python. Verifica que no estes usando un Python minimal.\n"
            "  - En Linux: sudo apt install python3-tk\n"
            "\n"
            "Como alternativa, usa el modo CLI:\n"
            "  python scripts/xlsb_to_tsv.py archivo.xlsb\n",
            file=sys.stderr,
        )
        sys.exit(7)

    root = tk.Tk()
    root.title("Convertir .xlsb a .tsv")
    root.geometry("640x460")
    root.minsize(500, 360)

    # Variable mutable para el path seleccionado
    state = {'inp': None}

    # ---------- Header ----------
    hdr = tk.Label(
        root,
        text="Convertir archivo .xlsb a .tsv",
        font=("Segoe UI", 14, "bold"),
    )
    hdr.pack(pady=(14, 4))

    sub = tk.Label(
        root,
        text="Pensado para planillas tipo MINEDU (27 columnas canonicas).",
        fg="gray",
        font=("Segoe UI", 9),
    )
    sub.pack()

    # ---------- File picker ----------
    picker_frame = tk.Frame(root)
    picker_frame.pack(pady=(10, 4), padx=20, fill="x")

    select_btn = tk.Button(
        picker_frame,
        text="Seleccionar archivo .xlsb...",
        command=lambda: on_select(),
        width=22,
        height=1,
    )
    select_btn.pack(side="left")

    file_label = tk.Label(
        picker_frame,
        text="(ningun archivo seleccionado)",
        fg="gray",
        anchor="w",
    )
    file_label.pack(side="left", padx=(10, 0), fill="x", expand=True)

    path_label = tk.Label(root, text="", fg="gray", font=("Consolas", 8), anchor="w")
    path_label.pack(padx=20, fill="x")

    # ---------- Convert button ----------
    convert_btn = tk.Button(
        root,
        text="Convertir a .tsv",
        command=lambda: on_convert(),
        state="disabled",
        bg="#1976D2",
        fg="white",
        activebackground="#1565C0",
        activeforeground="white",
        font=("Segoe UI", 10, "bold"),
        height=2,
        cursor="hand2",
    )
    convert_btn.pack(pady=(10, 8), padx=20, fill="x")

    # ---------- Log ----------
    log_label = tk.Label(root, text="Progreso:", anchor="w", font=("Segoe UI", 9, "bold"))
    log_label.pack(padx=20, anchor="w")

    log_text = scrolledtext.ScrolledText(root, height=12, width=70, font=("Consolas", 9))
    log_text.pack(padx=20, pady=(2, 14), fill="both", expand=True)

    def log(msg):
        log_text.insert("end", msg + "\n")
        log_text.see("end")
        root.update_idletasks()

    def on_select():
        path = filedialog.askopenfilename(
            title="Selecciona el archivo .xlsb",
            filetypes=[("Excel binario", "*.xlsb"), ("Todos los archivos", "*.*")],
        )
        if not path:
            return
        state['inp'] = path
        size_mb = os.path.getsize(path) / (1024 * 1024)
        file_label.config(
            text=f"📄  {os.path.basename(path)}  ({size_mb:,.1f} MB)",
            fg="black",
        )
        path_label.config(text=path)
        convert_btn.config(state="normal")
        log_text.delete("1.0", "end")
        log(f"Archivo seleccionado: {path}")
        log(f"Tamano: {size_mb:.1f} MB")
        log("")
        log("Listo para convertir. Click en 'Convertir a .tsv'.")

    def on_convert():
        inp = state['inp']
        if not inp:
            return
        out = derive_output_path(inp)

        # Bloquear UI mientras corre
        select_btn.config(state="disabled")
        convert_btn.config(state="disabled", text="Convirtiendo...")
        log_text.delete("1.0", "end")
        log(f"Entrada: {inp}")
        log(f"Salida:  {out}")
        log("")

        def progress(count, elapsed):
            log(f"  {count:,} filas ({elapsed:.0f}s)")

        try:
            convert(inp, out, log=log, progress=progress)
        except SystemExit as e:
            # Errores que el propio convert() lanza con sys.exit(N)
            log("")
            log(f"❌ Conversion fallida (codigo {e.code}).")
            messagebox.showerror(
                "Conversion fallida",
                f"La conversion fallo. Revisa los mensajes en el log.\n\n"
                f"Codigo de error: {e.code}",
            )
        except Exception as e:
            log("")
            log(f"❌ Error inesperado: {type(e).__name__}: {e}")
            messagebox.showerror("Error inesperado", f"{type(e).__name__}: {e}")
        else:
            log("")
            log("✅ Conversion exitosa.")
            log(f"Archivo generado: {out}")
            # Preguntar si quiere abrir la carpeta
            if messagebox.askyesno(
                "Listo",
                f"Se genero el TSV en:\n{out}\n\n"
                f"Quieres abrir la carpeta donde quedo?",
            ):
                try:
                    folder = os.path.dirname(out) or '.'
                    if sys.platform.startswith('win'):
                        os.startfile(folder)
                    elif sys.platform == 'darwin':
                        os.system(f'open "{folder}"')
                    else:
                        os.system(f'xdg-open "{folder}"')
                except Exception as e:
                    log(f"(No pude abrir la carpeta: {e})")
        finally:
            select_btn.config(state="normal")
            convert_btn.config(state="normal", text="Convertir a .tsv")

    root.mainloop()


# ============================================================
# Entry point
# ============================================================

def usage():
    print(__doc__)


def main():
    if len(sys.argv) > 1 and sys.argv[1] in ('-h', '--help', '/?'):
        usage()
        sys.exit(0)

    # Sin argumentos -> GUI. Con argumentos -> CLI.
    if len(sys.argv) > 1:
        run_cli(sys.argv[1:])
    else:
        run_gui()


if __name__ == '__main__':
    main()
