$ErrorActionPreference = "Stop"

$Repo = if ($env:JIT_CLI_REPO) { $env:JIT_CLI_REPO } else { "wanyun/JitCli" }
$BinName = if ($env:JIT_CLI_BIN_NAME) { $env:JIT_CLI_BIN_NAME } else { "jit" }
$InstallDir = if ($env:JIT_CLI_INSTALL_DIR) { $env:JIT_CLI_INSTALL_DIR } else { Join-Path $HOME ".local\bin" }
$Version = if ($env:JIT_CLI_VERSION) { $env:JIT_CLI_VERSION } else { "latest" }

function Fail([string]$Message) {
    Write-Host $Message -ForegroundColor Red
    exit 1
}

function Resolve-Version {
    if ($Version -ne "latest") { return $Version }
    try {
        $response = Invoke-WebRequest -Uri "https://github.com/$Repo/releases/latest" -MaximumRedirection 0 -ErrorAction SilentlyContinue -UseBasicParsing
        if ($response.Headers.Location) {
            return ($response.Headers.Location.ToString().Split("/") | Select-Object -Last 1)
        }
    } catch {
        if ($_.Exception.Response -and $_.Exception.Response.Headers.Location) {
            return ($_.Exception.Response.Headers.Location.ToString().Split("/") | Select-Object -Last 1)
        }
    }
    Fail "failed to resolve latest version, set JIT_CLI_VERSION explicitly"
}

function Get-Arch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    if ($arch -eq "AMD64") { return "amd64" }
    if ($arch -eq "ARM64") { return "arm64" }
    Fail "unsupported architecture: $arch"
}

function Main {
    $resolvedVersion = Resolve-Version
    $arch = Get-Arch
    $archive = "$BinName-windows-$arch.zip"
    $url = "https://github.com/$Repo/releases/download/$resolvedVersion/$archive"

    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "jit-install-$PID"
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
    try {
        $archivePath = Join-Path $tmpDir $archive
        Write-Host "Downloading $BinName $resolvedVersion (windows/$arch)"
        Invoke-WebRequest -Uri $url -OutFile $archivePath -UseBasicParsing

        $extractDir = Join-Path $tmpDir "extract"
        New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
        Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force

        $binaryPath = Join-Path $extractDir "$BinName.exe"
        if (!(Test-Path $binaryPath)) {
            Fail "binary not found in archive: $archive"
        }

        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Copy-Item -Path $binaryPath -Destination (Join-Path $InstallDir "$BinName.exe") -Force
        Write-Host "Installed to $(Join-Path $InstallDir "$BinName.exe")"
        Write-Host "Ensure '$InstallDir' is in your PATH"
    } finally {
        if (Test-Path $tmpDir) {
            Remove-Item -Path $tmpDir -Recurse -Force
        }
    }
}

Main
