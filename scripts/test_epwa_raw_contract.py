import ast
import copy
import pathlib
import re
import unittest

from epwa_raw_contract import RAW_FIELDS, find_raw

ROOT = pathlib.Path(__file__).resolve().parent


class RawContractTests(unittest.TestCase):
    def test_boundary_vocabulary_parity(self):
        mcp = (ROOT.parent / 'mcp/epwa-contract.mjs').read_text()
        fields = re.search(r'RAW_ARTIFACT_FIELDS = new Set\(\[(.*?)\]\)', mcp, re.S)[1]
        self.assertEqual(RAW_FIELDS, set(re.findall(r'"(.*?)"', fields)))
        cli = (ROOT / 'uiai-open-result.sh').read_text()
        fields = re.search(r'forbidden=(\{.*?\})', cli)[1]
        self.assertEqual(RAW_FIELDS, ast.literal_eval(fields))

    def test_recursive_paths_and_empty_placeholders(self):
        for field in RAW_FIELDS:
            with self.subTest(field=field):
                self.assertEqual(find_raw({'session': [{'payload': {field: 'raw'}}]}), f'$.session[0].payload.{field}')
                self.assertEqual(find_raw({field: None}), '')
                self.assertEqual(find_raw({field: ''}), '')
        self.assertEqual(find_raw({'epwa_delivery': {'state': 'ready'}}), '')

    def test_diagnostic_consumers_reject_nested_leaks(self):
        ready = {'artifact_ref': 'artifact:test', 'delivery_state': 'ready',
                 'artifact_url': 'https://epwa-ci.invalid/record',
                 'portable_url': 'https://epwa-ci.invalid/package.zip',
                 'epwa_delivery': {'schema': 'uiai.epwa_delivery.v1', 'state': 'ready',
                                   'artifact': {'artifact_ref': 'artifact:test'},
                                   'epwa': {'record_url': 'https://epwa-ci.invalid/record',
                                            'portable_url': 'https://epwa-ci.invalid/package.zip'}}}
        for script in ('soak-browser-flakiness.sh', 'stress-browser-diagnostics.sh'):
            source = (ROOT / script).read_text()
            self.assertIn('from epwa_raw_contract import find_raw', source)
            definition = re.search(r'(def require_delivery\(.*?)(?=\ndef )', source, re.S)[1]
            namespace = {'find_raw': find_raw}
            exec(compile(ast.parse(definition), script, 'exec'), namespace)
            validate = namespace['require_delivery']
            validate(ready, 'test')
            for field in RAW_FIELDS:
                with self.subTest(script=script, field=field):
                    body = copy.deepcopy(ready)
                    body['session'] = [{'payload': {field: 'raw'}}]
                    with self.assertRaisesRegex((ValueError, AssertionError), r'\$\.session\[0\]\.payload\.' + field):
                        validate(body, 'test')


if __name__ == '__main__':
    unittest.main()
