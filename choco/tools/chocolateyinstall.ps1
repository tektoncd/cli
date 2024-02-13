$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.35.1/tkn_0.35.1_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'c1103315a7bde403a8106002e2a915711e64832efb26e4776063a0393e472306'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
