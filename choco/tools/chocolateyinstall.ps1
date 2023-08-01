$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.31.2/tkn_0.31.2_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'a7308bbc4edf2d9999622bc69c310cfb50b8255e1ec85dd24a81d2266d4c0311'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
