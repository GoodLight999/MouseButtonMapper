@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command ^
  "Stop-Process -Name MouseButtonMapper -Force -ErrorAction SilentlyContinue; Remove-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name 'MouseButtonMapper' -ErrorAction SilentlyContinue; Remove-Item (Join-Path ([Environment]::GetFolderPath('Startup')) 'MouseButtonMapper.lnk') -Force -ErrorAction SilentlyContinue; Remove-Item (Join-Path $env:LOCALAPPDATA 'Programs\MouseButtonMapper') -Recurse -Force -ErrorAction SilentlyContinue; Write-Host 'Legacy installation removed. %LOCALAPPDATA%\MouseButtonMapper\config.json was kept.'"
endlocal
pause
