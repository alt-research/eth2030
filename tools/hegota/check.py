#!/usr/bin/env python3
"""Check fixed Hegotá A artifacts offline; compare source updates without adopting them."""
import argparse
import hashlib
import json
import re
import subprocess
from pathlib import Path
from codec_goldens import vectors
from semantic_goldens import check as check_semantics

ROOT = Path(__file__).resolve().parents[2]

def require(condition, message):
    if not condition: raise ValueError(message)

def read(path):return json.loads(path.read_text())
def digest(path):return hashlib.sha256(path.read_bytes()).hexdigest()

def validate_admission(data, expected, task_ids):
    rows=data['records'];ids=[r['eip'] for r in rows]
    require(data['count']==72 and len(ids)==72 and len(set(ids))==72 and set(ids)==expected,'72-EIP scope is incomplete or duplicated')
    for r in rows:
        require(r['disposition'] in {'implement','conditional','watch','excluded'},f"EIP-{r['eip']}: unknown disposition")
        require(r['tasks'] and set(r['tasks']) <= task_ids,f"EIP-{r['eip']}: invalid task mapping")
        require(r['reason'] and r['activation_conditions'] and r['discussion'] and r['history'],f"EIP-{r['eip']}: missing admission evidence")
        h=r['history'][-1]
        require(h['new_spec_sha256']==r['spec_sha256'] and h['affected_tasks'] and h['date'] and h['rationale'],f"EIP-{r['eip']}: invalid change history")

def validate_lock(lock, root):
    require(lock['version']==1 and len(lock['repositories'])>=14,'missing source repositories')
    for r in lock['repositories']:
        require(re.fullmatch('[0-9a-f]{40}',r['sha']) is not None,'source is not commit pinned')
    locals_checked=0
    for f in lock['files']:
        require(re.fullmatch('[0-9a-f]{64}',f['sha256']) is not None,'source lacks SHA-256')
        require(re.search('/[0-9a-f]{40}/',f['url']) is not None,'normative source URL is not pinned')
        if 'local_path' in f:
            p=root/f['local_path'];require(p.is_file(),f"missing source: {f['local_path']}")
            require(digest(p)==f['sha256'],f"source changed: {f['local_path']}");locals_checked+=1
    require(digest(root/'pkg/go.sum')==lock['go_sum_sha256'],'Go dependency lock changed without source update')
    return locals_checked

