[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$FilePath,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[0-9A-Fa-f]{40}$')]
    [string]$CertificateThumbprint
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

if (-not (Test-Path -LiteralPath $FilePath -PathType Leaf)) {
    throw "Authenticode target does not exist: $FilePath"
}
$resolvedFile = (Resolve-Path -LiteralPath $FilePath).Path
$normalizedThumbprint = $CertificateThumbprint.ToUpperInvariant()
$signTool = Resolve-SignTool

& $signTool verify /pa /all /v $resolvedFile
if ($LASTEXITCODE -ne 0) {
    throw "signtool verify failed with exit code $LASTEXITCODE."
}

$signature = Get-AuthenticodeSignature -LiteralPath $resolvedFile
if ($signature.Status -ne [System.Management.Automation.SignatureStatus]::Valid) {
    throw "Authenticode verification returned status $($signature.Status)."
}
if ($null -eq $signature.SignerCertificate) {
    throw 'Authenticode verification did not return a signer certificate.'
}
if ($signature.SignerCertificate.Thumbprint -cne $normalizedThumbprint) {
    throw 'The Authenticode signer does not match the release certificate.'
}
if ($null -eq $signature.TimeStamperCertificate) {
    throw 'The Authenticode signature does not contain a trusted timestamp.'
}

Write-Host "Verified Authenticode signature: $resolvedFile"
