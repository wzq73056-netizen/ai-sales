from pathlib import Path
data = Path(r'E:\ai\data\voice_test.silk').read_bytes()
# 去掉前导 0x02，得到标准 SILK_V3
if data[0] == 0x02:
    data = data[1:]
print('头:', data[:10])
Path(r'E:\ai\data\voice_test_clean.silk').write_bytes(data)
print('已写标准 silk:', len(data), 'bytes')
