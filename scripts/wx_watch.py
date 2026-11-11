#!/usr/bin/env python
"""
微信消息监听（只读）

原理：
  WeChat 收到消息会写 WAL（message_0.db-wal / session.db-wal）
  → 检测文件变化 → 用内存里取到的密钥增量解密 → 读未读会话
  → 只在有新消息时输出，其余时间只做 stat，开销极小

用法：
  python wx_watch.py --once              # 跑一轮，打印当前状态
  python wx_watch.py --interval 1.5      # 常驻监听
  python wx_watch.py --once --verbose    # 打印所有会话

状态输出：
  E:\\ai\\data\\wx_watch_status.json    最新会话 + 未读
  E:\\ai\\data\\wx_messages.jsonl       新消息流水（只追加）

安全：
  - 只读源数据库（read_only），不改微信任何文件
  - 密钥只存在进程内存，不落盘
"""
from __future__ import annotations

import argparse
import hashlib
import json
import subprocess as sp
import urllib.request
import os
import shutil
import subprocess
import sys
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path

SKILL_SCRIPTS = Path(r'E:\ai\vendor\yichen-skills\yichen-wechat-local-vault\scripts')
sys.path.insert(0, str(SKILL_SCRIPTS))

import windows_live as wl          # noqa: E402
import snapshot_reader as sr       # noqa: E402
from windows_crypto import decrypt_pages        # noqa: E402
from windows_wal import read_snapshot           # noqa: E402
from windows_native import process_ids          # noqa: E402

DEFAULT_ROOT = Path(r'C:\Users\11\xwechat_files\wxid_15d0megq0l7822_302a\db_storage')
WORK = Path(r'E:\ai\data\wx_live')
STATUS = Path(r'E:\ai\data\wx_watch_status.json')
MESSAGES = Path(r'E:\ai\data\wx_messages.jsonl')
VAULT_CLI = SKILL_SCRIPTS / 'vault_cli.py'
HANDLED = Path(r'E:\ai\data\wx_handled.json')
HEARTBEAT = Path(r'E:\ai\data\wx_heartbeat.txt')
KEY_FAIL_FLAG = Path(r'E:\ai\data\wx_key_failed.flag')


def read_handled() -> dict:
    try:
        return json.loads(HANDLED.read_text(encoding='utf-8'))
    except Exception:  # noqa: BLE001
        return {}


def write_handled(data: dict) -> None:
    HANDLED.parent.mkdir(parents=True, exist_ok=True)
    HANDLED.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding='utf-8')


def message_key(item: dict) -> str:
    """消息级指纹：时间 + 内容哈希。
    不能用会话 ID 做主键——微信同一条会话会以不同的 chat_id 出现（md5/wxid 两种），
    按会话记就会「同一条消息回很多次」，用户看到的是一次请求很久后被反复回复。"""
    raw = f"{item.get('last_time')}|{(item.get('last_content') or '').strip()[:80]}"
    return hashlib.sha1(raw.encode('utf-8')).hexdigest()


# 已经回过的消息指纹（全局，不按会话），最多保留 800 条
REPLIED = Path(r'E:\ai\data\wx_replied_msgs.json')

# 超过这个分钟数的消息不再回复（避免监听重启后补发一堆过期回复）
MAX_MSG_AGE_MIN = float(os.environ.get('WX_MAX_MSG_AGE_MIN', '10'))

# 同一个会话两次回复之间的最小间隔（分钟）
REPLY_COOLDOWN_MIN = float(os.environ.get('WX_REPLY_COOLDOWN_MIN', '5'))


def read_replied() -> dict:
    try:
        return json.loads(REPLIED.read_text(encoding='utf-8'))
    except Exception:  # noqa: BLE001
        return {}


def mark_replied(key: str, chat_id: str) -> None:
    data = read_replied()
    data[key] = {'chat_id': chat_id, 'at': datetime.now().strftime('%Y-%m-%d %H:%M:%S')}

    # 只留最近 800 条，避免文件无限增长
    if len(data) > 800:
        items = sorted(data.items(), key=lambda kv: kv[1].get('at') or '')
        data = dict(items[-800:])

    REPLIED.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding='utf-8')


