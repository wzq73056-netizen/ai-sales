import sys, importlib.util
spec = importlib.util.spec_from_file_location('wxw', r'E:\ai\scripts\wx_watch.py')
m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
root = m.Path(r'C:\Users\11\xwechat_files\wxid_15d0megq0l7822_302a\db_storage')
keys = m.extract_keys(root, 120)
m.decrypt_into(root, m.WORK, keys, 'all')
chats = m.list_chats()
print('chats:', len(chats))
for c in chats:
    print('  ', c.get('display_name'), '|', c.get('chat_id'))
for c in chats:
    name = c.get('display_name')
    if name in m.SERVICE_NAMES: continue
    try:
        msgs = m.chat_history(c['chat_id'], 3)
        print(f'--- {name} 消息 {len(msgs)}')
        for x in msgs[-2:]:
            print('     ', x.get('time'), x.get('sender'), '|', (x.get('content') or '')[:40])
    except Exception as e:
        print(f'--- {name} 读取失败: {type(e).__name__}: {e}')
