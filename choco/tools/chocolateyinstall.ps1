$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.31.1/tkn_0.31.1_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'f9a8c39d0ee6cf03a2a092466b0578c8740a4bddfc1910ce3042868cfac891b1'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
