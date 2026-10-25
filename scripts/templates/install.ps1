    $installDir = if ($env:OUTFIT_INSTALL_DIR) { $env:OUTFIT_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Outfit/bin' }
    $installDir = [IO.Path]::GetFullPath($installDir)
    if ($DryRun) { Write-Host "Would install Outfit in $installDir"; return }
    [Console]::Error.WriteLine('  [3/3] Installing the CLI...')
    [IO.Directory]::CreateDirectory($installDir) | Out-Null
    $destination = Join-Path $installDir 'outfit.exe'
    if ((Test-Path $destination) -and ((Get-Item $destination).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Refusing a symlink/reparse point at the Outfit install path.' }
    $staged = Join-Path $installDir ('.outfit-' + [Guid]::NewGuid().ToString('N') + '.exe')
    try {
        Copy-Item -LiteralPath $binary -Destination $staged
        if ((Get-FileHash -LiteralPath $staged -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expected) { throw 'Staged Outfit checksum mismatch' }
        if (Test-Path $destination) { [IO.File]::Replace($staged, $destination, [NullString]::Value) }
        else { [IO.File]::Move($staged, $destination) }
    } finally { if (Test-Path $staged) { Remove-Item -LiteralPath $staged -Force } }
    Write-Host ''
    if (-not [Console]::IsOutputRedirected -and $env:TERM -ne 'dumb' -and -not [Environment]::GetEnvironmentVariables().Contains('NO_COLOR')) {
        Write-Host '  [OK] Outfit @VERSION@ installed successfully.' -ForegroundColor Green
    } else { Write-Host '  [OK] Outfit @VERSION@ installed successfully.' }
    Write-Host "       Binary: $destination"
    Write-Host '       Download verified with SHA-256.'
    Write-Host ''
    $resolved = Get-Command outfit -ErrorAction SilentlyContinue | Select-Object -First 1
    $nextCommand = 'outfit'
    if ($resolved -and $resolved.CommandType -eq 'Application' -and [string]::Equals($resolved.Source, $destination, [StringComparison]::OrdinalIgnoreCase)) {
        Write-Host '  PATH is ready. Start here:'
    } else {
        if ($resolved -and $resolved.CommandType -eq 'Application') {
            Write-Host ('  Another Outfit is earlier on PATH: ' + $resolved.Source)
        } elseif ($resolved) {
            Write-Host '  An alias or function named outfit takes precedence. The commands below use the installed file directly.'
            $nextCommand = "& '" + $destination.Replace("'", "''") + "'"
        }
        $quotedDir = "'" + $installDir.Replace("'", "''") + "'"
        Write-Host '  Enable Outfit in this PowerShell terminal:'
        Write-Host ''
        Write-Host ('    $env:Path = ' + $quotedDir + ' + '';'' + $env:Path')
        Write-Host ''
        Write-Host '  To keep it for future terminals, run:'
        Write-Host ''
        Write-Host ('    $outfitBin = ' + $quotedDir)
        Write-Host '    $userEntries = [Environment]::GetEnvironmentVariable(''Path'', ''User'') -split '';'' | Where-Object { $_ -and $_ -ne $outfitBin }'
        Write-Host '    $newUserPath = (@($outfitBin) + @($userEntries)) -join '';'''
        Write-Host '    [Environment]::SetEnvironmentVariable(''Path'', $newUserPath, ''User'')'
        Write-Host ''
        Write-Host '  Open a new terminal after saving your user PATH. No PATH settings were changed by this installer.'
        if ($resolved -and $resolved.CommandType -eq 'Application') {
            Write-Host '  If the older CLI is on the system PATH, update/remove that old entry in Environment Variables too; Windows places system PATH entries before user entries.'
        }
        Write-Host ''
        Write-Host '  Then start here:'
    }
    Write-Host ''
    Write-Host ("    $nextCommand version      # Check the CLI")
    Write-Host ("    $nextCommand setup        # Choose a profile and review software")
    Write-Host ("    $nextCommand doctor       # Check tooling and credential setup")
    Write-Host ''
    Write-Host '  Dotfiles are separate: review with outfit dotfiles before applying.'
    Write-Host '  WSL: run the Linux bootstrap inside your chosen distribution for Linux tooling.'
    Write-Host '  Help and profiles: https://getoutfit.dev/'
