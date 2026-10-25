import json
import os
import subprocess
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

SCRIPT=Path(__file__).resolve().parent/'warm-browser-session.sh'
class WarmSessionCleanup(unittest.TestCase):
    def test_exact_created_identity_cleanup_for_all_delivery_states(self):
        state={'status':201,'body':True,'header':True,'deletes':[]}
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(200);self.end_headers();self.wfile.write(b'{}')
            def do_POST(self):
                self.rfile.read(int(self.headers.get('Content-Length',0)))
                self.send_response(state['status'])
                if state['header']: self.send_header('X-UIAI-Created-Session-ID','owned-warm-fixture')
                self.end_headers()
                self.wfile.write(json.dumps({'session':{'id':'owned-warm-fixture'}}).encode() if state['body'] else b'not-json')
            def do_DELETE(self):
                state['deletes'].append(self.path);self.send_response(200);self.end_headers();self.wfile.write(b'{}')
            def log_message(self,*args): pass
        server=ThreadingHTTPServer(('127.0.0.1',0),Handler)
        thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
        try:
            env=dict(os.environ,UIAI_WARM_BASE=f'http://127.0.0.1:{server.server_port}',UIAI_WARM_COUNT='1')
            for status,body,header,expected in [(201,True,True,1),(202,True,True,1),(202,False,True,1),(500,False,True,1),(202,False,False,0)]:
                with self.subTest(status=status,body=body,header=header):
                    state.update(status=status,body=body,header=header,deletes=[])
                    result=subprocess.run(['bash',str(SCRIPT)],env=env,capture_output=True,text=True,timeout=10)
                    self.assertEqual(result.returncode,0,result.stderr)
                    self.assertEqual(state['deletes'],['/api/session/owned-warm-fixture']*expected)
                    self.assertNotIn('owned-warm-fixture',result.stdout+result.stderr)
                    if expected:self.assertIn('closed',result.stdout+result.stderr)
                    else:self.assertIn('cleanup=unknown',result.stderr)
        finally:
            server.shutdown();server.server_close();thread.join()
if __name__=='__main__':unittest.main()