def message_age_minutes(last_time: str):
    """消息时间到现在的分钟数；解析不了返回 None"""
    try:
        when = datetime.strptime(str(last_time)[:19], '%Y-%m-%d %H:%M:%S')
    except Exception:  # noqa: BLE001
        return None

    return (datetime.now() - when).total_seconds() / 60


def chat_replied_recently(chat_id: str) -> bool:
    """这个会话最近 REPLY_COOLDOWN_MIN 分钟内回过没有"""
    cutoff = datetime.now().timestamp() - REPLY_COOLDOWN_MIN * 60

    for item in read_replied().values():
        if item.get('chat_id') != chat_id:
            continue

        try:
            at = datetime.strptime(str(item.get('at'))[:19], '%Y-%m-%d %H:%M:%S').timestamp()
        except Exception:  # noqa: BLE001
            continue

        if at >= cutoff:
            return True

    return False

# 微信消息类型（非文本一律转人工，避免 AI 瞎猜）
NON_TEXT_TYPES = {
    3: '图片', 34: '语音', 43: '视频', 47: '表情', 42: '名片',
    48: '位置', 49: '文件/链接', 50: '语音通话', 10000: '系统消息', 10002: '系统消息',
}

# 系统号/公众号，不算待回复
SERVICE_NAMES = {'腾讯新闻', '微信团队', '服务通知', '朋友推荐消息', '语音记事本', '文件传输助手'}


API = os.environ.get('WX_API', 'http://127.0.0.1:8888')
SEND_PS1 = Path(r'E:\ai\scripts\wx_send.ps1')


def api_call(path: str, payload: dict, token: str = '', method: str = 'POST') -> dict:
    headers = {'Content-Type': 'application/json; charset=utf-8'}
    if token:
        headers['x-token'] = token

    if method == 'GET':
        request = urllib.request.Request(f'{API}{path}', headers=headers, method='GET')
    else:
        data = json.dumps(payload, ensure_ascii=False).encode('utf-8')
        request = urllib.request.Request(f'{API}{path}', data=data, method='POST', headers=headers)
    with urllib.request.urlopen(request, timeout=90) as response:
        return json.loads(response.read().decode('utf-8'))


def api_login() -> str:
    result = api_call('/base/login', {'username': 'admin', 'password': '123456', 'captcha': '', 'captchaId': ''})
    if result.get('code') != 0:
        raise RuntimeError(f'登录失败: {result.get("msg")}')
    return result['data']['token']


def sync_customers(token: str, items: list) -> None:
    try:
        api_call('/lead/syncWxCustomer', {'Items': items}, token)
    except Exception as exc:  # noqa: BLE001
        log(f'  客户价值同步失败: {exc}')


def claim_wx_add_job(token: str):
    """向后台领取一条加好友任务（后台进程拿不到微信桌面，改由监听器执行）"""
    try:
        result = api_call('/lead/claimWxAddJob', {}, token, method='GET')

        if result.get('code') != 0:
            return None

        return (result.get('data') or {}).get('job')
    except Exception:  # noqa: BLE001
        return None


def finish_wx_add_job(token: str, job_id, state: str, detail: str = '') -> None:
    try:
        api_call('/lead/finishWxAddJob', {'Id': job_id, 'State': state, 'Detail': detail[:280]}, token)
    except Exception:  # noqa: BLE001
        pass


def run_add_friend(contact: str, remark: str):
    """调用 PowerShell 执行一次加好友，返回 (state, detail)"""
    import subprocess

    args = [
        'powershell', '-NoProfile', '-ExecutionPolicy', 'Bypass',
        '-File', r'E:\ai\scripts\wx_add_friend.ps1',
        '-Contact', contact, '-Remark', remark, '-Send',
    ]

    try:
        proc = subprocess.run(
            args, capture_output=True, timeout=180,
            encoding='utf-8', errors='replace',
            creationflags=0x08000000,
        )
        text = (proc.stdout or '').strip()
        err_text = (proc.stderr or '').strip()

        if not text and err_text:
            text = 'STDERR: ' + err_text

        state = 'unknown'

        if '"state":"' in text:
            state = text.split('"state":"', 1)[1].split('"', 1)[0]

        return state, text
    except Exception as exc:  # noqa: BLE001
        return 'error', str(exc)


