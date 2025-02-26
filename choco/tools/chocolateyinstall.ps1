$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.40.0/tkn_0.40.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'ab62363c501afca8a8ce59654240eff7c1d1f4f1ec9d6558501c84b80292577d'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
