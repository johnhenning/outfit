import importlib.util,sys,tempfile,unittest
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2]
sys.path.insert(0,str(ROOT/'scripts'))
import release
class ReleaseTests(unittest.TestCase):
 def test_engine_checksums_cover_only_engine_downloads(self):
  with tempfile.TemporaryDirectory() as d:
   root=Path(d)
   for name in ['outfit_v0.1.0_linux_amd64','outfit-site.tar.gz','release-lock.json','outfit-profiles_v0.1.0.tar.gz','outfit_v0.0.1_linux_amd64']:(root/name).write_bytes(name.encode())
   release.write_checksums(root,'v0.1.0')
   names=[line.split('  ',1)[1] for line in (root/'SHA256SUMS').read_text().splitlines()]
   self.assertEqual(names,['outfit-site.tar.gz','outfit_v0.1.0_linux_amd64','release-lock.json'])
if __name__=='__main__':unittest.main()
