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
    It 'reports success and copyable PowerShell PATH instructions' {
        $env:OUTFIT_INSTALL_DIR = Join-Path $TestDrive "John's tools"
        $output = (& $installScript 6>&1 | Out-String)
        $output | Should -Match 'installed successfully'
        $output | Should -Match 'v0.0.0-test'
        $output | Should -Match 'outfit version'
        $output | Should -Match 'outfit setup'
        $output | Should -Match 'outfit doctor'
        $line = @($output -split '\r?\n' | Where-Object { $_.Trim().StartsWith('$env:Path =') })[0].Trim()
        $previous = $env:Path
        try {
            & ([scriptblock]::Create($line))
            ($env:Path -split ';')[0] | Should -Be $env:OUTFIT_INSTALL_DIR
        } finally { $env:Path = $previous }
    }
    It 'does not report PATH ready when an older CLI shadows the installed CLI' {
        Mock Get-Command { [pscustomobject]@{ Source = 'C:\older\outfit.exe'; CommandType = 'Application' } } -ParameterFilter { $Name -eq 'outfit' }
        $previous = $env:Path
        try {
            $env:Path = $env:Path + ';' + $env:OUTFIT_INSTALL_DIR
            $output = (& $installScript 6>&1 | Out-String)
            $output | Should -Not -Match 'PATH is ready'
            $output | Should -Match 'Another Outfit is earlier on PATH'
        } finally { $env:Path = $previous }
    }
    It 'reports PATH ready when command resolution points to the installed CLI' {
        Mock Get-Command { [pscustomobject]@{ Source = (Join-Path $env:OUTFIT_INSTALL_DIR 'outfit.exe'); CommandType = 'Application' } } -ParameterFilter { $Name -eq 'outfit' }
        $output = (& $installScript 6>&1 | Out-String)
        $output | Should -Match 'PATH is ready'
        $output | Should -Not -Match '\$env:Path ='
    }
    It 'printed user PATH calculation moves an existing entry to the front without duplicates' {
        $output = (& $installScript 6>&1 | Out-String)
        $lines = @($output -split '\r?\n' | ForEach-Object { $_.Trim() } | Where-Object { $_ -match '^\$(outfitBin|userEntries|newUserPath) =' })
        $fixturePath = ('older;' + $env:OUTFIT_INSTALL_DIR + ';other;' + $env:OUTFIT_INSTALL_DIR).Replace("'", "''")
        # Execute only the printed calculation against synthetic input, never the registry write.
        $calculation = ($lines -join "`n").Replace("[Environment]::GetEnvironmentVariable('Path', 'User')", "'$fixturePath'")
        $newPath = & ([scriptblock]::Create($calculation + "`n" + '$newUserPath'))
        @($newPath -split ';')[0] | Should -Be $env:OUTFIT_INSTALL_DIR
        @($newPath -split ';' | Where-Object { $_ -eq $env:OUTFIT_INSTALL_DIR }).Count | Should -Be 1
        @($newPath -split ';').Count | Should -Be 3
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
    It 'dry run never writes an install directory' {
        & $installScript -DryRun
        Test-Path $env:OUTFIT_INSTALL_DIR | Should -BeFalse
    }
    It 'accepts the website scriptblock preview syntax with a simulated target' {
        $downloaded = Get-Content -LiteralPath $installScript -Raw
        & ([scriptblock]::Create($downloaded)) -DryRun --target windows-amd64
        Test-Path $env:OUTFIT_INSTALL_DIR | Should -BeFalse
    }
    It 'all generated scripts parse as PowerShell' {
        Get-ChildItem $fixture -Filter '*.ps1' -Recurse | ForEach-Object {
            $tokens = $null; $parseErrors = $null
            $null = [Management.Automation.Language.Parser]::ParseFile($_.FullName, [ref]$tokens, [ref]$parseErrors)
            $parseErrors.Count | Should -Be 0
        }
    }
}
