param(
    [string]$VmName = "MeuNAS-Build",
    [Parameter(Mandatory = $true)]
    [string]$IsoPath,
    [string]$VmRoot = "C:\HyperV",
    [int]$MemoryGB = 8,
    [int]$CpuCount = 4,
    [int]$DiskGB = 100,
    [string]$SwitchName = "Default Switch"
)

$ErrorActionPreference = "Stop"

if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    throw "Execute este script em um PowerShell como administrador."
}

if (-not (Test-Path -LiteralPath $IsoPath)) {
    throw "ISO nao encontrada: $IsoPath"
}

$hyperv = Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V
if ($hyperv.State -ne "Enabled") {
    throw "Hyper-V nao esta habilitado. Execute: Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All"
}

$vmPath = Join-Path $VmRoot $VmName
$vhdPath = Join-Path $vmPath "$VmName.vhdx"

if (Get-VM -Name $VmName -ErrorAction SilentlyContinue) {
    throw "Ja existe uma VM chamada $VmName."
}

New-Item -ItemType Directory -Force -Path $vmPath | Out-Null

New-VM `
    -Name $VmName `
    -Generation 2 `
    -MemoryStartupBytes (${MemoryGB}GB) `
    -Path $vmPath `
    -NewVHDPath $vhdPath `
    -NewVHDSizeBytes (${DiskGB}GB) `
    -SwitchName $SwitchName | Out-Null

Set-VMProcessor -VMName $VmName -Count $CpuCount
Set-VMMemory -VMName $VmName -DynamicMemoryEnabled $true -MinimumBytes 4GB -StartupBytes (${MemoryGB}GB) -MaximumBytes (${MemoryGB}GB)

$dvd = Add-VMDvdDrive -VMName $VmName -Path $IsoPath -PassThru
Set-VMFirmware -VMName $VmName -FirstBootDevice $dvd

# Ubuntu Server instala bem com Secure Boot desativado em VM Gen 2.
Set-VMFirmware -VMName $VmName -EnableSecureBoot Off

Enable-VMIntegrationService -VMName $VmName -Name "Guest Service Interface" -ErrorAction SilentlyContinue

Write-Host "VM criada: $VmName"
Write-Host "ISO: $IsoPath"
Write-Host "Disco: $vhdPath"
Write-Host "Inicie com: Start-VM -Name `"$VmName`""
