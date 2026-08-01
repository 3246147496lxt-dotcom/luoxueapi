[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$QuotaViewerDirectory,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$UnsignedExecutablePath,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$SignedOutputDirectory,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$ReleaseVersion,

    [Parameter(Mandatory = $false)]
    [ValidateNotNull()]
    [uri]$TimestampUrl = 'https://timestamp.digicert.com'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

function Invoke-AuthenticodeVerification {
    param(
        [string]$Path,
        [string]$Thumbprint
    )

    & $authenticodeVerifier `
        -FilePath $Path `
        -CertificateThumbprint $Thumbprint
}

function Wait-ForPathRemoval {
    param(
        [string]$Path,
        [int]$Attempts = 20
    )

    for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
        if (-not (Test-Path -LiteralPath $Path)) {
            return $true
        }
        Start-Sleep -Seconds 1
    }
    return -not (Test-Path -LiteralPath $Path)
}

if ($TimestampUrl.Scheme -ne 'https') {
    throw 'The Authenticode timestamp URL must use HTTPS.'
}
if (-not (Test-Path -LiteralPath $UnsignedExecutablePath -PathType Leaf)) {
    throw 'The unsigned Windows application executable does not exist.'
}
$unsignedExecutable = (Resolve-Path -LiteralPath $UnsignedExecutablePath).Path
if ([IO.Path]::GetFileName($unsignedExecutable) -cne 'luoxue-quota-viewer.exe') {
    throw 'The release input must be the luoxue-quota-viewer.exe application.'
}

$quotaViewerRoot = (Resolve-Path -LiteralPath $QuotaViewerDirectory).Path
$tauriSourceRoot = Join-Path $quotaViewerRoot 'src-tauri'
$targetReleaseDirectory = Join-Path `
    $tauriSourceRoot `
    'target\x86_64-pc-windows-msvc\release'
$targetExecutable = Join-Path $targetReleaseDirectory 'luoxue-quota-viewer.exe'
$nsisOutputDirectory = Join-Path $targetReleaseDirectory 'bundle\nsis'

$versionVerifier = Join-Path $PSScriptRoot 'verify-windows-release-version.ps1'
$authenticodeVerifier = Join-Path `
    $PSScriptRoot `
    'verify-windows-authenticode.ps1'
$tauriSigner = Join-Path $PSScriptRoot 'sign-windows-file.ps1'
foreach ($requiredScript in @(
    $versionVerifier,
    $authenticodeVerifier,
    $tauriSigner
)) {
    if (-not (Test-Path -LiteralPath $requiredScript -PathType Leaf)) {
        throw "Required Windows release script is missing: $requiredScript"
    }
}

& $versionVerifier `
    -FilePath $unsignedExecutable `
    -ReleaseVersion $ReleaseVersion
$unsignedSignature = Get-AuthenticodeSignature -LiteralPath $unsignedExecutable
if ($unsignedSignature.Status -ne [System.Management.Automation.SignatureStatus]::NotSigned) {
    throw 'The release input application must be unsigned before Tauri packaging.'
}

$certificateBase64 = [Environment]::GetEnvironmentVariable(
    'WINDOWS_SIGNING_CERTIFICATE_PFX_BASE64',
    [EnvironmentVariableTarget]::Process
)
$certificatePasswordText = [Environment]::GetEnvironmentVariable(
    'WINDOWS_SIGNING_CERTIFICATE_PASSWORD',
    [EnvironmentVariableTarget]::Process
)
if ([string]::IsNullOrWhiteSpace($certificateBase64)) {
    throw 'WINDOWS_SIGNING_CERTIFICATE_PFX_BASE64 is required for a full release.'
}
if ([string]::IsNullOrEmpty($certificatePasswordText)) {
    throw 'WINDOWS_SIGNING_CERTIFICATE_PASSWORD is required for a full release.'
}
Remove-Item Env:WINDOWS_SIGNING_CERTIFICATE_PFX_BASE64 -ErrorAction SilentlyContinue
Remove-Item Env:WINDOWS_SIGNING_CERTIFICATE_PASSWORD -ErrorAction SilentlyContinue

if ([string]::IsNullOrWhiteSpace($env:RUNNER_TEMP)) {
    throw 'RUNNER_TEMP is required for isolated Windows signing state.'
}
$runnerTemp = (Resolve-Path -LiteralPath $env:RUNNER_TEMP).Path
$runIdentifier = [guid]::NewGuid().ToString('N')
$pfxPath = Join-Path $runnerTemp "luoxue-signing-$runIdentifier.pfx"
$signingConfigPath = Join-Path $runnerTemp "luoxue-signing-$runIdentifier.json"
$signingLogPath = Join-Path $runnerTemp "luoxue-signing-$runIdentifier.jsonl"
$installationDirectory = Join-Path $runnerTemp "luoxue-install-$runIdentifier"

