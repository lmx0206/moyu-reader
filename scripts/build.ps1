# build.ps1 — one-click build of moyu-reader.exe with the version baked in.
#
# The version is derived automatically from git (the latest tag), so you never
# type it by hand:
#   - on a tag (e.g. v0.6.2)        -> 0.6.2          -> moyu-reader_v0.6.2.exe
#   - a few commits past a tag       -> 0.6.2-3-gabc1 -> moyu-reader_v0.6.2-3-gabc1.exe
#   - uncommitted changes            -> ...-dirty
#   - no tags at all                 -> a short commit hash, or "dev"
#
# Usage (from the repo root):  ./scripts/build.ps1
$ErrorActionPreference = "Stop"

# Build from the repo root regardless of where the script is invoked.
$root = Resolve-Path (Join-Path $PSScriptRoot "..")
Push-Location $root
try {
    $desc = (& git describe --tags --always --dirty 2>$null)
    if (-not $desc) { $desc = "dev" }
    $ver = $desc -replace '^v', ''
    $out = "moyu-reader_v$ver.exe"

    Write-Host "Building $out (version $ver) ..."
    go build -ldflags "-s -w -X moyureader/internal/version.Version=$ver" -o $out ./cmd/reader
    Write-Host "Done: $out"
} finally {
    Pop-Location
}
