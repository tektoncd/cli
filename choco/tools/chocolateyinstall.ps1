$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.36.0/tkn_0.36.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '580db24f608616b787bdd26a75eea582a183a99ece6d2c61489971f9c3d67ca9'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
