$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.38.1/tkn_0.38.1_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '205bba1cccdb4da3670eff1eae960272959214b39563b0fb42a9fc5bb20c57c7'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
