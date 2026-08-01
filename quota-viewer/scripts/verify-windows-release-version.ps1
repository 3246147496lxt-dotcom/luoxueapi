[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$FilePath,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$ReleaseVersion
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$semVerPattern = (
    '^(?<major>0|[1-9][0-9]*)\.(?<minor>0|[1-9][0-9]*)\.' +
    '(?<patch>0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|' +
    '[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|' +
    '[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?$'
)
$versionMatch = [regex]::Match($ReleaseVersion, $semVerPattern)
if (-not $versionMatch.Success) {
    throw 'The expected Windows release version must be SemVer without build metadata.'
}

$coreComponents = [System.Collections.Generic.List[int]]::new()
foreach ($componentName in @('major', 'minor', 'patch')) {
    [uint64]$component = 0
    if (
        -not [uint64]::TryParse(
            $versionMatch.Groups[$componentName].Value,
            [ref]$component
        ) -or
        $component -gt [uint16]::MaxValue
    ) {
        throw "The Windows release $componentName component must fit in 16 bits."
    }
    $coreComponents.Add([int]$component)
}

if (-not (Test-Path -LiteralPath $FilePath -PathType Leaf)) {
    throw "Windows release executable does not exist: $FilePath"
}
$resolvedFile = (Resolve-Path -LiteralPath $FilePath).Path
if ([IO.Path]::GetExtension($resolvedFile) -cne '.exe') {
    throw 'Windows release version verification only accepts .exe files.'
}

$versionInfo = [Diagnostics.FileVersionInfo]::GetVersionInfo($resolvedFile)
$fileVersion = if ($null -eq $versionInfo.FileVersion) {
    ''
} else {
    $versionInfo.FileVersion.Trim()
}
$productVersion = if ($null -eq $versionInfo.ProductVersion) {
    ''
} else {
    $versionInfo.ProductVersion.Trim()
}
if ($fileVersion -cne $ReleaseVersion) {
    throw "FileVersion '$fileVersion' does not match release '$ReleaseVersion'."
}
if ($productVersion -cne $ReleaseVersion) {
    throw "ProductVersion '$productVersion' does not match release '$ReleaseVersion'."
}

$expectedRawVersion = [version]::new(
    $coreComponents[0],
    $coreComponents[1],
    $coreComponents[2],
    0
)
if (
    $null -eq $versionInfo.FileVersionRaw -or
    $versionInfo.FileVersionRaw -ne $expectedRawVersion
) {
    throw "FileVersionRaw does not match $expectedRawVersion."
}
if (
    $null -eq $versionInfo.ProductVersionRaw -or
    $versionInfo.ProductVersionRaw -ne $expectedRawVersion
) {
    throw "ProductVersionRaw does not match $expectedRawVersion."
}

Write-Host "Verified Windows version metadata for ${resolvedFile}: $ReleaseVersion"
