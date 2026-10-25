import importlib.util
import unittest
from pathlib import Path

spec=importlib.util.spec_from_file_location('queue_guard',Path(__file__).resolve().parent/'check-runner-queue.py')
module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
class QueueGuardTests(unittest.TestCase):
    def test_available_or_completed_runner_is_not_cancelled(self):
        for status in ('in_progress','completed'):
            module.await_runner(lambda:{'jobs':[{'name':module.TARGET,'status':status}]},clock=lambda:0,pause=lambda _:self.fail('unexpected wait'))
    def test_missing_queued_waiting_are_bounded(self):
        for status in (None,'queued','waiting'):
            now=[0]
            def pause(seconds):now[0]+=seconds
            payload={'jobs':[] if status is None else [{'name':module.TARGET,'status':status}]}
            with self.assertRaisesRegex(RuntimeError,'restore online'):
                module.await_runner(lambda:payload,timeout=10,clock=lambda:now[0],pause=pause)
            self.assertEqual(now[0],10)
    def test_transition_to_running_succeeds(self):
        now=[0]
        def pause(seconds):now[0]+=seconds
        def read():return {'jobs':[{'name':module.TARGET,'status':'queued' if now[0]<5 else 'in_progress'}]}
        module.await_runner(read,timeout=10,clock=lambda:now[0],pause=pause)
        self.assertEqual(now[0],5)
    def test_unknown_duplicate_and_unreadable_fail_closed(self):
        for payload in ({},{'jobs':[{'name':module.TARGET,'status':'unexpected'}]},{'jobs':[{'name':module.TARGET},{'name':module.TARGET}]}):
            with self.assertRaises(RuntimeError):module.await_runner(lambda:payload,clock=lambda:0,pause=lambda _:self.fail('unexpected wait'))
    def test_workflow_uses_native_fail_fast_and_ovh_only_payload(self):
        source=(Path(__file__).resolve().parent.parent/'.github/workflows/browser-reliability.yml').read_text()
        self.assertIn('fail-fast: true',source)
        self.assertIn('runner: [self-hosted, ovh, uiai-engine]',source)
        self.assertIn('actions: read',source)
        self.assertNotIn('apt-get install',source)
        self.assertNotIn('actions: write',source)
if __name__=='__main__':unittest.main()