def compare(old, new, old_admission, new_admission):
    before={r['eip']:r for r in old_admission['records']};after={r['eip']:r for r in new_admission['records']}
    require(set(before)==set(after),'scope additions/removals require a new explicit 72-EIP baseline version')
    changes=[]
    for eip in sorted(before):
        a,b=before[eip],after[eip]
        if a['spec_sha256']==b['spec_sha256'] and a['disposition']==b['disposition']:continue
        h=b['history'][-1]
        require(h.get('previous_local_sha256')==a['spec_sha256'] and h.get('new_spec_sha256')==b['spec_sha256'] and h.get('affected_tasks') and h.get('rationale') and h.get('date'),f'EIP-{eip}: source/admission update lacks an old/new impact decision')
        changes.append({'eip':eip,'old_sha256':a['spec_sha256'],'new_sha256':b['spec_sha256'],'old_disposition':a['disposition'],'new_disposition':b['disposition'],'affected_tasks':h['affected_tasks'],'reason':h['rationale']})
    oldrepos={(r['repo'],r['ref']):r['sha'] for r in old['repositories']}
    repos=[{'repo':r['repo'],'ref':r['ref'],'old':oldrepos.get((r['repo'],r['ref'])),'new':r['sha']} for r in new['repositories'] if oldrepos.get((r['repo'],r['ref']))!=r['sha']]
    oldfiles={f['url']:f['sha256'] for f in old['files']}
    files=[{'url':f['url'],'old_sha256':oldfiles.get(f['url']),'new_sha256':f['sha256']} for f in new['files'] if oldfiles.get(f['url'])!=f['sha256']]
    return {'adopted':False,'eip_changes':changes,'repository_changes':repos,'file_changes':files,'removed_source_urls':sorted(set(oldfiles)-{f['url'] for f in new['files']})}

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--compare-lock',type=Path);parser.add_argument('--compare-admission',type=Path);args=parser.parse_args()
    lock=read(ROOT/'specs/hegota/source-lock.json');admission=read(ROOT/'specs/hegota/admission.json')
    source=read(ROOT/'docs/plans/hegota-sources-2026-11.json');expected={r['eip'] for r in source['records'] if r['in_review_scope']}
    tasks=set(re.findall(r'^\| (HGT-\d{3}) \|',(ROOT/'docs/plans/hegota-development-plan-2026-11.md').read_text(),re.M))
    validate_admission(admission,expected,tasks)
    count=validate_lock(lock,ROOT)
    for path,repo in [('refs/EIPs','ethereum/EIPs'),('refs/consensus-specs','ethereum/consensus-specs')]:
        expected_sha=next(r['sha'] for r in lock['repositories'] if r['repo']==repo)
        actual=subprocess.check_output(['git','-C',str(ROOT/path),'rev-parse','HEAD'],text=True).strip();require(actual==expected_sha,f'{path}: HEAD differs from source lock')
    entries={r['eip']:r for r in admission['records']}
    for r in source['records']:
        if r['in_review_scope']:require(entries[r['eip']]['spec_sha256']==r['spec_sha256'],f"EIP-{r['eip']}: research/admission drift")
    profile=read(ROOT/'specs/hegota/profiles.json');require({d['id'] for d in profile['decisions']}=={f'D{i:02}' for i in range(1,10)},'missing composition decision')
    require(next(p for p in profile['profiles'] if p['id']=='hegota-release')['activation']=='none','release profile activated without verification')
    require(read(ROOT/'specs/hegota/codec-goldens.json')==vectors(),'independent goldens changed')
    check_semantics()
    for r in read(ROOT/'specs/hegota/conformance.json')['requirements']:
        require((ROOT/r['caller']).is_file(),f"missing runtime evidence {r['caller']}")
        require(r['owner'] and r['positive_vector'] and r['negative_vector'] and r['oracle'] and r['open_question'] and r['spec_commit'],'incomplete conformance record')
    task_register=read(ROOT/'specs/hegota/tasks.json')
    require({t['id'] for t in task_register['tasks']}=={'HGT-001','HGT-004','HGT-005','HGT-006','HGT-007','HGT-008','HGT-009'},'A task omission')
    for task in task_register['tasks']:
        require(task['status']=='verified' and task['owner'] and task['scope_limit'],'incomplete A task status')
        for path in task['artifacts']:require((ROOT/path).is_file(),f'missing A artifact: {path}')
    runtime=read(ROOT/'evidence/hegota-a/runtime.json')
    for path,expected_hash in runtime['source_file_sha256'].items():
        require(digest(ROOT/path)==expected_hash,f'runtime evidence source changed: {path}')
    verification=read(ROOT/'evidence/hegota-a/verification.json')
    for path,expected_hash in verification['artifacts'].items():
        require(digest(ROOT/path)==expected_hash,f'verification artifact changed: {path}')
    baseline=read(ROOT/'evidence/hegota-a/baseline.json')
    require(baseline['expected']==baseline['executed']==baseline['passed']==12 and baseline['failed']==baseline['skipped']==0,'baseline incomplete')
    if args.compare_lock or args.compare_admission:
        require(args.compare_lock and args.compare_admission,'provide both update artifacts')
        newadmission=read(args.compare_admission);validate_admission(newadmission,expected,tasks)
        print(json.dumps(compare(lock,read(args.compare_lock),admission,newadmission),ensure_ascii=False,indent=2))
    else:print(json.dumps({'result':'pass','scope_eips':72,'local_source_hashes_checked':count,'source_files_locked':len(lock['files']),'profiles':len(profile['profiles']),'decisions':9,'codec_goldens':7},ensure_ascii=False))
if __name__=='__main__':main()
