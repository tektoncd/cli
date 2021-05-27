$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.18.0/tkn_0.18.0_Windows_x86_64.zip' 

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '914a5988b4e2a43dfaa0d6718339422f79238c4d0adf23f13f44825736b878d8'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
