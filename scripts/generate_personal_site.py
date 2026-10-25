#!/usr/bin/env python3
"""Generate personal endpoints from an engine lock and verified public bundle."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
from generate_bootstraps import generate


def generate_site(lock_path, release_dir, version, output):
    if not re.fullmatch(r'v\d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?', version):
        raise ValueError('Invalid personal bundle version')
    filename = f'outfit-personal_{version}.tar.gz'
    entries = [line.split() for line in (release_dir / 'SHA256SUMS').read_text().splitlines()]
    matches = [parts[0] for parts in entries if len(parts) == 2 and parts[1] == filename]
    actual = hashlib.sha256((release_dir / filename).read_bytes()).hexdigest()
    if len(matches) != 1 or matches[0] != actual:
        raise ValueError('Personal bundle checksum mismatch')
    lock = json.loads(lock_path.read_text())
    lock['profiles'] = {'url': f'https://github.com/johnhenning/outfit-personal/releases/download/{version}/{filename}',
                        'sha256': actual, 'version': version}
    profiles = ['john-mac-everyday', 'john-mac-dev']
    generate(lock, output, profiles=profiles, default_profile='john-mac-everyday')
    generate(lock, output / 'v1', profiles=profiles, default_profile='john-mac-everyday')
    shutil.copyfile(Path(__file__).resolve().parents[1] / 'site/personal/index.html', output / 'index.html')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('lock', type=Path)
    parser.add_argument('release_dir', type=Path)
    parser.add_argument('version')
    parser.add_argument('output', type=Path)
    args = parser.parse_args()
    generate_site(args.lock, args.release_dir, args.version, args.output)
