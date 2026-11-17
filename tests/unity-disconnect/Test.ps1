param([string]$ProjectRoot=(Split-Path (Split-Path $PSScriptRoot -Parent) -Parent),[string]$ClientSource,[string]$SessionSource)
$ErrorActionPreference='Stop'
if(-not $ClientSource){
 $candidates=@(Get-ChildItem -LiteralPath (Join-Path $ProjectRoot 'Assets') -Filter MMOWebSocketClient.cs -Recurse)
 if($candidates.Count -ne 1){throw 'Exactly one MMOWebSocketClient.cs is required.'}
 $ClientSource=$candidates[0].FullName
}
if(-not $SessionSource){$SessionSource=Join-Path (Split-Path (Split-Path $ClientSource -Parent) -Parent) 'Core/MMOSession.cs'}
$vswhere=Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
$compiler=& $vswhere -latest -requires Microsoft.Component.MSBuild -find 'MSBuild/Current/Bin/Roslyn/csc.exe' | Select-Object -First 1
if(-not $compiler){throw 'Visual Studio C# compiler was not found.'}
$output=Join-Path ([IO.Path]::GetTempPath()) ('NDUnityDisconnect-'+[Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $output | Out-Null
try {
 & $compiler /nologo /codepage:65001 /target:exe "/out:$output/ClientTests.exe" $ClientSource $SessionSource (Join-Path $PSScriptRoot 'UnityStubs.cs') (Join-Path $PSScriptRoot 'ClientTests.cs')
 if($LASTEXITCODE -ne 0){throw 'Client test compilation failed.'}
 & (Join-Path $output 'ClientTests.exe')
 if($LASTEXITCODE -ne 0){throw 'Client behavior tests failed.'}
}finally{
 $full=[IO.Path]::GetFullPath($output)
 $temp=[IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\')+'\'
 if(-not $full.StartsWith($temp,[StringComparison]::OrdinalIgnoreCase) -or -not (Split-Path $full -Leaf).StartsWith('NDUnityDisconnect-')){throw 'Unexpected test output directory.'}
 Remove-Item -LiteralPath $full -Recurse -Force
}
