#!/usr/bin/env python3
"""Generate synthetic bootstrap fixtures for PowerShell/Pester; no live installs."""
from pathlib import Path
import hashlib
from generate_bootstraps import generate
root=Path(__file__).resolve().parents[1]/'tests/fixtures/bootstrap'
root.mkdir(parents=True,exist_ok=True)
binary=b'outfit-bootstrap-test-executable\n';(root/'binary.fixture').write_bytes(binary)
lock={'version':'v0.0.0-test','repository':'example/outfit','engines':{f'{s}_{a}':hashlib.sha256(binary).hexdigest() for s in ['darwin','linux','windows'] for a in ['amd64','arm64']},'profiles':{'url':'https://example.invalid/profiles.tar.gz','sha256':'a'*64}}
generate(lock,root)
