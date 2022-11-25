$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.28.0/tkn_0.28.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'ffecfde520f57c9cf1ee9ceb5287469a71a12087fc0b06c31d290e94e761d1dc'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
