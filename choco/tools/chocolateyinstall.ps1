$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.32.0/tkn_0.32.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '708342c75e9b1ddf957b0f5bb4474f10051b998459cfd17b2a08bf18c734c881'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