$runnerTempPrefix = [IO.Path]::GetFullPath($runnerTemp).TrimEnd(
    [IO.Path]::DirectorySeparatorChar,
    [IO.Path]::AltDirectorySeparatorChar
) + [IO.Path]::DirectorySeparatorChar
$installationFullPath = [IO.Path]::GetFullPath($installationDirectory)
if (-not $installationFullPath.StartsWith(
    $runnerTempPrefix,
    [StringComparison]::OrdinalIgnoreCase
)) {
    throw 'The release verification install path escaped RUNNER_TEMP.'
}
if ($installationFullPath -match '\s') {
    throw 'RUNNER_TEMP must not contain whitespace for silent NSIS verification.'
}

$existingThumbprints = [System.Collections.Generic.HashSet[string]]::new(
    [System.StringComparer]::OrdinalIgnoreCase
)
Get-ChildItem -Path Cert:\CurrentUser\My | ForEach-Object {
    [void]$existingThumbprints.Add($_.Thumbprint)
}

$certificateBytes = $null
$securePassword = $null
$newCertificateThumbprints = @()
$cleanupErrors = [System.Collections.Generic.List[string]]::new()
$uninstallCompleted = $false
$releaseSucceeded = $false

try {
    try {
        $certificateBytes = [Convert]::FromBase64String($certificateBase64)
    }
    catch {
        throw 'The Windows signing certificate secret is not valid base64.'
    }
    if ($certificateBytes.Length -eq 0) {
        throw 'The Windows signing certificate secret decoded to an empty file.'
    }
    [IO.File]::WriteAllBytes($pfxPath, $certificateBytes)

    $securePassword = ConvertTo-SecureString `
        -String $certificatePasswordText `
        -AsPlainText `
        -Force
    $importedCertificates = @(
        Import-PfxCertificate `
            -FilePath $pfxPath `
            -CertStoreLocation 'Cert:\CurrentUser\My' `
            -Password $securePassword
    )
    if ($importedCertificates.Count -eq 0) {
        throw 'The PFX did not import any certificates.'
    }

    $importedThumbprints = [System.Collections.Generic.HashSet[string]]::new(
        [System.StringComparer]::OrdinalIgnoreCase
    )
    foreach ($certificate in $importedCertificates) {
        [void]$importedThumbprints.Add($certificate.Thumbprint)
        if (-not $existingThumbprints.Contains($certificate.Thumbprint)) {
            $newCertificateThumbprints += $certificate.Thumbprint
        }
    }

    $signingCertificates = @(
        Get-ChildItem -Path Cert:\CurrentUser\My -CodeSigningCert |
            Where-Object {
                $_.HasPrivateKey -and $importedThumbprints.Contains($_.Thumbprint)
            }
    )
    if ($signingCertificates.Count -ne 1) {
        throw 'The PFX must contain exactly one certificate with a code-signing private key.'
    }
    $signingCertificate = $signingCertificates[0]
    $certificateThumbprint = $signingCertificate.Thumbprint.ToUpperInvariant()
    $now = Get-Date
    if ($signingCertificate.NotBefore -gt $now -or $signingCertificate.NotAfter -le $now) {
        throw 'The Windows code-signing certificate is not currently valid.'
    }

    Remove-Item -LiteralPath $pfxPath -Force
    [Array]::Clear($certificateBytes, 0, $certificateBytes.Length)
    $certificateBytes = $null
    $certificateBase64 = $null
    $certificatePasswordText = $null
    $securePassword = $null

    if (-not (Test-Path -LiteralPath $targetReleaseDirectory)) {
        [void](New-Item -Path $targetReleaseDirectory -ItemType Directory)
    }
    if (Test-Path -LiteralPath $targetExecutable) {
        throw 'Refusing to overwrite an existing Tauri release executable.'
    }
    Copy-Item -LiteralPath $unsignedExecutable -Destination $targetExecutable
    & $versionVerifier `
        -FilePath $targetExecutable `
        -ReleaseVersion $ReleaseVersion

    [IO.File]::WriteAllText(
        $signingLogPath,
        '',
        [Text.UTF8Encoding]::new($false)
    )
    $pwshCommand = Get-Command 'pwsh' -ErrorAction SilentlyContinue
    if ($null -eq $pwshCommand) {
        throw 'pwsh is required for the Tauri custom signing command.'
    }
    $signingConfig = [ordered]@{
        bundle = [ordered]@{
            windows = [ordered]@{
                certificateThumbprint = $certificateThumbprint
                digestAlgorithm = 'sha256'
                timestampUrl = $TimestampUrl.AbsoluteUri
                tsp = $true
                signCommand = [ordered]@{
                    cmd = $pwshCommand.Source
                    args = @(
                        '-NoLogo',
                        '-NoProfile',
                        '-File',
                        $tauriSigner,
                        '-FilePath',
                        '%1',
                        '-CertificateThumbprint',
                        $certificateThumbprint,
                        '-TimestampUrl',
                        $TimestampUrl.AbsoluteUri,
                        '-SigningLogPath',
                        $signingLogPath
                    )
                }
            }
        }
    }
    $signingConfigJson = (
        $signingConfig | ConvertTo-Json -Depth 20
    ) + [Environment]::NewLine
    [IO.File]::WriteAllText(
        $signingConfigPath,
        $signingConfigJson,
        [Text.UTF8Encoding]::new($false)
    )

    Push-Location $quotaViewerRoot
    try {
        & pnpm run tauri:bundle:windows --config $signingConfigPath
        if ($LASTEXITCODE -ne 0) {
            throw "Tauri signed NSIS bundle failed with exit code $LASTEXITCODE."
        }
    }
    finally {
        Pop-Location
    }

    $signingEntries = @(
        Get-Content -LiteralPath $signingLogPath |
            Where-Object { -not [string]::IsNullOrWhiteSpace($_) } |
            ForEach-Object { $_ | ConvertFrom-Json }
    )
    $mainIndices = [System.Collections.Generic.List[int]]::new()
    $uninstallerIndices = [System.Collections.Generic.List[int]]::new()
    $setupIndices = [System.Collections.Generic.List[int]]::new()
    for ($index = 0; $index -lt $signingEntries.Count; $index++) {
        $entry = $signingEntries[$index]
        if ($entry.signer_thumbprint -cne $certificateThumbprint) {
            throw 'The Tauri signing log contains an unexpected certificate.'
        }
        if ($entry.file_name -ceq 'luoxue-quota-viewer.exe') {
            $mainIndices.Add($index)
        }
        elseif ($entry.file_name -ceq 'uninstall.exe') {
            $uninstallerIndices.Add($index)
        }
        elseif ($entry.file_name -like '*-setup.exe') {
            $setupIndices.Add($index)
        }
    }
    if (
        $mainIndices.Count -ne 1 -or
        $uninstallerIndices.Count -ne 1 -or
        $setupIndices.Count -ne 1 -or
        $mainIndices[0] -ge $uninstallerIndices[0] -or
        $uninstallerIndices[0] -ge $setupIndices[0]
    ) {
        throw 'Signing log did not prove main EXE -> NSIS !uninstfinalize -> setup order.'
    }

    $generatedInstallers = @(
        Get-ChildItem `
            -LiteralPath $nsisOutputDirectory `
            -Filter '*-setup.exe' `
            -File
    )
    if ($generatedInstallers.Count -ne 1) {
        throw 'Expected exactly one Tauri-signed NSIS setup executable.'
    }
    $generatedInstaller = $generatedInstallers[0].FullName
    & $versionVerifier `
        -FilePath $generatedInstaller `
        -ReleaseVersion $ReleaseVersion
    Invoke-AuthenticodeVerification `
        -Path $generatedInstaller `
        -Thumbprint $certificateThumbprint

    $installProcess = Start-Process `
        -FilePath $generatedInstaller `
        -ArgumentList @('/S', "/D=$installationFullPath") `
        -Wait `
        -PassThru
    if ($installProcess.ExitCode -ne 0) {
        throw "Silent NSIS verification install failed with exit code $($installProcess.ExitCode)."
    }
    $installedExecutable = Join-Path $installationFullPath 'luoxue-quota-viewer.exe'
    $installedUninstaller = Join-Path $installationFullPath 'uninstall.exe'
    & $versionVerifier `
        -FilePath $installedExecutable `
        -ReleaseVersion $ReleaseVersion
    Invoke-AuthenticodeVerification `
        -Path $installedExecutable `
        -Thumbprint $certificateThumbprint
    Invoke-AuthenticodeVerification `
        -Path $installedUninstaller `
        -Thumbprint $certificateThumbprint

    $uninstallProcess = Start-Process `
        -FilePath $installedUninstaller `
        -ArgumentList '/S' `
        -Wait `
        -PassThru
    if ($uninstallProcess.ExitCode -ne 0) {
        throw "Silent NSIS verification uninstall failed with exit code $($uninstallProcess.ExitCode)."
    }
    if (-not (Wait-ForPathRemoval -Path $installationFullPath)) {
        throw 'The NSIS verification installation was not removed cleanly.'
    }
    $uninstallCompleted = $true

    if (-not (Test-Path -LiteralPath $SignedOutputDirectory)) {
        [void](New-Item -Path $SignedOutputDirectory -ItemType Directory)
    }
    elseif (-not (Test-Path -LiteralPath $SignedOutputDirectory -PathType Container)) {
        throw 'The signed Windows output path is not a directory.'
    }

    $installerName = 'luoxue-quota-viewer_{0}_windows_x64-setup.exe' -f $ReleaseVersion
    $signedInstaller = Join-Path $SignedOutputDirectory $installerName
    if (Test-Path -LiteralPath $signedInstaller) {
        throw 'Refusing to overwrite an existing signed Windows installer.'
    }
    Copy-Item -LiteralPath $generatedInstaller -Destination $signedInstaller
    & $versionVerifier `
        -FilePath $signedInstaller `
        -ReleaseVersion $ReleaseVersion
    Invoke-AuthenticodeVerification `
        -Path $signedInstaller `
        -Thumbprint $certificateThumbprint

    $digest = (Get-FileHash -LiteralPath $signedInstaller -Algorithm SHA256).Hash.ToLowerInvariant()
    $checksumPath = "$signedInstaller.sha256"
    $checksumLine = '{0}  {1}{2}' -f $digest, $installerName, [Environment]::NewLine
    [IO.File]::WriteAllText(
        $checksumPath,
        $checksumLine,
        [Text.UTF8Encoding]::new($false)
    )

    $releaseSucceeded = $true
    Write-Host "Built, signed, installed, and verified Windows release: $installerName"
}
finally {
    if (
        -not $uninstallCompleted -and
        (Test-Path -LiteralPath $installationFullPath -PathType Container)
    ) {
        $fallbackUninstaller = Join-Path $installationFullPath 'uninstall.exe'
        if (Test-Path -LiteralPath $fallbackUninstaller -PathType Leaf) {
            try {
                $fallbackProcess = Start-Process `
                    -FilePath $fallbackUninstaller `
                    -ArgumentList '/S' `
                    -Wait `
                    -PassThru
                if ($fallbackProcess.ExitCode -ne 0) {
                    $cleanupErrors.Add('verification installation registry state')
                }
                [void](Wait-ForPathRemoval -Path $installationFullPath -Attempts 10)
            }
            catch {
                $cleanupErrors.Add('verification installation registry state')
            }
        }
        if (Test-Path -LiteralPath $installationFullPath) {
            try {
                Remove-Item -LiteralPath $installationFullPath -Recurse -Force
            }
            catch {
                $cleanupErrors.Add('verification installation directory')
            }
        }
    }

    foreach ($temporaryFile in @($pfxPath, $signingConfigPath, $signingLogPath)) {
        if (Test-Path -LiteralPath $temporaryFile) {
            try {
                Remove-Item -LiteralPath $temporaryFile -Force
            }
            catch {
                $cleanupErrors.Add("temporary signing file $temporaryFile")
            }
        }
    }
    foreach ($thumbprint in $newCertificateThumbprints) {
        try {
            $certificatePath = "Cert:\CurrentUser\My\$thumbprint"
            $certificateItem = Get-Item -LiteralPath $certificatePath
            if ($certificateItem.HasPrivateKey) {
                Remove-Item -LiteralPath $certificatePath -DeleteKey -Force
            }
            else {
                Remove-Item -LiteralPath $certificatePath -Force
            }
            if (Test-Path -LiteralPath $certificatePath) {
                throw 'The imported signing certificate still exists after removal.'
            }
        }
        catch {
            $cleanupErrors.Add("certificate and private key $thumbprint")
        }
    }
    if ($null -ne $certificateBytes) {
        [Array]::Clear($certificateBytes, 0, $certificateBytes.Length)
    }
    $certificateBase64 = $null
    $certificatePasswordText = $null
    $securePassword = $null

    if ($cleanupErrors.Count -gt 0) {
        throw "Failed to clean Windows signing state: $($cleanupErrors -join ', ')."
    }
    if (-not $releaseSucceeded) {
        Write-Host 'Windows release signing failed closed.'
    }
}
