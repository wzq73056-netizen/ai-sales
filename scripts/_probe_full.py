import importlib.util, sys, time
spec = importlib.util.spec_from_file_location('wxw', r'E:\ai\scripts\wx_watch.py')
m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
root = m.Path(r'C:\Users\11\xwechat_files\wxid_15d0megq0l7822_302a\db_storage')
print('取密钥…', flush=True)
keys = m.extract_keys(root, 120)
print('密钥 ok', flush=True)
t0 = time.time()
state = m.snapshot_state(root, keys, True, full=True, reply=True, send=False)
print('snapshot 用了', round(time.time()-t0,1), 's', flush=True)
for item in state['pending_reply']:
    print('  待回复:', item['name'], '|', item['last_content'], flush=True)
    print('    draft=', item.get('draft_reply'), 'skipped=', item.get('skipped_reply'), 'err=', item.get('draft_error'), flush=True)
