param(
    [string]$Version = "8.4.0-rc5",
    [string]$OutputDirectory = "dist"
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

& "$PSScriptRoot\build.ps1"

$PackageName = "MouseButtonMapper-v$Version-windows-x64"
$Stage = Join-Path $OutputDirectory $PackageName
$Zip = Join-Path $OutputDirectory "$PackageName.zip"
Remove-Item $Stage -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item $Zip -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $Stage -Force | Out-Null

Copy-Item MouseButtonMapper.exe $Stage
Copy-Item scripts\Add_Startup_Shortcut.cmd $Stage
Copy-Item scripts\Remove_Startup_Shortcut.cmd $Stage
Copy-Item scripts\Emergency_Stop_MouseButtonMapper.cmd $Stage
Copy-Item scripts\Remove_Legacy_Install_Keep_Config.cmd $Stage
Copy-Item docs\PORTABLE_README.txt (Join-Path $Stage "README.txt")

$Hash = (Get-FileHash (Join-Path $Stage "MouseButtonMapper.exe") -Algorithm SHA256).Hash.ToLowerInvariant()
"$Hash  MouseButtonMapper.exe" | Set-Content -Encoding ascii (Join-Path $Stage "SHA256SUMS.txt")
Compress-Archive -Path "$Stage\*" -DestinationPath $Zip -CompressionLevel Optimal
Write-Host "Created $Zip"
