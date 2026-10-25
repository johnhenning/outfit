#!/usr/bin/env python3
"""Build a pinned config bundle, six standalone CLIs, and matching URL bootstraps."""
import argparse,gzip,hashlib,io,json,os,subprocess,tarfile
from pathlib import Path
from generate_bootstraps import generate
ROOT=Path(__file__).resolve().parents[1]
def bundle(root,destination):
 stream=io.BytesIO()
 with tarfile.open(fileobj=stream,mode='w') as tar:
  for p in sorted(root.rglob('*')):
   rel=p.relative_to(root)
   if any(s in ['.git','dist','.local','__pycache__'] for s in rel.parts):continue
   if p.is_symlink():raise ValueError('Profile bundles cannot contain symlinks')
   if not p.is_file():continue
   data=p.read_bytes();info=tarfile.TarInfo(rel.as_posix());info.size=len(data);info.mode=0o600;info.mtime=0;tar.addfile(info,io.BytesIO(data))
 with destination.open('wb') as target:
  with gzip.GzipFile(fileobj=target,mode='wb',mtime=0,filename='') as compressed:compressed.write(stream.getvalue())
 return hashlib.sha256(destination.read_bytes()).hexdigest()
def main():
 p=argparse.ArgumentParser();p.add_argument('--version',required=True);p.add_argument('--profiles',type=Path,required=True);p.add_argument('--profiles-version');p.add_argument('--go',default='go');p.add_argument('--output',type=Path,default=ROOT/'dist');a=p.parse_args();a.output.mkdir(parents=True,exist_ok=True);pv=a.profiles_version or a.version;archive=a.output/f'outfit-profiles_{pv}.tar.gz';sha=bundle(a.profiles.resolve(),archive)
 url=f'https://github.com/johnhenning/outfit-profiles/releases/download/{pv}/{archive.name}'
 lock={'version':a.version,'repository':'johnhenning/outfit','profiles':{'url':url,'sha256':sha,'version':pv},'engines':{}}
 for system in ['darwin','linux','windows']:
  for arch in ['amd64','arm64']:
   name=f'outfit_{a.version}_{system}_{arch}'+('.exe' if system=='windows' else '');asset=a.output/name;env=os.environ.copy();env.update(GOOS=system,GOARCH=arch,CGO_ENABLED='0');flags=f'-s -w -X main.version={a.version} -X github.com/johnhenning/outfit/internal/cli.DefaultSource={url} -X github.com/johnhenning/outfit/internal/cli.DefaultSHA256={sha}'
   subprocess.run([a.go,'build','-trimpath','-ldflags',flags,'-o',str(asset.resolve()),'./cmd/outfit'],cwd=ROOT,env=env,check=True);lock['engines'][f'{system}_{arch}']=hashlib.sha256(asset.read_bytes()).hexdigest();print('built',name,flush=True)
 (a.output/'release-lock.json').write_text(json.dumps(lock,indent=2)+'\n');generate(lock,a.output/'site/tools/outfit');generate(lock,a.output/'site/tools/outfit/v1');(a.output/'SHA256SUMS').write_text(''.join(hashlib.sha256(f.read_bytes()).hexdigest()+'  '+f.name+'\n' for f in sorted(a.output.iterdir()) if f.is_file() and f.name!='SHA256SUMS'))
 source=ROOT/'site/tools/outfit/index.html'
 if source.exists():(a.output/'site/tools/outfit/index.html').write_bytes(source.read_bytes())
 print('Release lock and bootstrap endpoints written to',a.output)
if __name__=='__main__':main()
