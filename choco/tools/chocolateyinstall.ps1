$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.30.1/tkn_0.30.1_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'bafb35a3ed707a4d7e02422fa4573ee85f5d9710a323feec553fd4f4cc7cdd45'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
