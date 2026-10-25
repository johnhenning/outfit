# Pester 5: synthetic downloads and a temporary install directory, never real setup.
BeforeAll {
    $fixture = Join-Path $PSScriptRoot '../fixtures/bootstrap'
    $installScript = Join-Path $fixture 'install.ps1'
    $binaryFixture = Join-Path $fixture 'binary.fixture'
}
Describe 'Outfit Windows bootstrap' {
    BeforeEach {
        $env:OUTFIT_INSTALL_DIR = Join-Path $TestDrive ([Guid]::NewGuid().ToString('N'))
        Mock Invoke-WebRequest { Copy-Item -LiteralPath $binaryFixture -Destination $OutFile }
    }
    AfterAll { Remove-Item Env:OUTFIT_INSTALL_DIR -ErrorAction SilentlyContinue }
    It 'installs only the checked CLI bytes' {
        & $installScript
        $destination = Join-Path $env:OUTFIT_INSTALL_DIR 'outfit.exe'
        (Get-FileHash $destination).Hash | Should -Be (Get-FileHash $binaryFixture).Hash
    }
    It 'rejects checksum mismatch without replacing the old CLI' {
        New-Item -ItemType Directory $env:OUTFIT_INSTALL_DIR | Out-Null
        $destination = Join-Path $env:OUTFIT_INSTALL_DIR 'outfit.exe'
        [IO.File]::WriteAllText($destination, 'previous CLI')
        Mock Invoke-WebRequest { [IO.File]::WriteAllText($OutFile, 'corrupt download') }
        { & $installScript } | Should -Throw '*checksum mismatch*'
        [IO.File]::ReadAllText($destination) | Should -Be 'previous CLI'
    }
    It 'preserves the old CLI if staging fails' {
        New-Item -ItemType Directory $env:OUTFIT_INSTALL_DIR | Out-Null
        $destination = Join-Path $env:OUTFIT_INSTALL_DIR 'outfit.exe'
        [IO.File]::WriteAllText($destination, 'previous CLI')
        Mock Copy-Item { throw 'simulated disk failure' } -ParameterFilter { $Destination -like '*.outfit-*' }
        { & $installScript } | Should -Throw
        [IO.File]::ReadAllText($destination) | Should -Be 'previous CLI'
    }
    It 'atomically replaces an existing CLI' {
        New-Item -ItemType Directory $env:OUTFIT_INSTALL_DIR | Out-Null
        $destination = Join-Path $env:OUTFIT_INSTALL_DIR 'outfit.exe'
        [IO.File]::WriteAllText($destination, 'previous CLI')
        & $installScript
        (Get-FileHash $destination).Hash | Should -Be (Get-FileHash $binaryFixture).Hash
        @(Get-ChildItem $env:OUTFIT_INSTALL_DIR -Filter '.outfit-*' -Force).Count | Should -Be 0
    }
    It 'all generated scripts parse as PowerShell' {
        Get-ChildItem $fixture -Filter '*.ps1' -Recurse | ForEach-Object {
            $tokens = $null; $parseErrors = $null
            $null = [Management.Automation.Language.Parser]::ParseFile($_.FullName, [ref]$tokens, [ref]$parseErrors)
            $parseErrors.Count | Should -Be 0
        }
    }
}
