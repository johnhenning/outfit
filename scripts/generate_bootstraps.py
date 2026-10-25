#!/usr/bin/env python3
"""Generate self-contained bootstrap endpoints from verified release metadata.

No Python is required on the destination machine. This script runs during release.
"""
import argparse,json,re
from pathlib import Path

def validate(lock):
 if not re.fullmatch(r'v\d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?',lock['version']):raise ValueError('invalid release version')
 if not re.fullmatch(r'[\w.-]+/[\w.-]+',lock['repository']):raise ValueError('invalid release repo')
 for key,value in lock['engines'].items():
  if key not in [f'{s}_{a}' for s in ['darwin','linux','windows'] for a in ['amd64','arm64']] or not re.fullmatch('[a-f0-9]{64}',value):raise ValueError('invalid engine checksum')
 if not lock['profiles']['url'].startswith('https://') or any(x in lock['profiles']['url'] for x in "'\"`$\n\r "):raise ValueError('invalid profile URL')
 if not re.fullmatch('[a-f0-9]{64}',lock['profiles']['sha256']):raise ValueError('invalid profile checksum')

def shell(lock,mode,profile=''):
 validate(lock)
 if mode not in ['run','install'] or (profile and not re.fullmatch(r'[a-z0-9-]+',profile)):raise ValueError('invalid bootstrap mode/profile')
 checks='\n'.join(f'    {k}) expected={v} ;;' for k,v in sorted(lock['engines'].items()) if not k.startswith('windows'))
 action='''  install_dir=${OUTFIT_INSTALL_DIR:-"$HOME/.local/bin"}
  mkdir -p "$install_dir"
  [ ! -L "$install_dir/outfit" ] || { echo 'Refusing to replace a symlink at the Outfit install path.' >&2; exit 1; }
  staged=$(mktemp "$install_dir/.outfit-XXXXXXXX")
  cp "$binary" "$staged"
  chmod 755 "$staged"
  mv -f "$staged" "$install_dir/outfit"
  printf 'Installed Outfit in %s/outfit\\nAdd that directory to PATH, then run outfit setup.\\n' "$install_dir"
'''
 if mode=='run':
  args=f"setup --source '{lock['profiles']['url']}' --sha256 '{lock['profiles']['sha256']}'"+(f" --profile '{profile}'" if profile else '')
  action=f'''  # The downloaded program gets terminal input, never the script's pipe.
  unattended=false
  for argument in "$@"; do case "$argument" in --yes|-y|--dry-run) unattended=true ;; esac; done
  if [ "$unattended" = true ]; then
    "$binary" {args} "$@"
  elif [ -t 0 ]; then
    "$binary" {args} "$@"
  elif ( : </dev/tty ) 2>/dev/null; then
    "$binary" {args} "$@" </dev/tty
  else
    echo 'Interactive setup needs a terminal. Inspect the profile, then rerun with --yes for unattended setup.' >&2
    exit 1
  fi
'''
 return f'''#!/bin/sh
# Outfit {lock['version']}. Generated from a release lock; all downloads verify before execution.
# The final function invocation is deliberately last so a truncated download cannot start setup.
outfit_bootstrap() {{
  set -eu
  case "$(uname -s)" in Darwin) os=darwin ;; Linux) os=linux ;; *) echo 'Use install.ps1 or a supported Linux/macOS shell.' >&2; exit 1 ;; esac
  case "$(uname -m)" in arm64|aarch64) arch=arm64 ;; x86_64|amd64) arch=amd64 ;; *) echo 'Unsupported architecture.' >&2; exit 1 ;; esac
  case "${{os}}_${{arch}}" in
{checks}
    *) echo 'No verified release for this platform.' >&2; exit 1 ;;
  esac
  command -v curl >/dev/null 2>&1 || {{ echo 'curl is required.' >&2; exit 1; }}
  tmp=$(mktemp -d "${{TMPDIR:-/tmp}}/outfit.XXXXXXXX")
  trap 'rm -rf "$tmp"' EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  binary="$tmp/outfit"
  curl -fsSL --proto '=https' --proto-redir '=https' --retry 2 --connect-timeout 15 --max-time 300 'https://github.com/{lock['repository']}/releases/download/{lock['version']}/outfit_{lock['version']}_'"${{os}}_${{arch}}" -o "$binary"
  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$binary"); actual=${{actual%% *}}
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$binary"); actual=${{actual%% *}}
  elif command -v openssl >/dev/null 2>&1; then
    actual=$(openssl dgst -sha256 "$binary"); actual=${{actual##* }}
  else
    echo 'A SHA-256 tool (sha256sum, shasum or openssl) is required.' >&2; exit 1
  fi
  [ "$actual" = "$expected" ] || {{ echo 'Outfit checksum mismatch; nothing was executed.' >&2; exit 1; }}
  chmod 700 "$binary"
{action}}}
outfit_bootstrap "$@"
'''

