$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.39.0/tkn_0.39.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'e3166678b81a9e94f9a8ac7a88331fca8a67b5e50dbb4b1a33529cc283225692'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