def claimed_chats(token: str) -> set:
    try:
        # 任何人工介入过（接管/完成/忽略）的会话都不再由 AI 自动回复，需手动恢复
        result = api_call('/lead/getAiPausedChats', {}, token, method='GET')
        if result.get('code') != 0:
            return set()
        return {chat for chat in ((result.get('data') or {}).get('chats') or []) if chat}
    except Exception:  # noqa: BLE001
        return set()


def check_intent(token: str, text: str, history: str) -> dict:
    result = api_call('/lead/checkWxIntent', {'Text': text, 'History': history}, token)
    if result.get('code') != 0:
        return {'NeedHuman': False}
    return result.get('data') or {'NeedHuman': False}


def create_handoff(token: str, chat_id: str, name: str, intent: dict, last_msg: str, draft: str) -> dict:
    payload = {
        'ChatId': chat_id,
        'NickName': name,
        'Category': intent.get('Category', ''),
        'Reason': intent.get('Reason', ''),
        'LastMsg': last_msg,
        'DraftReply': draft,
        'Urgency': intent.get('Urgency', 1),
        'Status': 0,
    }
    result = api_call('/lead/createWxHandoff', payload, token)
    return result.get('data') or {}


def gen_reply(token: str, name: str, last_message: str, history: str = '') -> str:
    result = api_call('/lead/genDmReply',
                      {'Nick': name, 'LastMessage': last_message, 'History': history, 'Channel': 'wechat'},
                      token)
    if result.get('code') != 0:
        raise RuntimeError(f'生成失败: {result.get("msg")}')
    return result['data']['Reply']


def current_chat_id(name: str) -> str:
    """按昵称查当前会话 ID（微信的 chat_id 每轮解密都会变，不能缓存）"""
    try:
        for chat in list_chats():
            if (chat.get('display_name') or '') == name:
                return chat.get('chat_id') or ''
    except Exception:  # noqa: BLE001
        return ''

    return ''


def verify_sent(chat_id: str, message: str, tries: int = 3, wait: float = 2.0, name: str = '') -> bool:
    """发完再从数据库确认一次：最后一条是不是我们刚发的那句。

    之前只看 wx_send.ps1 返回的 ok，但它按回车就算成功——
    搜不到联系人 / 没选中会话时消息其实还留在输入框，
    于是出现了「日志说发了，聊天里看不到」。
    """
    target = (message or '').strip()[:20]

    if not target:
        return False

    for _ in range(tries):
        try:
            keys = extract_keys(DEFAULT_ROOT, 30)
            decrypt_into(DEFAULT_ROOT, WORK, keys, 'all')

            # 会话 ID 会变，每次重新按昵称查
            target_id = current_chat_id(name) if name else chat_id
            msgs = chat_history(target_id or chat_id, limit=3) if (target_id or chat_id) else []
        except Exception as exc:  # noqa: BLE001
            log(f'  发送校验读库失败: {exc}')
            time.sleep(wait)
            continue

        if msgs:
            last = (msgs[-1].get('content') or '').strip()[:20]
            if last == target:
                return True

        time.sleep(wait)

    return False


def send_wechat(contact: str, message: str, dry_run: bool) -> dict:
    argv = ['powershell', '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', str(SEND_PS1),
            '-Contact', contact, '-Message', message]
    if dry_run:
        argv.append('-DryRun')

    # 不弹黑框：CREATE_NO_WINDOW
    result = sp.run(argv, capture_output=True, timeout=120, creationflags=0x08000000)
    out = result.stdout.decode('utf-8', 'replace').strip()
    try:
        return json.loads(out.splitlines()[-1])
    except Exception:  # noqa: BLE001
        return {'ok': False, 'raw': out[:200], 'err': result.stderr.decode('utf-8', 'replace')[:200]}


AUDIO_API = os.environ.get('WX_AUDIO_API', 'http://127.0.0.1:8898')


def add_followup(token: str, nick: str, chat_id: str, kind: str, content: str, media_url: str = '') -> None:
    """记一条客户跟进"""
    try:
        api_call('/lead/addWxFollowup', {
            'NickName': nick,
            'ChatId': chat_id,
            'Kind': kind,
            'Content': content[:900],
            'Operator': 'AI',
        }, token)
    except Exception:  # noqa: BLE001
        pass


