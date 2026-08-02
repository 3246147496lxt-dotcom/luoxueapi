[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$QuotaViewerDirectory,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$ReleaseVersion
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$semVerPattern = (
    '^(?<major>0|[1-9][0-9]*)\.(?<minor>0|[1-9][0-9]*)\.' +
    '(?<patch>0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|' +
    '[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|' +
    '[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?$'
)

function Assert-WindowsReleaseVersion {
    param([string]$Version)

    $match = [regex]::Match($Version, $semVerPattern)
    if (-not $match.Success) {
        throw 'The Windows release version must be SemVer without build metadata.'
    }

    foreach ($componentName in @('major', 'minor', 'patch')) {
        [uint64]$component = 0
        if (
            -not [uint64]::TryParse(
                $match.Groups[$componentName].Value,
                [ref]$component
            ) -or
            $component -gt [uint16]::MaxValue
        ) {
            throw "The Windows release $componentName component must fit in 16 bits."
        }
    }
}

function Get-JsonVersion {
    param(
        [object]$JsonObject,
        [string]$Label
    )

    $property = $JsonObject.PSObject.Properties['version']
    if (
        $null -eq $property -or
        $property.Value -isnot [string] -or
        [string]::IsNullOrWhiteSpace($property.Value)
    ) {
        throw "$Label must contain a non-empty string version."
    }
    return $property.Value
}

function Get-SingleVersionFromTomlBlock {
    param(
        [string]$Block,
        [string]$Label
    )

    $matches = [regex]::Matches(
        $Block,
        '(?m)^version[ \t]*=[ \t]*"(?<value>[^"\r\n]+)"[ \t]*(?=\r?$)'
    )
    if ($matches.Count -ne 1) {
        throw "$Label must contain exactly one version field."
    }
    return $matches[0].Groups['value'].Value
}

function Set-SingleVersionInTomlBlock {
    param(
        [string]$Block,
        [string]$Version,
        [string]$Label
    )

    $pattern = [regex]::new(
        '(?m)^(?<prefix>version[ \t]*=[ \t]*")[^"\r\n]+' +
        '(?<suffix>"[ \t]*)(?=\r?$)'
    )
    $matches = $pattern.Matches($Block)
    if ($matches.Count -ne 1) {
        throw "$Label must contain exactly one version field."
    }

    $match = $matches[0]
    $replacement = (
        $match.Groups['prefix'].Value +
        $Version +
        $match.Groups['suffix'].Value
    )
    return $Block.Remove($match.Index, $match.Length).Insert(
        $match.Index,
        $replacement
    )
}

function Replace-MatchedBlock {
    param(
        [string]$Text,
        [System.Text.RegularExpressions.Match]$Match,
        [string]$Replacement
    )

    return $Text.Remove($Match.Index, $Match.Length).Insert(
        $Match.Index,
        $Replacement
    )
}

Assert-WindowsReleaseVersion -Version $ReleaseVersion

$quotaViewerRoot = (Resolve-Path -LiteralPath $QuotaViewerDirectory).Path
$packageJsonPath = Join-Path $quotaViewerRoot 'package.json'
$tauriSourceRoot = Join-Path $quotaViewerRoot 'src-tauri'
$tauriConfigPath = Join-Path $tauriSourceRoot 'tauri.conf.json'
$cargoManifestPath = Join-Path $tauriSourceRoot 'Cargo.toml'
$cargoLockPath = Join-Path $tauriSourceRoot 'Cargo.lock'

foreach ($requiredPath in @(
    $packageJsonPath,
    $tauriConfigPath,
    $cargoManifestPath,
    $cargoLockPath
)) {
    if (-not (Test-Path -LiteralPath $requiredPath -PathType Leaf)) {
        throw "Required version source is missing: $requiredPath"
    }
}

$packageJson = Get-Content -LiteralPath $packageJsonPath -Raw | ConvertFrom-Json
$tauriConfig = Get-Content -LiteralPath $tauriConfigPath -Raw | ConvertFrom-Json
$cargoManifestText = Get-Content -LiteralPath $cargoManifestPath -Raw
$cargoLockText = Get-Content -LiteralPath $cargoLockPath -Raw

$cargoPackageSections = [regex]::Matches(
    $cargoManifestText,
    '(?ms)^\[package\][ \t]*\r?\n.*?(?=^\[|\z)'
)
if ($cargoPackageSections.Count -ne 1) {
    throw 'Cargo.toml must contain exactly one [package] section.'
}
$cargoPackageSection = $cargoPackageSections[0]

$cargoLockPackages = [regex]::Matches(
    $cargoLockText,
    '(?ms)^\[\[package\]\][ \t]*\r?\n.*?(?=^\[\[package\]\]|\z)'
)
$quotaViewerLockPackages = @(
    $cargoLockPackages | Where-Object {
        $_.Value -match (
            '(?m)^name[ \t]*=[ \t]*"luoxue-quota-viewer"[ \t]*(?=\r?$)'
        )
    }
)
if ($quotaViewerLockPackages.Count -ne 1) {
    throw 'Cargo.lock must contain exactly one luoxue-quota-viewer package.'
}
$quotaViewerLockPackage = $quotaViewerLockPackages[0]

$sourceVersions = @(
    Get-JsonVersion -JsonObject $packageJson -Label 'package.json'
    Get-JsonVersion -JsonObject $tauriConfig -Label 'tauri.conf.json'
    Get-SingleVersionFromTomlBlock `
        -Block $cargoPackageSection.Value `
        -Label 'Cargo.toml [package]'
    Get-SingleVersionFromTomlBlock `
        -Block $quotaViewerLockPackage.Value `
        -Label 'Cargo.lock application package'
)
$distinctSourceVersions = @($sourceVersions | Sort-Object -Unique)
if ($distinctSourceVersions.Count -ne 1) {
    throw "Quota viewer version sources have drifted: $($sourceVersions -join ', ')."
}

$packageJson.version = $ReleaseVersion
$tauriConfig.version = $ReleaseVersion
$newCargoPackageSection = Set-SingleVersionInTomlBlock `
    -Block $cargoPackageSection.Value `
    -Version $ReleaseVersion `
    -Label 'Cargo.toml [package]'
$newCargoLockPackage = Set-SingleVersionInTomlBlock `
    -Block $quotaViewerLockPackage.Value `
    -Version $ReleaseVersion `
    -Label 'Cargo.lock application package'

$newCargoManifestText = Replace-MatchedBlock `
    -Text $cargoManifestText `
    -Match $cargoPackageSection `
    -Replacement $newCargoPackageSection
$newCargoLockText = Replace-MatchedBlock `
    -Text $cargoLockText `
    -Match $quotaViewerLockPackage `
    -Replacement $newCargoLockPackage

$utf8NoBom = [Text.UTF8Encoding]::new($false)
$packageJsonText = (
    $packageJson | ConvertTo-Json -Depth 100
) + [Environment]::NewLine
$tauriConfigText = (
    $tauriConfig | ConvertTo-Json -Depth 100
) + [Environment]::NewLine
[IO.File]::WriteAllText($packageJsonPath, $packageJsonText, $utf8NoBom)
[IO.File]::WriteAllText($tauriConfigPath, $tauriConfigText, $utf8NoBom)
[IO.File]::WriteAllText($cargoManifestPath, $newCargoManifestText, $utf8NoBom)
[IO.File]::WriteAllText($cargoLockPath, $newCargoLockText, $utf8NoBom)

$cargoCommand = Get-Command 'cargo' -ErrorAction SilentlyContinue
if ($null -eq $cargoCommand) {
    throw 'cargo is required to validate the stamped release manifest and lock file.'
}
$metadataLines = @(
    & $cargoCommand.Source metadata `
        --format-version 1 `
        --locked `
        --no-deps `
        --manifest-path $cargoManifestPath
)
if ($LASTEXITCODE -ne 0) {
    throw "cargo metadata --locked failed with exit code $LASTEXITCODE."
}
$metadata = ($metadataLines -join [Environment]::NewLine) | ConvertFrom-Json
$releasePackages = @(
    $metadata.packages | Where-Object { $_.name -eq 'luoxue-quota-viewer' }
)
if (
    $releasePackages.Count -ne 1 -or
    $releasePackages[0].version -cne $ReleaseVersion
) {
    throw 'Cargo metadata does not contain the stamped quota viewer version.'
}

$writtenPackageJson = Get-Content -LiteralPath $packageJsonPath -Raw |
    ConvertFrom-Json
$writtenTauriConfig = Get-Content -LiteralPath $tauriConfigPath -Raw |
    ConvertFrom-Json
if (
    (Get-JsonVersion -JsonObject $writtenPackageJson -Label 'package.json') -cne
        $ReleaseVersion -or
    (Get-JsonVersion -JsonObject $writtenTauriConfig -Label 'tauri.conf.json') -cne
        $ReleaseVersion
) {
    throw 'The JSON release version sources were not stamped correctly.'
}

Write-Host "Prepared Windows Tauri sources for release $ReleaseVersion."
