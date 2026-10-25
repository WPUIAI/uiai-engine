#!/usr/bin/env python3
"""Bound a single workflow's runner wait; native matrix fail-fast owns cancellation."""
import json
import os
import re
import sys
import time
import urllib.request

TARGET = 'Browser Reliability (browser)'

def await_runner(read_jobs, timeout=120, clock=time.monotonic, pause=time.sleep):
    deadline=clock()+timeout
    while True:
        payload=read_jobs()
        if not isinstance(payload,dict) or not isinstance(payload.get('jobs'),list):
            raise RuntimeError('runner queue metadata unreadable; check Actions permissions')
        matches=[job for job in payload['jobs'] if job.get('name')==TARGET]
        if len(matches)>1:
            raise RuntimeError('runner queue target is ambiguous')
        if matches:
            status=matches[0].get('status')
            if status in ('in_progress','completed'):
                return
            if status not in ('queued','waiting','pending'):
                raise RuntimeError('runner queue state unknown; inspect this run')
        if clock()>=deadline:
            raise RuntimeError('OVH browser runner did not start within120s; restore online ovh/uiai-engine labels and Chrome capacity, then rerun this workflow')
        pause(min(5,max(0,deadline-clock())))

def main():
    repo=os.environ.get('GITHUB_REPOSITORY','')
    run_id=os.environ.get('GITHUB_RUN_ID','')
    token=os.environ.get('GITHUB_TOKEN','')
    if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+',repo) or not run_id.isdigit() or not token:
        raise RuntimeError('exact workflow identity and Actions read token are required')
    request=urllib.request.Request(f'https://api.github.com/repos/{repo}/actions/runs/{run_id}/jobs?per_page=100',headers={'Authorization':'Bearer '+token,'Accept':'application/vnd.github+json','X-GitHub-Api-Version':'2022-11-28'})
    def read_jobs():
        try:
            with urllib.request.urlopen(request,timeout=10) as response:return json.load(response)
        except Exception:
            raise RuntimeError('runner queue API unavailable; inspect this run and Actions read permissions') from None
    await_runner(read_jobs)
    print('OVH browser lane admitted; queue guard complete')

if __name__=='__main__':
    try:main()
    except RuntimeError as error:
        print('::error::'+str(error),file=sys.stderr)
        raise SystemExit(1)
