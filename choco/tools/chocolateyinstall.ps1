$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.37.0/tkn_0.37.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '29128908f8d60072c96eb8577f9fa344193f3e33b5a0e3b63c146cb454a5f38f'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
