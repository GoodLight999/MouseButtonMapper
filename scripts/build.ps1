$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

gofmt -w *.go
go test ./...
go test -race ./...
go vet -unsafeptr=false ./...
go build -trimpath -ldflags="-s -w -H=windowsgui" -o MouseButtonMapper.exe .

$process = Start-Process -FilePath ".\MouseButtonMapper.exe" -ArgumentList "--self-test" -Wait -PassThru
if ($process.ExitCode -ne 0) {
    throw "MouseButtonMapper.exe --self-test failed with exit code $($process.ExitCode)"
}
Get-FileHash .\MouseButtonMapper.exe -Algorithm SHA256
