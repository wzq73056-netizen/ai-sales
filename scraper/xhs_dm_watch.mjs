/**
 * 小红书私信自动回复（白名单）
 *
 * 用法：
 *   node xhs_dm_watch.mjs --once            # 跑一轮看看
 *   node xhs_dm_watch.mjs --interval 60     # 常驻
 *   node xhs_dm_watch.mjs --send            # 真实发送
 *
 * 只回白名单里的人（默认只有「冬冬瓜咚」947928823），名单外一律不动。
 */
import fs from 'node:fs'
import { connect, detectBlock, login } from './xhs_lib.mjs'
import { applyStealth, humanDelay, humanType, scrollLikeHuman, takeQuota } from './humanize.mjs'

const API = process.env.API || 'http://127.0.0.1:8888'
const args = process.argv.slice(2)
const once = args.includes('--once')
const doSend = args.includes('--send')
// 轮询间隔默认 300 秒并加 ±40% 抖动，固定 60 秒节奏是最明显的脚本特征
const baseInterval = Number((args[args.indexOf('--interval') + 1] || 300)) * 1000
const jitter = (ms) => Math.round(ms * (0.8 + Math.random() * 0.4))

const LEDGER = 'E:\\ai\\data\\xhs_dm_ledger.json'

const sleep = ms => new Promise(r => setTimeout(r, ms))
const stamp = () => new Date().toLocaleTimeString('zh-CN', { hour12: false })
const log = msg => console.log(`[${stamp()}] ${msg}`)

const readJson = (file, fallback) => {
  try {
    return JSON.parse(fs.readFileSync(file, 'utf-8'))
  } catch {
    return fallback
  }
}

const writeJson = (file, data) => fs.writeFileSync(file, JSON.stringify(data, null, 2), 'utf-8')

/** 白名单从后端读（platform=xhs） */
async function loadWhitelistFromApi(token) {
  const res = await fetch(`${API}/lead/getDouyinWhitelist?platform=xhs`, { headers: { 'x-token': token } })
  const json = await res.json()
  const list = json?.data?.list || []

  return list
    .filter(item => Number(item.Enabled) === 1)
    .map(item => ({ nickname: item.Nickname, xhsId: item.DouyinId }))
}

async function genReply(token, nick, lastMessage, history) {
  const res = await fetch(`${API}/lead/genDmReply`, {
    method: 'POST',
    headers: { 'x-token': token, 'Content-Type': 'application/json' },
    body: JSON.stringify({ Nick: nick, LastMessage: lastMessage, History: history , Channel: 'xhs'})
  })
  const json = await res.json()

  if (json.code !== 0) throw new Error(json.msg)

  return json.data.Reply
}

const { browser, page } = await connect()
await applyStealth(page)

async function openChatPage() {
  await page.goto('https://www.xiaohongshu.com/chat', { waitUntil: 'domcontentloaded', timeout: 60000 })
await humanDelay('read')
  await page.waitForTimeout(6000)
}

/** 把会话列表里看到的消息上报到后台 */
async function reportConversations(list, token) {
  if (!list || !list.length || !token) return

  const items = list
    .map(row => {
      const text = String(row.text || '')
      // 第一段是昵称，剩下的当消息预览
      const rest = text.slice(row.nick.length).trim()

      return { peerId: row.nick, peerName: row.nick, text: rest, direction: 0, time: '' }
    })
    .filter(item => item.text)

  if (!items.length) return

  try {
    await fetch(`${API}/lead/recordDmMessages`, {
      method: 'POST',
      headers: { 'x-token': token, 'Content-Type': 'application/json' },
      body: JSON.stringify({ Platform: 'xhs', DeviceId: 0, Items: items })
    })
  } catch {
    // 上报失败不影响监听
  }
}

/** 读会话列表 */
async function readConversations() {
  return page.evaluate(() => {
    return [...document.querySelectorAll('.xhs-im-conv-item')].map(el => {
      const text = (el.innerText || '').replace(/\s+/g, ' ').trim()

      return { text, nick: text.split(' ')[0] || '' }
    })
  })
}

/** 读当前会话气泡 */
async function readBubbles() {
  return page.evaluate(() => {
    return [...document.querySelectorAll('.chat-item__bubble-row')].map(el => {
      const cls = String(el.className)

      return {
        fromMe: cls.includes('--right'),
        text: (el.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 300)
      }
    }).filter(b => b.text)
  })
}

