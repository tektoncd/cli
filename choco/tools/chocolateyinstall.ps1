$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.38.0/tkn_0.38.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '756414c0ac4245138e313c9662bc5960251ade6fb06cf58c95be88a8d8dc1c57'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