MEDIA_DIR = Path(r'E:\ai\data\media')


def save_voice_wav(wav: bytes, timestamp) -> str:
    """语音 WAV 落到 media 目录，返回给前端播放的相对地址"""
    try:
        MEDIA_DIR.mkdir(parents=True, exist_ok=True)
        name = f'voice-{int(timestamp)}.wav'
        (MEDIA_DIR / name).write_bytes(wav)
        return f'/media/{name}'
    except Exception as exc:  # noqa: BLE001
        log(f'  保存语音失败: {exc}')
        return ''


def transcribe_voice(timestamp) -> tuple[str, str]:
    """按消息时间戳找到语音数据，转 WAV 落盘 + 识别成文字，返回（文字, 音频地址）"""
    import sqlite3

    db = WORK / 'message/media_0.db'
    if not db.exists() or timestamp is None:
        return '', ''

    try:
        con = sqlite3.connect(f'file:{db}?immutable=1', uri=True)
        row = con.execute(
            'SELECT voice_data FROM VoiceInfo WHERE create_time BETWEEN ? AND ? ORDER BY create_time DESC LIMIT 1',
            (int(timestamp) - 2, int(timestamp) + 2)
        ).fetchone()
        con.close()
    except Exception as exc:  # noqa: BLE001
        log(f'  读取语音数据失败: {exc}')
        return '', ''

    if not row or not row[0]:
        return '', ''

    silk = row[0]

    try:
        request = urllib.request.Request(f'{AUDIO_API}/convert', data=silk, method='POST',
                                         headers={'Content-Type': 'application/octet-stream'})
        with urllib.request.urlopen(request, timeout=60) as response:
            wav = response.read()

        request = urllib.request.Request(f'{AUDIO_API}/asr', data=wav, method='POST',
                                         headers={'Content-Type': 'application/octet-stream'})
        with urllib.request.urlopen(request, timeout=120) as response:
            payload = json.loads(response.read().decode('utf-8'))

        audio_url = save_voice_wav(wav, timestamp)

        return (payload.get('text') or '').strip(), audio_url
    except Exception as exc:  # noqa: BLE001
        log(f'  语音转写失败: {exc}')
        return '', ''


# 终端/管道默认可能是 GBK，日志里一旦出现 emoji 就会崩溃，统一改成 UTF-8
try:
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')
    sys.stderr.reconfigure(encoding='utf-8', errors='replace')
except Exception:  # noqa: BLE001
    pass


def log(msg: str) -> None:
    print(f'[{datetime.now().strftime("%H:%M:%S")}] {msg}', flush=True)


def source_files(root: Path, scope: str = 'all') -> list[Path]:
    return wl.selected_files(root, scope=scope)


def sidecar_mtimes(files: list[Path]) -> str:
    """用 WAL/shm 的 mtime+size 拼一个指纹，变化说明有新写入"""
    parts = []

    for path in files:
        for suffix in ('', '-wal', '-shm'):
            target = Path(str(path) + suffix)
            try:
                stat = target.stat()
                parts.append(f'{target.name}:{stat.st_mtime_ns}:{stat.st_size}')
            except OSError:
                parts.append(f'{target.name}:0:0')

    return '|'.join(parts)


def extract_keys(root: Path, seconds: int = 120) -> dict:
    files = source_files(root)
    pids = process_ids()

    return wl.extract_keys(files, pids, seconds)[0]


def shred_plaintext(work: Path) -> None:
    """把解密出来的明文库文件覆写后删除，避免长期落盘"""
    if not work.exists():
        return

    for path in work.rglob('*'):
        if not path.is_file():
            continue

        try:
            size = path.stat().st_size
            if size:
                with path.open('r+b') as handle:
                    handle.write(b'\x00' * min(size, 64 * 1024))
                    handle.flush()
                    os.fsync(handle.fileno())
            path.unlink()
        except Exception:  # noqa: BLE001
            pass


