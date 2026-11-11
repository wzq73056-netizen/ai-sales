import sys, time
sys.path.insert(0, r'E:\ai\scripts')
import importlib.util
spec = importlib.util.spec_from_file_location('wxw', r'E:\ai\scripts\wx_watch.py')
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

root = m.Path(r'C:\Users\11\xwechat_files\wxid_15d0megq0l7822_302a\db_storage')
print('取密钥…')
t = time.time()
keys = m.extract_keys(root, 120)
print(f'  密钥 {len(keys)} 个，{time.time()-t:.2f}s')

for i in range(3):
    t = time.time()
    rec = m.decrypt_into(root, m.WORK, keys, 'chat')
    unread = m.read_unread(m.WORK)
    print(f'  快检第{i+1}轮: {time.time()-t:.2f}s  库 {len(rec)}  未读 {len(unread)}')

t = time.time()
rec = m.decrypt_into(root, m.WORK, keys, 'all')
print(f'全量解密: {time.time()-t:.2f}s  库 {len(rec)}')
