import importlib.util,sys,tempfile,unittest
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2]
sys.path.insert(0,str(ROOT/'scripts'))
import release
class ReleaseTests(unittest.TestCase):
 def test_personal_site_pins_its_own_verified_bundle(self):
  import generate_personal_site
  import hashlib,json
  with tempfile.TemporaryDirectory() as d:
   root=Path(d);archive=root/'outfit-personal_v0.1.0-alpha.2.tar.gz';archive.write_bytes(b'synthetic personal archive')
   sha=hashlib.sha256(archive.read_bytes()).hexdigest();(root/'SHA256SUMS').write_text(sha+'  '+archive.name+'\n')
   lock={'version':'v0.1.0-alpha.1','repository':'johnhenning/outfit','engines':{'darwin_arm64':'a'*64},'profiles':{'url':'https://example.com/public.tar.gz','sha256':'b'*64}}
   lockpath=root/'release-lock.json';lockpath.write_text(json.dumps(lock))
   generate_personal_site.generate_site(lockpath,root,'v0.1.0-alpha.2',root/'site')
   script=(root/'site/profiles/john-mac-dev.sh').read_text()
   self.assertIn('outfit-personal/releases/download/v0.1.0-alpha.2/',script)
   self.assertIn(sha,script)
   self.assertNotIn('https://example.com/public.tar.gz',script)
   archive.write_bytes(b'corrupted archive')
   with self.assertRaises(ValueError):generate_personal_site.generate_site(lockpath,root,'v0.1.0-alpha.2',root/'corrupt-site')
   self.assertFalse((root/'corrupt-site/run.sh').exists())
   with self.assertRaises(ValueError):generate_personal_site.generate_site(lockpath,root,'../../invalid',root/'invalid-site')
 def test_engine_checksums_cover_only_engine_downloads(self):
  with tempfile.TemporaryDirectory() as d:
   root=Path(d)
   for name in ['outfit_v0.1.0_linux_amd64','outfit-site.tar.gz','release-lock.json','outfit-profiles_v0.1.0.tar.gz','outfit_v0.0.1_linux_amd64']:(root/name).write_bytes(name.encode())
   release.write_checksums(root,'v0.1.0')
   names=[line.split('  ',1)[1] for line in (root/'SHA256SUMS').read_text().splitlines()]
   self.assertEqual(names,['outfit-site.tar.gz','outfit_v0.1.0_linux_amd64','release-lock.json'])
if __name__=='__main__':unittest.main()
