@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command ^
  "$lnk=Join-Path ([Environment]::GetFolderPath('Startup')) 'MouseButtonMapper.lnk'; Remove-Item $lnk -Force -ErrorAction SilentlyContinue; Write-Host ('Removed: '+$lnk)"
endlocal
pause
