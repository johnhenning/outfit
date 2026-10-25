import hashlib,json,os,subprocess,tempfile,unittest
from pathlib import Path
import importlib.util
ROOT=Path(__file__).resolve().parents[2]
class BootstrapTests(unittest.TestCase):
 def setUp(self):
  self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup);self.d=Path(self.tmp.name);(self.d/'bin').mkdir();self.engine=b'#!/bin/sh\nprintf "%s\\n" "$@" > "$TEST_CAPTURE"\n';(self.d/'engine').write_bytes(self.engine)
  spec=importlib.util.spec_from_file_location('generate',ROOT/'scripts/generate_bootstraps.py');self.gen=importlib.util.module_from_spec(spec);spec.loader.exec_module(self.gen)
  self.lock={'version':'v0.1.0','repository':'johnhenning/outfit','engines':{f'{s}_{a}':hashlib.sha256(self.engine).hexdigest() for s in ['darwin','linux','windows'] for a in ['amd64','arm64']},'profiles':{'url':'https://example.com/profiles.tar.gz','sha256':'a'*64}}
  self.env=os.environ.copy();self.env.update(PATH=str(self.d/'bin')+':'+self.env['PATH'],TEST_ENGINE=str(self.d/'engine'),TEST_CAPTURE=str(self.d/'capture'),OUTFIT_INSTALL_DIR=str(self.d/'installed'))
  (self.d/'bin/curl').write_text('#!/bin/sh\nif [ "$TEST_DOWNLOAD_FAIL" = yes ]; then exit 22; fi\nwhile [ "$#" -gt 0 ]; do\n if [ "$1" = -o ]; then shift; cp "$TEST_ENGINE" "$1"; exit; fi\n shift\ndone\nexit 3\n');(self.d/'bin/curl').chmod(0o755)
 def run_script(self,mode='run',profile='compute'):
  script=self.d/'bootstrap.sh';script.write_text(self.gen.shell(self.lock,mode,profile));return subprocess.run(['/bin/sh',str(script),'--yes'],env=self.env,text=True,capture_output=True)
 def test_known_profile_uses_temporary_cli(self):
  r=self.run_script();self.assertEqual(r.returncode,0,r.stderr);args=(self.d/'capture').read_text().splitlines();self.assertEqual(args,['setup','--source','https://example.com/profiles.tar.gz','--sha256','a'*64,'--profile','compute','--yes']);self.assertFalse((self.d/'installed').exists())
 def test_bad_checksum_never_executes(self):
  self.lock['engines']={k:'0'*64 for k in self.lock['engines']};r=self.run_script();self.assertNotEqual(r.returncode,0);self.assertFalse((self.d/'capture').exists())
 def test_download_failure_never_executes(self):
  self.env['TEST_DOWNLOAD_FAIL']='yes';r=self.run_script();self.assertNotEqual(r.returncode,0);self.assertFalse((self.d/'capture').exists())
 def test_install_only_does_not_run_setup(self):
  r=self.run_script('install','');self.assertEqual(r.returncode,0,r.stderr);self.assertTrue((self.d/'installed/outfit').exists());self.assertFalse((self.d/'capture').exists())
 def test_preview_needs_no_terminal(self):
  script=self.d/'preview.sh';script.write_text(self.gen.shell(self.lock,'run','compute'));r=subprocess.run(['/bin/sh',str(script),'--dry-run'],env=self.env,text=True,capture_output=True,stdin=subprocess.DEVNULL);self.assertEqual(r.returncode,0,r.stderr);self.assertEqual((self.d/'capture').read_text().splitlines()[-1],'--dry-run')
 def test_truncated_script_cannot_execute(self):
  body=self.gen.shell(self.lock,'run','compute');script=self.d/'partial.sh';script.write_text(body[:len(body)//2]);r=subprocess.run(['/bin/sh',str(script),'--yes'],env=self.env,capture_output=True);self.assertNotEqual(r.returncode,0);self.assertFalse((self.d/'capture').exists())
 def test_personal_site_runs_an_explicit_default_profile(self):
  self.gen.generate(self.lock,self.d/'site',profiles=['john-mac-everyday','john-mac-dev'],default_profile='john-mac-everyday')
  r=subprocess.run(['/bin/sh',str(self.d/'site/run.sh'),'--dry-run'],env=self.env,text=True,capture_output=True)
  self.assertEqual(r.returncode,0,r.stderr)
  args=(self.d/'capture').read_text().splitlines()
  self.assertEqual(args[args.index('--profile')+1],'john-mac-everyday')
  self.assertTrue((self.d/'site/profiles/john-mac-dev.ps1').exists())
  self.assertFalse((self.d/'site/profiles/workstation.sh').exists())
if __name__=='__main__':unittest.main()
