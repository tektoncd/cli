$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.21.0/tkn_0.21.0_Windows_x86_64.zip' 

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'CFA6968E4C4BA4408043C62150886C40BA577FA34EA0C6D977FA8259D2098A26'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
