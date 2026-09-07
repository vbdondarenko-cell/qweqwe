$ErrorActionPreference = 'Stop'

$WrapperDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Target = Join-Path $WrapperDir 'gradle-wrapper.jar'
$ExpectedSha256 = '497c8c2a7e5031f6aa847f88104aa80a93532ec32ee17bdb8d1d2f67a194a9c7'
$Url = 'https://services.gradle.org/distributions/gradle-9.6.0-wrapper.jar'

if (Test-Path -LiteralPath $Target) {
    exit 0
}

$Temp = "$Target.tmp.$PID"
try {
    Invoke-WebRequest -Uri $Url -OutFile $Temp -UseBasicParsing
    $Actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $Temp).Hash.ToLowerInvariant()
    if ($Actual -ne $ExpectedSha256) {
        throw "Gradle wrapper JAR checksum mismatch. Expected $ExpectedSha256, got $Actual"
    }
    Move-Item -Force -LiteralPath $Temp -Destination $Target
}
finally {
    if (Test-Path -LiteralPath $Temp) {
        Remove-Item -Force -LiteralPath $Temp
    }
}
