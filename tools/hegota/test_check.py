import copy
import hashlib
import json
import tempfile
import unittest
from pathlib import Path
from check import ROOT, compare, validate_admission, validate_lock

class RegisterValidation(unittest.TestCase):
 def setUp(self):
  self.admission=json.loads((ROOT/'specs/hegota/admission.json').read_text());self.expected={r['eip'] for r in self.admission['records']};self.tasks={t for r in self.admission['records'] for t in r['tasks']}
 def test_duplicate_eip_is_rejected(self):
  self.admission['records'][1]=self.admission['records'][0]
  with self.assertRaisesRegex(ValueError,'scope'):validate_admission(self.admission,self.expected,self.tasks)
 def test_missing_task_and_history_rejected(self):
  self.admission['records'][0]['tasks']=[]
  with self.assertRaisesRegex(ValueError,'task'):validate_admission(self.admission,self.expected,self.tasks)
 def test_unrecorded_upstream_change_rejected(self):
  old=copy.deepcopy(self.admission);self.admission['records'][0]['spec_sha256']='a'*64
  with self.assertRaisesRegex(ValueError,'impact decision'):compare({'repositories':[],'files':[]},{'repositories':[],'files':[]},old,self.admission)
 def test_recorded_change_reports_impacted_tasks_without_adopting(self):
  old=copy.deepcopy(self.admission);r=self.admission['records'][0];h=copy.deepcopy(r['history'][-1]);h['previous_local_sha256']=r['spec_sha256'];h['new_spec_sha256']='a'*64;h['date']='2026-11-26';r['spec_sha256']='a'*64;r['history'].append(h)
  result=compare({'repositories':[],'files':[]},{'repositories':[],'files':[]},old,self.admission)
  self.assertFalse(result['adopted']);self.assertEqual(result['eip_changes'][0]['affected_tasks'],r['tasks'])
 def test_tampered_normative_file_rejected(self):
  with tempfile.TemporaryDirectory() as d:
   root=Path(d);p=root/'source.md';p.write_text('spec');(root/'pkg').mkdir();(root/'pkg/go.sum').write_text('deps')
   lock={'version':1,'repositories':[{'sha':'a'*40}]*14,'files':[{'url':'https://example.org/'+'a'*40+'/source.md','sha256':hashlib.sha256(b'spec').hexdigest(),'local_path':'source.md'}],'go_sum_sha256':hashlib.sha256(b'deps').hexdigest()}
   self.assertEqual(validate_lock(lock,root),1);p.write_text('tampered')
   with self.assertRaisesRegex(ValueError,'source changed'):validate_lock(lock,root)
if __name__=='__main__':unittest.main()
