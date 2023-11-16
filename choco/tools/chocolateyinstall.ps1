$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.33.0/tkn_0.33.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '85efa3c755c235b381e1591d26911d163b39f165879dcf5a4f6f4d9f18d7ddef'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
