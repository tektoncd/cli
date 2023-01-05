$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.29.0/tkn_0.29.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'b7214731bf08ee56239578b05e8283331ceb3cb5f5f06e5f82901d34c934b52a'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
