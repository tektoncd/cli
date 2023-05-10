$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.31.0/tkn_0.31.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '6d22ad060905e4b3ab6316fe1463b70a33dc3b94b264a332c27dc0a13a4c394b'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
