import sys, importlib.util, json
spec = importlib.util.spec_from_file_location('wxw', r'E:\ai\scripts\wx_watch.py')
m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
root = m.Path(r'C:\Users\11\xwechat_files\wxid_15d0megq0l7822_302a\db_storage')
keys = m.extract_keys(root, 120)
m.decrypt_into(root, m.WORK, keys, 'all')
chats = m.list_chats()
for c in chats:
    if c.get('display_name') in m.SERVICE_NAMES: continue
    msgs = m.chat_history(c['chat_id'], 3)
    for x in msgs[-2:]:
        print('type=', x.get('type'), '| ts=', x.get('timestamp'), '|', (x.get('content') or '')[:20])
