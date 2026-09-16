# Convierte el BTR_202604.xlsb (mestizo ZIP+binario) a un .xlsx real.
# El archivo original tiene la firma ZIP pero contiene partes .bin de .xlsb
# adentro, lo cual confunde a openpyxl y pyxlsb. Excel lo reescribe como
# .xlsx puro (todo XML adentro del ZIP) sin problema.
$ErrorActionPreference = 'Stop'

$src = "C:\github\workbuddy\Buscador de apis\BTR_202604_local.xlsb"
$dst = "C:\github\workbuddy\Buscador de apis\BTR_202604_clean.xlsx"

if (Test-Path $dst) { Remove-Item $dst -Force }

Write-Host "[1/3] Abriendo Excel en background..." -ForegroundColor Cyan
$xl = New-Object -ComObject Excel.Application
$xl.Visible = $false
$xl.DisplayAlerts = $false
$xl.ScreenUpdating = $false

Write-Host "[2/3] Abriendo el archivo fuente (puede tardar con 137 MB)..." -ForegroundColor Cyan
$wb = $xl.Workbooks.Open($src, $false, $true)  # ReadOnly=$false, FormatLinks=$true

Write-Host "[3/3] Guardando como .xlsx real (xlFileFormatXMLWorkbook = 51)..." -ForegroundColor Cyan
# 51 = xlFileFormatXMLWorkbook (.xlsx standard)
# 50 = xlFileFormatXLSB (.xlsb real, tambien sirve)
$wb.SaveAs($dst, 51)
$wb.Close($false)
$xl.Quit()
[System.Runtime.InteropServices.Marshal]::ReleaseComObject($xl) | Out-Null

Write-Host "OK: $dst" -ForegroundColor Green
Get-Item $dst | Select-Object Name, Length, LastWriteTime | Format-List
