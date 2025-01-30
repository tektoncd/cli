$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.39.1/tkn_0.39.1_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '2211eb81b83ac33fb7f859b2917d9b0e3f5029d570076b476a96e1b2fd272f0e'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
