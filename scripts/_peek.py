lines = open(r'E:\ai\scripts\wx_watch.py', encoding='utf-8').read().splitlines()
for i in range(374, 392):
    print(i+1, repr(lines[i]))
