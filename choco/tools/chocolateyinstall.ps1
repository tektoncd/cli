$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.22.0/tkn_0.22.0_Windows_x86_64.zip' 

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '21C1C0E0734EF44698B9D3A1540AE73FE6414CCF3255E5C5F21555FE34700A8B'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
