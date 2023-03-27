$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.30.0/tkn_0.30.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '040f1969ba6ae151ccc7ed9ca48b10be4f98133149bafae2f99d9c6fdbb53ad2'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