def decrypt_into(root: Path, work: Path, keys: dict, scope: str = 'all') -> list[dict]:
    """把所有 chat 库解密到工作目录（覆盖），返回库信息"""
    records = []

    for path in source_files(root, scope):
        data = read_snapshot(path, max_bytes=512 * 1024 * 1024)
        key = keys.get(data[:16].hex())

        if key is None:
            raise RuntimeError(f'database salt changed: {path.name}')

        plain = decrypt_pages(data, bytes.fromhex(key))
        relative = path.relative_to(root)
        destination = work / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(plain)
        records.append({'database': relative.as_posix(), 'bytes': len(plain)})

    # 写一个最小 manifest，供 snapshot_reader 校验
    manifest = {
        'snapshot_id': str(uuid.uuid4()),
        'profile': 'windows-chat-capture-v1',
        'captured_at': datetime.now(timezone.utc).isoformat(),
        'capture_started_at': datetime.now(timezone.utc).isoformat(),
        'capture_consistency': 'incremental live decrypt',
        'source_id': wl.source_id(root),
        'databases': [
            {'database': r['database'], 'pages': r['bytes'] // 4096, 'authenticated': True,
             'sqlite_integrity': 'skipped', 'encrypted_snapshot_sha256': '', 'plaintext_sha256': ''}
            for r in records
        ],
    }
    # 校验器要求“checkpointed standalone snapshot”，先把 sidecar 清掉
    for sidecar in list(work.rglob('*-wal')) + list(work.rglob('*-shm')) + list(work.rglob('*-journal')):
        if sidecar.is_file():
            sidecar.unlink()

    (work / 'snapshot-manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2), encoding='utf-8')

    return records


def read_unread(work: Path) -> dict[str, int]:
    """从 session.db 读未读数：username -> unread_count"""
    import sqlite3

    db = work / 'session/session.db'
    if not db.exists():
        return {}

    out = {}

    try:
        con = sqlite3.connect(f'file:{db}?immutable=1', uri=True)
        for username, count in con.execute('SELECT username, unread_count FROM SessionTable'):
            if count and int(count) > 0:
                out[username] = int(count)
        con.close()
    except Exception as exc:  # noqa: BLE001
        log(f'读未读失败: {exc}')

    return out


def clear_sidecars() -> None:
    for sidecar in list(WORK.rglob('*-wal')) + list(WORK.rglob('*-shm')) + list(WORK.rglob('*-journal')):
        try:
            if sidecar.is_file():
                sidecar.unlink()
        except OSError:
            pass


def cli(*args: str) -> dict:
    clear_sidecars()
    result = subprocess.run(
        [sys.executable, str(VAULT_CLI), 'snapshot', '--snapshot', str(WORK), *args],
        capture_output=True, timeout=60,
    )

    if result.returncode != 0:
        raise RuntimeError(result.stderr.decode('utf-8', 'replace')[:200])

    return json.loads(result.stdout.decode('utf-8', 'replace'))


def list_chats() -> list[dict]:
    return cli('chats').get('chats', [])


def chat_history(chat_id: str, limit: int = 5) -> list[dict]:
    return cli('history', chat_id, '--limit', str(limit)).get('messages', [])


def snapshot_state(root: Path, keys: dict, verbose: bool = False, full: bool = True, reply: bool = False, send: bool = False, token: str = '') -> dict:
    # 没传 token 就自己登一次，避免回复逻辑被空 token 短路
    if not token:
        try:
            token = api_login()
        except Exception:  # noqa: BLE001
            token = ''

    records = decrypt_into(root, WORK, keys, 'all' if full else 'chat')
    unread = read_unread(WORK)

    if not full:
        previous = {}
        try:
            previous = json.loads(STATUS.read_text(encoding='utf-8'))
        except Exception:  # noqa: BLE001
            pass

        state = {
            'updated_at': datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
            'databases': len(records),
            'chats': previous.get('chats', 0),
            'unread_sessions': len(unread),
            # 快路径没有读内容，保留上一轮读到的待回复，避免被清空
            'pending_reply': previous.get('pending_reply', []),
            'sessions': previous.get('sessions', []),
        }
        STATUS.parent.mkdir(parents=True, exist_ok=True)
        STATUS.write_text(json.dumps(state, ensure_ascii=False, indent=2), encoding='utf-8')
        return state

    chats = list_chats()

    summary = []
    new_messages = []
    snapshots = []

    for chat in chats:
        name = chat.get('display_name') or ''
        cid = chat.get('chat_id') or ''

        # 系统号/公众号直接跳过，避免无谓的历史读取
        if name in SERVICE_NAMES:
            continue

        try:
            messages = chat_history(cid, limit=12)
        except Exception:  # noqa: BLE001
            messages = []

        if not messages:
            continue

        # 去重统一走消息级指纹（见下面 message_key / read_replied），这里不再按会话比对

        last = messages[-1]
        me = '哈拉少'
        is_mine = last.get('sender') == me

        # 把最近对话拼成上下文（不含最后一条，最后一条单独作为 LastMessage）
        history_lines = []
        for msg in messages[:-1]:
            who = '我' if msg.get('sender') == me else '对方'
            content = (msg.get('content') or '').strip()
            if content:
                history_lines.append(f"{who}: {content}")
        history_text = '\n'.join(history_lines[-10:])

        item = {
            'chat_id': cid,
            'name': name,
            'last_time': last.get('time'),
            'last_sender': last.get('sender'),
            'last_content': (last.get('content') or '')[:100],
            'from_me': is_mine,
            'msg_type': last.get('type'),
            'timestamp': last.get('timestamp'),
            'messages': len(messages),
            'history': history_text,
        }
        summary.append(item)

        snapshots.append({
            'ChatId': cid,
            'NickName': name,
            'LastMsgAt': last.get('time') or '',
            'LastFromMe': is_mine,
            'MyCount': sum(1 for m in messages if m.get('sender') == me),
            'TheirCount': sum(1 for m in messages if m.get('sender') != me),
            'Texts': [str(m.get('content') or '') for m in messages],
        })

        # 过滤规则（按顺序）：
        #   1. 自己发的不回
        #   2. 这条消息已经回过（消息级指纹，不受 chat_id 漂移影响）
        #   3. 消息太旧（监听重启/卡顿后不补发过期回复）
        #   4. 这个会话刚回过（冷却期）
        if is_mine:
            continue

        fingerprint = message_key(item)

        if fingerprint in read_replied():
            continue

        age = message_age_minutes(item.get('last_time'))

        if age is not None and age > MAX_MSG_AGE_MIN:
            log(f'  跳过过期消息 {name}（{age:.0f} 分钟前）：{item["last_content"][:30]}')
            continue

        if chat_replied_recently(cid):
            log(f'  会话冷却中，跳过 {name}')
            continue

        new_messages.append(item)

    state = {
        'updated_at': datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
        'databases': len(records),
        'chats': len(chats),
        'unread_sessions': len(unread),
        'pending_reply': new_messages,
        'sessions': summary,
    }

    if snapshots and token:
        sync_customers(token, snapshots)

    # 生成 AI 草稿 / 发送
    if (reply or send) and new_messages:
        claimed = claimed_chats(token) if token else set()

        for item in new_messages:
            if not token:
                break

            try:
                # 已被人工接管的会话，AI 不再自动回
                if item['chat_id'] in claimed:
                    log(f'  {item["name"]} 已人工接管，跳过')
                    continue

                # 语音消息：先尝试转写，成功当文本处理
                msg_type = item.get('msg_type')

                if msg_type == 34:
                    log(f'  语音消息，转写中…')
                    transcript, audio_url = transcribe_voice(item.get('timestamp'))

                    if transcript:
                        log(f'  转写成功：{transcript[:40]}')
                        item['transcript'] = transcript
                        item['last_content'] = transcript
                        item['audio_url'] = audio_url
                        add_followup(token, item['name'], item['chat_id'], '接收',
                                     f'[语音] {transcript}', audio_url)
                    else:
                        log('  转写失败 → 转人工')
                        create_handoff(token, item['chat_id'], item['name'],
                                       {'Category': '语音转写失败', 'Reason': '语音转文字失败，需人工收听', 'Urgency': 2},
                                       '[语音] 转写失败', '')
                        item['skipped_reply'] = True
                        continue

                # 其余非文本（图片/视频/文件…）直接转人工，不让 AI 猜
                elif msg_type and msg_type != 1:
                    kind = NON_TEXT_TYPES.get(msg_type, f'类型{msg_type}')
                    log(f'  转人工 {item["name"]}（非文本：{kind}）')

                    # AI 看不到图片/文件里的内容（微信是加密存储），只做一件事：
                    # 不猜、不回，交给人工；同时把最近的文字上下文一并带过去
                    history = (item.get('history') or '').strip()
                    context = f'；最近对话：{history[-160:]}' if history else ''
                    reason = f'对方发了{kind}，AI 看不到里面内容，需人工在微信里查看{context}'

                    draft = ''
                    if kind == '图片':
                        draft = '收到，我看一下图片，稍后回复您。'

                    create_handoff(token, item['chat_id'], item['name'],
                                   {'Category': f'{kind}消息', 'Reason': reason, 'Urgency': 2},
                                   f'[{kind}] ' + (item['last_content'] or '')[:200], draft)
                    item['skipped_reply'] = True
                    continue

                # 先判断要不要人工介入（打款/合同/看现场等）
                intent = check_intent(token, item['last_content'], item.get('history', ''))
                item['intent'] = intent

                if intent.get('NeedHuman'):
                    log(f'  转人工 {item["name"]}（{intent.get("Category")}）：{intent.get("Reason")}')

                    handoff = create_handoff(token, item['chat_id'], item['name'], intent,
                                             item['last_content'], '')
                    item['handoff_id'] = handoff.get('Id')
                    item['skipped_reply'] = True
                    continue

                draft = gen_reply(token, item['name'], item['last_content'], item.get('history', ''))
                item['draft_reply'] = draft
                log(f'  草稿 {item["name"]}: {draft}')

                if send:
                    # 先看最后一条是不是已经就是这句（避免重复发送）
                    try:
                        _keys = extract_keys(DEFAULT_ROOT, 30)
                        decrypt_into(DEFAULT_ROOT, WORK, _keys, 'all')
                        _msgs = chat_history(current_chat_id(item['name']) or item['chat_id'], limit=2)
                    except Exception:  # noqa: BLE001
                        _msgs = []

                    if _msgs and (str(_msgs[-1].get('content') or '').strip()[:20] == draft.strip()[:20]):
                        log(f'  最近一条已经是这句，跳过发送 {item["name"]}')
                        continue

                    result = send_wechat(item['name'], draft, dry_run=not send)
                    item['send_result'] = result
                    log(f'  发送结果: {result}')

                    sent_ok = bool(result.get('ok'))
                    verified = sent_ok and verify_sent(item['chat_id'], draft, name=item['name'])

                    if verified:
                        mark_replied(message_key(item), item['chat_id'])

                        add_followup(token, item['name'], item['chat_id'], '自动回复', f'AI 回复：{draft}')
                    else:
                        log(f'  发送未确认 {item["name"]}：{result}')

                        # 没确认发出去就不要再记账本，也不要静默丢掉，转人工让人看一眼
                        create_handoff(token, item['chat_id'], item['name'],
                                       {'Category': '回复未发出',
                                        'Reason': f'自动回复没有确认送达（{result.get("reason") or "数据库里没找到这条消息"}），需要人工补发',
                                        'Urgency': 2},
                                       f'本应回复：{draft}', draft)
            except Exception as exc:  # noqa: BLE001
                item['draft_error'] = str(exc)[:200]
                log(f'  生成失败 {item["name"]}: {exc}')

    STATUS.parent.mkdir(parents=True, exist_ok=True)
    STATUS.write_text(json.dumps(state, ensure_ascii=False, indent=2), encoding='utf-8')

    return state


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--db-root', default=str(DEFAULT_ROOT))
    parser.add_argument('--interval', type=float, default=1.5)
    parser.add_argument('--once', action='store_true')
    parser.add_argument('--verbose', action='store_true')
    parser.add_argument('--seconds', type=int, default=120, help='密钥扫描上限')
    parser.add_argument('--reply', action='store_true', help='发现待回复时用 AI 生成草稿')
    parser.add_argument('--send', action='store_true', help='生成后直接发送（需 --reply）')
    args = parser.parse_args()

    root = Path(args.db_root)
    if not root.is_dir():
        log(f'数据目录不存在: {root}')
        return 2

    WORK.mkdir(parents=True, exist_ok=True)

    log('提取密钥…')
    started = time.time()
    keys = extract_keys(root, args.seconds)
    log(f'密钥就绪（{len(keys)} 个，{time.time() - started:.1f}s）')

    fingerprint = sidecar_mtimes(source_files(root))

    token = api_login() if args.reply or args.send else ''
    state = snapshot_state(root, keys, args.verbose, reply=args.reply, send=args.send, token=token)
    log(f'初始：{state["chats"]} 个会话，待回复 {len(state["pending_reply"])}')

    for item in state['pending_reply']:
        log(f'  待回复 {item["name"]}: {item["last_content"][:40]}')

    if args.once:
        print(json.dumps(state, ensure_ascii=False, indent=2))
        return 0

    try:
        HEARTBEAT.write_text(datetime.now().strftime('%Y-%m-%d %H:%M:%S'), encoding='utf-8')
    except Exception:  # noqa: BLE001
        pass

    log(f'开始监听，间隔 {args.interval}s')

    last_beat = 0.0
    last_shred = 0.0
    last_report = 0.0

    while True:
        # 心跳：健康检查脚本用它判断是否卡死
        if time.time() - last_beat > 30:
            try:
                HEARTBEAT.write_text(datetime.now().strftime('%Y-%m-%d %H:%M:%S'), encoding='utf-8')
            except Exception:  # noqa: BLE001
                pass
            last_beat = time.time()

        time.sleep(args.interval)

        # 自动加好友：后台进程拿不到微信桌面，这里代跑
        if token:
            job = claim_wx_add_job(token)

            if job:
                contact = job.get('Contact') or job.get('contact') or ''
                remark = job.get('Remark') or job.get('remark') or ''
                job_id = job.get('Id') or job.get('id')

                log(f"加好友任务: {contact}")
                state, detail = run_add_friend(contact, remark)
                finish_wx_add_job(token, job_id, state, detail)
                log(f"加好友结果: {state} | {detail[:160]}")

        current = sidecar_mtimes(source_files(root))

        if current == fingerprint:
            # 空闲时也定期擦除明文（新消息来时再解密）
            if time.time() - last_shred > 60:
                shred_plaintext(WORK)
                last_shred = time.time()

            continue

        fingerprint = current
        time.sleep(0.8)  # 等写入落盘

        try:
            state = snapshot_state(root, keys, args.verbose, full=True, reply=args.reply, send=args.send, token=token)
        except Exception as exc:  # noqa: BLE001
            log(f'解密失败，重新取密钥: {exc}')
            try:
                keys = extract_keys(root, args.seconds)
                state = snapshot_state(root, keys, args.verbose, full=True, reply=args.reply, send=args.send, token=token)
            except Exception as exc2:  # noqa: BLE001
                log(f'仍然失败: {exc2}')

                # 密钥可能随微信升级变了，落个标记并弹桌面通知（1 小时内只提醒一次）
                try:
                    should_notify = True
                    if KEY_FAIL_FLAG.exists():
                        age = time.time() - KEY_FAIL_FLAG.stat().st_mtime
                        should_notify = age > 3600

                    KEY_FAIL_FLAG.write_text(datetime.now().strftime('%Y-%m-%d %H:%M:%S'), encoding='utf-8')

                    if should_notify:
                        subprocess.Popen([
                            'powershell', '-NoProfile', '-WindowStyle', 'Hidden',
                            '-ExecutionPolicy', 'Bypass',
                            '-File', r'E:\ai\scripts\notify-handoff.ps1',
                            '-Title', '微信监听异常：取不到密钥',
                            '-Text', f'微信可能已升级或退出登录，自动回复已停。{str(exc2)[:60]}',
                            '-Seconds', '15'
                        ], creationflags=0x08000000)
                except Exception:  # noqa: BLE001
                    pass

                continue

        log(f'变化：{state["chats"]} 会话，待回复 {len(state["pending_reply"])}')

        if KEY_FAIL_FLAG.exists():
            try:
                KEY_FAIL_FLAG.unlink()
            except Exception:  # noqa: BLE001
                pass

        with MESSAGES.open('a', encoding='utf-8') as handle:
            for item in state['pending_reply']:
                handle.write(json.dumps({**item, 'detected_at': state['updated_at']}, ensure_ascii=False) + '\n')
                log(f'  新消息 {item["name"]}: {item["last_content"][:50]}')

        # 处理完立刻擦掉明文库，磁盘上不长期保留聊天记录
        shred_plaintext(WORK)

        fingerprint = sidecar_mtimes(source_files(root))


if __name__ == '__main__':
    raise SystemExit(main())
