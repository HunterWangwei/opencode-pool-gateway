param(
    [string]$Version = (Get-Content "$PSScriptRoot\..\VERSION" -Raw).Trim()
)

$ErrorActionPreference = "Stop"
$root = Resolve-Path "$PSScriptRoot\.."
$dist = Join-Path $root "dist"
$commit = try { (git -C $root rev-parse --short HEAD 2>$null).Trim() } catch { "unknown" }
if (-not $commit) { $commit = "unknown" }
$date = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$ldflags = "-s -w -X main.version=$Version -X main.commit=$commit -X main.buildDate=$date"

New-Item -ItemType Directory -Force $dist | Out-Null

$targets = @(
    @{ GOOS = "windows"; GOARCH = "amd64"; Name = "opencode-pool-gateway-$Version-windows-amd64.exe" },
    @{ GOOS = "linux"; GOARCH = "amd64"; Name = "opencode-pool-gateway-$Version-linux-amd64" },
    @{ GOOS = "linux"; GOARCH = "arm64"; Name = "opencode-pool-gateway-$Version-linux-arm64" }
)

foreach ($target in $targets) {
    $env:CGO_ENABLED = "0"
    $env:GOOS = $target.GOOS
    $env:GOARCH = $target.GOARCH
	go build -trimpath -ldflags $ldflags -o (Join-Path $dist $target.Name) $root
	if ($LASTEXITCODE -ne 0) { throw "go build failed for $($target.GOOS)/$($target.GOARCH)" }
}

Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
$artifacts = $targets | ForEach-Object { Join-Path $dist $_.Name }
Get-FileHash $artifacts -Algorithm SHA256 |
    ForEach-Object { "$($_.Hash.ToLower())  $([IO.Path]::GetFileName($_.Path))" } |
    Set-Content -Encoding ascii (Join-Path $dist "SHA256SUMS.txt")

Write-Host "Built OpenCode Pool Gateway $Version in $dist"
