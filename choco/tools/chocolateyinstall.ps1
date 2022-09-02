$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.26.0/tkn_0.26.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'a0b42c1300175ce379dadb822ca3633c34048eae0064533d7718746c79f11a65'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
