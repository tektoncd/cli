$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.26.0/tkn_0.26.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'A0B42C1300175CE379DADB822CA3633C34048EAE0064533D7718746C79F11A65'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
