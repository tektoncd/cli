$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.43.0/tkn_0.43.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'c57dc0f0e97414a7482cb9a420d47eab85410903fabc072925edb09691f0aadc'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
