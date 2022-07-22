$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.25.0/tkn_0.25.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '29b98eceacd86f67aa10436907075af0fd18b3f370746b9af18e62d734b12786'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
