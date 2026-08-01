[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$FilePath,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[0-9A-Fa-f]{40}$')]
    [string]$CertificateThumbprint,

    [Parameter(Mandatory = $true)]
    [ValidateNotNull()]
    [uri]$TimestampUrl,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$SigningLogPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

function Resolve-SignTool {
    $command = Get-Command 'signtool.exe' -ErrorAction SilentlyContinue
    if ($null -ne $command) {
        return $command.Source
    }

    $kitsRoot = Join-Path ${env:ProgramFiles(x86)} 'Windows Kits\10\bin'
    if (Test-Path -LiteralPath $kitsRoot -PathType Container) {
        $candidates = @(
            Get-ChildItem -LiteralPath $kitsRoot -Filter 'signtool.exe' -File -Recurse |
                Where-Object { $_.FullName -match '\\x64\\signtool\.exe$' } |
                Sort-Object -Property FullName -Descending
        )
        if ($candidates.Count -gt 0) {
            return $candidates[0].FullName
        }
    }

    throw 'signtool.exe was not found in PATH or the Windows SDK.'
}

if ($TimestampUrl.Scheme -ne 'https') {
    throw 'The Authenticode timestamp URL must use HTTPS.'
}
if (-not (Test-Path -LiteralPath $FilePath -PathType Leaf)) {
    throw "Authenticode signing target does not exist: $FilePath"
}
$resolvedFile = (Resolve-Path -LiteralPath $FilePath).Path
$extension = [IO.Path]::GetExtension($resolvedFile).ToLowerInvariant()
if ($extension -notin @('.exe', '.dll')) {
    throw 'The Tauri signing command only accepts Windows executable files.'
}

$normalizedThumbprint = $CertificateThumbprint.ToUpperInvariant()
$certificates = @(
    Get-ChildItem -Path Cert:\CurrentUser\My -CodeSigningCert |
        Where-Object {
            $_.HasPrivateKey -and $_.Thumbprint -ceq $normalizedThumbprint
        }
)
if ($certificates.Count -ne 1) {
    throw 'The requested code-signing certificate is unavailable or ambiguous.'
}
$certificate = $certificates[0]
$now = Get-Date
if ($certificate.NotBefore -gt $now -or $certificate.NotAfter -le $now) {
    throw 'The Windows code-signing certificate is not currently valid.'
}

$existingSignature = Get-AuthenticodeSignature -LiteralPath $resolvedFile
if ($existingSignature.Status -ne [System.Management.Automation.SignatureStatus]::NotSigned) {
    throw "Refusing to replace an existing signature with status $($existingSignature.Status)."
}

$signTool = Resolve-SignTool
for ($attempt = 1; $attempt -le 3; $attempt++) {
    & $signTool sign `
        /sha1 $normalizedThumbprint `
        /s My `
        /fd SHA256 `
        /tr $TimestampUrl.AbsoluteUri `
        /td SHA256 `
        /d 'Luoxue Quota Viewer' `
        /v `
        $resolvedFile
    if ($LASTEXITCODE -eq 0) {
        break
    }
    if ($attempt -eq 3) {
        throw "signtool sign failed with exit code $LASTEXITCODE."
    }
    Start-Sleep -Seconds ([Math]::Pow(2, $attempt))
}

$verifier = Join-Path $PSScriptRoot 'verify-windows-authenticode.ps1'
if (-not (Test-Path -LiteralPath $verifier -PathType Leaf)) {
    throw 'The Authenticode verifier is missing.'
}
& $verifier `
    -FilePath $resolvedFile `
    -CertificateThumbprint $normalizedThumbprint

$logDirectory = Split-Path -Parent $SigningLogPath
if (-not (Test-Path -LiteralPath $logDirectory -PathType Container)) {
    throw 'The signing log directory does not exist.'
}
$logEntry = [ordered]@{
    file_name = [IO.Path]::GetFileName($resolvedFile)
    file_path = $resolvedFile
    signer_thumbprint = $normalizedThumbprint
}
$logLine = ($logEntry | ConvertTo-Json -Compress) + [Environment]::NewLine
[IO.File]::AppendAllText(
    $SigningLogPath,
    $logLine,
    [Text.UTF8Encoding]::new($false)
)

Write-Host "Signed and verified Tauri artifact: $resolvedFile"
