$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.27.0/tkn_0.27.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '096C8E18315903A327D54FD2F79D4A84F46916CFDF5F4A53AF3638891D5B9097'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
