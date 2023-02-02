$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '262c0c60ebddac93438640f7b38f78fc29341199b9e4d99f2c29ae61042588e1'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
