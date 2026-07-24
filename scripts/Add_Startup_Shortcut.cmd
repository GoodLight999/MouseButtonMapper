@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command ^
  "$exe=(Resolve-Path '%~dp0MouseButtonMapper.exe').Path; $startup=[Environment]::GetFolderPath('Startup'); $lnk=Join-Path $startup 'MouseButtonMapper.lnk'; $w=New-Object -ComObject WScript.Shell; $s=$w.CreateShortcut($lnk); $s.TargetPath=$exe; $s.Arguments='--tray'; $s.WorkingDirectory=Split-Path $exe; $s.IconLocation=$exe+',0'; $s.Save(); Write-Host ('Created: '+$lnk)"
endlocal
pause