/** 输入并发送 */
async function sendReply(reply) {
  const editable = page.locator('.xhs-im-input-bar-wrap [contenteditable="true"]').first()

  if (!(await editable.count())) throw new Error('找不到输入框')

  await editable.click()
  await humanDelay('op')
  await humanType(page, reply)
  await humanDelay('click')

  const typed = await page.evaluate(() => {
    const el = document.querySelector('.xhs-im-input-bar-wrap [contenteditable="true"]')

    return el ? (el.innerText || '').trim() : ''
  })

  if (!typed) throw new Error('输入失败')

  if (!doSend) {
    await page.keyboard.press('Control+A')
    await page.keyboard.press('Delete')

    return { sent: false }
  }

  // 每天私信上限（默认 30 条），超了直接不发
  if (!takeQuota('dm', 'xhs')) return { sent: false, quota: true }

  await page.keyboard.press('Enter')
  await sleep(2000)

  const ok = await page.evaluate(text => document.body.innerText.includes(text.slice(0, 12)), reply)

  return { sent: ok }
}

async function main() {
  const token = (await login(API).catch(() => '')) || ''
  const whitelist = await loadWhitelistFromApi(token)

  log(`白名单 ${whitelist.length} 人：${whitelist.map(w => w.nickname).join('、')}`)

  await openChatPage()

  const block = await detectBlock(page)
  if (block.blocked) {
    log(`页面异常：${block.reason}`)

    try {
      const { execFileSync } = await import('node:child_process')
      execFileSync('powershell', ['-NoProfile', '-WindowStyle', 'Hidden', '-ExecutionPolicy', 'Bypass',
        '-File', 'E:\\ai\\scripts\\notify-handoff.ps1',
        '-Title', '小红书私信监听异常',
        '-Text', `页面异常：${block.reason}，自动回复已停`,
        '-Seconds', '15'], { stdio: 'ignore' })
    } catch {}

    process.exit(3)
  }

  const convs = await readConversations()
  await reportConversations(convs, token)
  log(`会话 ${convs.length} 个`)

  const ledger = readJson(LEDGER, {})
  let round = 0

  do {
    round++

    for (const entry of whitelist) {
      const matched = convs.findIndex(c => c.nick.includes(entry.nickname))

      if (matched < 0) {
        log(`  跳过 ${entry.nickname}：会话列表里没有`)
        continue
      }

      const items = page.locator('.xhs-im-conv-item')
      await items.nth(matched).click()
      await sleep(2500)

      // 校验对方身份：会话标题里要能对上新昵称
      const titleOk = await page.evaluate(nick => {
        const header = document.querySelector('.xhs-im-chat-header, [class*="chat-header"], .xhs-im-conversation-header')
        const text = header ? (header.innerText || '') : document.body.innerText.slice(0, 2000)

        return text.includes(nick)
      }, entry.nickname)

      if (!titleOk) {
        log(`  跳过：会话标题对不上 ${entry.nickname}`)
        continue
      }

      const bubbles = await readBubbles()
      if (!bubbles.length) {
        log('  跳过：没有消息')
        continue
      }

      const last = bubbles[bubbles.length - 1]

      if (last.fromMe) {
        log('  跳过：最后一条是我方')
        continue
      }

      const key = last.text.slice(0, 40)

      if (ledger[entry.xhsId] === key) {
        log('  跳过：这条回过了')
        continue
      }

      const history = bubbles.slice(-8).map(b => `${b.fromMe ? '我' : '对方'}: ${b.text}`).join('\n')

      log(`  对方说：${last.text.slice(0, 40)}`)

      const reply = await genReply(token, entry.nickname, last.text, history)
      log(`  AI：${reply}`)

      const result = await sendReply(reply)

      if (result.sent) {
        ledger[entry.xhsId] = key
        writeJson(LEDGER, ledger)
        log('  已发送')
      } else {
        log('  未发送（dry-run）')
      }
    }

    try {
      fs.writeFileSync('E:\\ai\\data\\xhs_heartbeat.txt', new Date().toLocaleString('sv-SE'), 'utf-8')
    } catch {
      // 心跳失败不影响主流程
    }

    if (once) break

    await sleep(jitter(baseInterval))
    await openChatPage()
  } while (true)

  await browser.close()
}

await main()