def powershell(lock,mode,profile=''):
 validate(lock)
 checks='\n'.join(f"        '{k.split('_')[1]}' {{ $expected = '{v}' }}" for k,v in sorted(lock['engines'].items()) if k.startswith('windows'))
 action='''    $installDir = if ($env:OUTFIT_INSTALL_DIR) { $env:OUTFIT_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Outfit/bin' }
    if ($DryRun) { Write-Host "Would install Outfit in $installDir"; return }
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
    Write-Host "Installed Outfit in $destination. Add its directory to PATH, then run outfit setup."
'''
 if mode=='run':
  args=['setup','--source',lock['profiles']['url'],'--sha256',lock['profiles']['sha256']]+(['--profile',profile] if profile else [])
  literal=', '.join("'"+a+"'" for a in args)
  action=f'''    $runArgs = @({literal})
    if ($Yes) {{ $runArgs += '--yes' }}
    if ($DryRun) {{ $runArgs += '--dry-run' }}
    $runArgs += $OutfitArgs
    & $binary @runArgs
    if ($LASTEXITCODE -ne 0) {{ throw "Outfit exited with code $LASTEXITCODE" }}
'''
 return f'''# Outfit {lock['version']}: generated, checksum-verified bootstrap.
& {{
param([switch]$Yes, [switch]$DryRun, [Parameter(ValueFromRemainingArguments=$true)][string[]]$OutfitArgs)
$ErrorActionPreference = 'Stop'
$arch = switch ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()) {{ 'x64' {{ 'amd64' }} 'arm64' {{ 'arm64' }} default {{ throw 'Unsupported Windows architecture' }} }}
switch ($arch) {{
{checks}
        default {{ throw 'No verified release for this architecture' }}
}}
$tmp = Join-Path ([IO.Path]::GetTempPath()) ('outfit-' + [Guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($tmp) | Out-Null
try {{
    $binary = Join-Path $tmp 'outfit.exe'
    Invoke-WebRequest -UseBasicParsing -Uri "https://github.com/{lock['repository']}/releases/download/{lock['version']}/outfit_{lock['version']}_windows_$arch.exe" -OutFile $binary
    $actual = (Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {{ throw 'Outfit checksum mismatch; nothing was executed.' }}
{action}}} finally {{
    Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
}}
}} @args
'''

def generate(lock,out,profiles=None,default_profile=''):
 out.mkdir(parents=True,exist_ok=True)
 for mode in ['install','run']:
  profile=default_profile if mode=='run' else ''
  (out/(mode+'.sh')).write_text(shell(lock,mode,profile));(out/(mode+'.ps1')).write_text(powershell(lock,mode,profile))
 for profile in (profiles if profiles is not None else ['workstation','dev','compute','media']):
  dest=out/'profiles';dest.mkdir(exist_ok=True);(dest/(profile+'.sh')).write_text(shell(lock,'run',profile));(dest/(profile+'.ps1')).write_text(powershell(lock,'run',profile))
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('lock',type=Path);p.add_argument('output',type=Path);p.add_argument('--profiles',nargs='+');p.add_argument('--default-profile',default='');a=p.parse_args();generate(json.loads(a.lock.read_text()),a.output,a.profiles,a.default_profile)
