/**
 * 抖音私信自动回复（白名单守护进程）
 *
 * 只在名单里的人才会自动回；名单外的一律不动。
 *
 * 用法：
 *   node dy_dm_watch.mjs --once          # 跑一轮，看会做什么
 *   node dy_dm_watch.mjs --interval 60   # 常驻
 *   node dy_dm_watch.mjs --once --send   # 真的发
 *
 * 流程：
 *   打开主页 → 核对抖音号 → 点私信 → 读气泡
 *   → 最后一条是对方说的 → 查重 → AI 生成 → 输入 → 发送 → 记账
 */
import { chromium } from 'playwright-core'
import { applyStealth, humanDelay, humanType, takeQuota } from './humanize.mjs'
import fs from 'node:fs'
/** CDP 连接装饰：记录本进程开的标签，browser.close() 时一起关掉（否则每跑一次留一个标签） */
function trackBrowser(browser) {
  const ctx = browser.contexts()[0]
  const owned = []
  const newPage = ctx.newPage.bind(ctx)
  ctx.newPage = async () => {
    const page = await newPage()

    // 每个页面先注入指纹补丁（必须在 goto 之前）
    await applyStealth(page)
    owned.push(page)
    return page
  }
  const close = browser.close.bind(browser)
  browser.close = async () => {
    for (const page of owned) await page.close().catch(() => {})
    await close()
  }
  return browser
}


const API = process.env.API || 'http://127.0.0.1:8888'

const args = process.argv.slice(2)
const once = args.includes('--once')
const doSend = args.includes('--send')
// 默认 300 秒 + ±40% 抖动（固定 60 秒是最明显的脚本节奏）
const baseInterval = Number(args[args.indexOf('--interval') + 1] || 300) * 1000
const jitter = (ms) => Math.round(ms * (0.8 + Math.random() * 0.4))
const deviceArg = Number((args[args.indexOf('--device') + 1] || 0))

// 每个抖音账号独立记账，避免串号
let deviceId = deviceArg || 1
let LEDGER = `E:\\ai\\data\\dy_dm_ledger-${deviceId}.json`
let INBOX_STATE = `E:\\ai\\data\\dy_inbox_state-${deviceId}.json`

const LOGIN_FLAG = () => `E:\\ai\\data\\dy_login_failed-${deviceId}.txt`

/** 掉登录：落标记 + 弹一次桌面通知（1 小时内不重复） */
async function alertLoginFail() {
  try {
    const file = LOGIN_FLAG()
    let shouldNotify = true

    if (fs.existsSync(file)) {
      const age = (Date.now() - fs.statSync(file).mtimeMs) / 1000
      shouldNotify = age > 3600
    }

    fs.writeFileSync(file, new Date().toLocaleString('sv-SE'), 'utf-8')

    if (shouldNotify) {
      const { spawn } = await import('node:child_process')
      spawn('powershell', ['-NoProfile', '-WindowStyle', 'Hidden', '-ExecutionPolicy', 'Bypass',
        '-File', 'E:\\ai\\scripts\\notify-handoff.ps1',
        '-Title', '抖音掉登录了',
        '-Text', '抖音私信监听无法工作，请在浏览器里重新扫码登录',
        '-Seconds', '15'], { stdio: 'ignore', detached: true, creationflags: 0x08000000 })
    }
  } catch {
    // 通知失败不影响主流程
  }
}

function clearLoginFlag() {
  try {
    if (fs.existsSync(LOGIN_FLAG())) fs.unlinkSync(LOGIN_FLAG())
  } catch {}
}

const sleep = ms => new Promise(r => setTimeout(r, ms))
const stamp = () => new Date().toLocaleTimeString('zh-CN', { hour12: false })

const log = msg => console.log(`[${stamp()}] ${msg}`)

function readJson(file, fallback) {
  try {
    return JSON.parse(fs.readFileSync(file, 'utf-8'))
  } catch {
    return fallback
  }
}

function writeJson(file, data) {
  fs.writeFileSync(file, JSON.stringify(data, null, 2), 'utf-8')
}

async function login() {
  const res = await fetch(`${API}/base/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: '123456', captcha: '', captchaId: '' })
  })
  const json = await res.json()

  return json.code === 0 ? json.data.token : ''
}

async function apiPost(token, path, payload) {
  const res = await fetch(`${API}${path}`, {
    method: 'POST',
    headers: { 'x-token': token, 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })

  return res.json()
}

async function checkIntent(token, text, history) {
  try {
    const json = await apiPost(token, '/lead/checkWxIntent', { Text: text, History: history })

    return json?.data || { NeedHuman: false }
  } catch {
    return { NeedHuman: false }
  }
}

async function addFollowup(token, nick, chatId, kind, content) {
  try {
    await apiPost(token, '/lead/addWxFollowup', {
      NickName: nick,
      ChatId: chatId,
      Kind: kind,
      Content: String(content).slice(0, 900),
      Operator: 'AI'
    })
  } catch {
    // 记录失败不影响发送
  }
}

async function createHandoff(token, payload) {
  try {
    await apiPost(token, '/lead/createWxHandoff', payload)
  } catch (e) {
    log(`  建单失败: ${String(e).slice(0, 60)}`)
  }
}

/** 从文本里提取手机号或微信号 */
function extractContact(text) {
  const src = String(text || '')

  const phone = src.match(/1[3-9]\d{9}/)
  if (phone) return phone[0]

  const wx = src.match(/(?:vx|wx|微信|V信)[:：]?\s*([A-Za-z][-_A-Za-z0-9]{5,19})/i)
  if (wx) return wx[1]

  return ''
}

async function genReply(token, nick, lastMessage, history) {
  const res = await fetch(`${API}/lead/genDmReply`, {
    method: 'POST',
    headers: { 'x-token': token, 'Content-Type': 'application/json' },
    body: JSON.stringify({ Nick: nick, LastMessage: lastMessage, History: history })
  })
  const json = await res.json()

  if (json.code !== 0) throw new Error(json.msg)

  return json.data.Reply
}

async function loadDevice(token) {
  const res = await fetch(`${API}/lead/getDouyinDevices`, { headers: { 'x-token': token } })
  const json = await res.json()
  const list = json?.data?.list || []
  const device = list.find(item => Number(item.Id) === deviceId) || list[0]

  if (!device) throw new Error('没有可用的抖音账号，请先在「账号管理」里添加')

  deviceId = Number(device.Id)
  LEDGER = `E:\\ai\\data\\dy_dm_ledger-${deviceId}.json`
  INBOX_STATE = `E:\\ai\\data\\dy_inbox_state-${deviceId}.json`

  return device
}

async function loadWhitelist(token) {
  const res = await fetch(`${API}/lead/getDouyinWhitelist?deviceId=${deviceId}`, { headers: { 'x-token': token } })
  const json = await res.json()
  const list = json?.data?.list || []

  return list.filter(item => Number(item.Enabled) === 1 && item.ProfileUrl)
}

async function touchDevice(token) {
  try {
    await apiPost(token, '/lead/touchDouyinDevice', { Id: deviceId })
  } catch {
    // 心跳失败不影响主流程
  }
}

let browser = null
let page = null

async function connect(device) {
  const endpoint = process.env.CDP || `http://127.0.0.1:${device.CdpPort || 9333}`
  browser = trackBrowser(await chromium.connectOverCDP(endpoint))
  const ctx = browser.contexts()[0]
  page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
}

async function readBubbles() {
  return page.evaluate(() => {
    const out = []

    let nodes = []
    for (const sel of [
      'div[class*="messageMessageBoxcontentBox"]',
      '[data-e2e*="message-content"]',
      'div[class*="messageContent"]',
      'div[class*="msgContent"]'
    ]) {
      nodes = [...document.querySelectorAll(sel)]
      if (nodes.length) break
    }

    for (const el of nodes) {
      const r = el.getBoundingClientRect()
      if (!r.width || !r.height) continue

      const text = (el.innerText || '').replace(/\s+/g, ' ').replace(/\s*已读$/, '').trim()
      const hasMedia = !!el.querySelector('img, audio, video, [class*="imageMsg"], [class*="voiceMsg"]')

      if (!text && !hasMedia) continue

      out.push({
        fromMe: /isFromMe/.test((el.className || '').toString()),
        text: text.slice(0, 300),
        hasMedia,
        top: r.top
      })
    }

    return out.sort((a, b) => a.top - b.top)
  })
}

async function sendReply(reply) {
  // 每天私信有上限（默认 30 条），超了直接不发
  if (doSend && !takeQuota('dm', 'douyin')) return { sent: false, quota: true }
  const input = page.locator('div[contenteditable="true"], textarea').first()
  if (!(await input.count())) throw new Error('没有输入框')

  await input.click()
  await humanDelay('click')
  // 逐字输入（30–80ms/字），比一次性 fill 更接近真人
  await humanType(page, reply.replace(/\u200b/g, ''))
  await humanDelay('op')

  const typed = await page.evaluate(() => {
    const el = document.querySelector('div[contenteditable="true"], textarea')
    return el ? (el.innerText || el.value || '').replace(/\u200b/g, '').trim() : ''
  })

  if (!typed.includes(reply.slice(0, 8))) throw new Error('输入校验失败')

  if (!doSend) {
    await page.keyboard.press('Control+A')
    await page.keyboard.press('Delete')

    return { sent: false }
  }

  const pos = await page.evaluate(() => {
    const action = document.querySelector('[class*="inputAction"]')
    if (!action) return null

    const nodes = [...action.querySelectorAll('*')]
      .map(el => el.getBoundingClientRect())
      .filter(r => r.width >= 28 && r.width <= 40 && r.height >= 28 && r.height <= 40)

    if (!nodes.length) {
      const a = action.getBoundingClientRect()
      return { x: a.right - 16, y: a.top + a.height / 2 }
    }

    nodes.sort((a, b) => b.right - a.right)
    const r = nodes[0]
    return { x: r.left + r.width / 2, y: r.top + r.height / 2 }
  })

  if (!pos) throw new Error('没找到发送按钮')

  await page.mouse.click(pos.x, pos.y)
  await sleep(2200)

  const ok = await page.evaluate(m => document.body.innerText.includes(m), reply)

  return { sent: ok }
}

/** 打开抖音收件箱（不访问任何人的主页） */
/** 把收件箱里看到的会话消息上报到后台（用于收件箱聚合和效果统计） */
async function reportInbox(rows) {
  if (!rows || !rows.length || !token) return

  const items = rows.map(row => ({
    peerId: row.nick,
    peerName: row.nick,
    text: row.preview || '',
    direction: 0,
    time: row.time || ''
  })).filter(item => item.text)

  if (!items.length) return

  try {
    await fetch(`${API}/lead/recordDmMessages`, {
      method: 'POST',
      headers: { 'x-token': token, 'Content-Type': 'application/json' },
      body: JSON.stringify({ Platform: 'douyin', DeviceId: deviceId, Items: items })
    })
  } catch {
    // 上报失败不影响监听
  }
}

async function ensureInbox() {
  const rows = await readInbox()

  await reportInbox(rows)

  if (rows.length) {
    const bubbles = await page.evaluate(() => document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]').length)

    if (bubbles > 0) {
      const back = await page.evaluate(() => {
        const el = document.querySelector('[class*="StackLayoutStackTitleBarleftArea"]')
        if (!el) return null
        const r = el.getBoundingClientRect()
        return { x: r.left + Math.min(15, r.width / 2), y: r.top + r.height / 2 }
      })

      if (back) {
        await page.mouse.click(back.x, back.y)
        await sleep(1800)
      }
    }

    return true
  }

  await page.goto('https://www.douyin.com/jingxuan', { waitUntil: 'domcontentloaded', timeout: 60000 })
  await sleep(6000)
  await page.keyboard.press('Escape').catch(() => {})

  const entry = page.locator('[data-e2e="im-entry"]').first()

  if (!(await entry.count())) {
    // 没有私信入口 → 多半是掉登录了
    const loggedOut = await page.evaluate(() => document.body.innerText.includes('登录'))
    if (loggedOut) await alertLoginFail()

    return false
  }

  try {
    await entry.click({ force: true, timeout: 8000 })
  } catch {
    const b = await entry.boundingBox().catch(() => null)
    if (!b) return false
    await page.mouse.click(b.x + b.width / 2, b.y + b.height / 2)
  }

  await sleep(4000)

  const ok = (await readInbox()).length > 0

  if (ok) {
    clearLoginFlag()
  } else {
    // 点了私信入口还是没有会话列表 → 大概率掉登录
    const loggedOut = await page.evaluate(() => {
      const text = document.body.innerText
      // 出现「登录」按钮且看不到会话列表
      const hasLoginBtn = [...document.querySelectorAll('button, [class*="login"]')]
        .some(el => (el.innerText || '').trim() === '登录')

      return hasLoginBtn || text.includes('扫码登录') || text.includes('手机号登录')
    })

    if (loggedOut) await alertLoginFail()
  }

  return ok
}

/** 读收件箱会话行（含未读数） */
async function readInbox() {
  return page.evaluate(() => {
    const rows = []

    // 抖音改版很频繁，这里按优先级试多个选择器
    let nodes = []
    for (const sel of [
      'div[class*="conversationConversationItemwrapper"]',
      '[data-e2e*="conversation-item"]',
      'div[class*="conversationItem"]',
      'li[class*="conversation"]'
    ]) {
      nodes = [...document.querySelectorAll(sel)]
      if (nodes.length) break
    }

    for (const el of nodes) {
      const r = el.getBoundingClientRect()
      if (!r.width || !r.height) continue
      if (r.right < 0 || r.left > window.innerWidth || r.bottom < 0 || r.top > window.innerHeight) continue
      if (typeof el.checkVisibility === 'function' && !el.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true })) continue

      const lines = (el.innerText || '').split('\n').map(s => s.trim()).filter(Boolean)
      if (!lines.length) continue

      let badge = null
      for (const sel of ['[class*="semi-badge-count"]', '[class*="badge"]', '[class*="unread"]']) {
        badge = el.querySelector(sel) || (el.parentElement && el.parentElement.querySelector(sel))
        if (badge) break
      }

      rows.push({
        nick: lines[0] || '',
        time: lines[1] || '',
        preview: lines[2] || '',
        unread: badge ? (badge.innerText || '').trim() : ''
      })
    }

    return rows
  })
}

async function handleContact(entry, token, ledger) {
  const before = ledger[entry.DouyinId || entry.douyinId] || ''

  await page.goto(entryProfile, { waitUntil: 'domcontentloaded', timeout: 60000 })
  await page.waitForTimeout(6500)

  const entryNick = entry.Nickname || entry.nickname
  const entryProfile = entry.ProfileUrl || entry.profile

  const identity = await page.evaluate(() => ({
    id: (document.body.innerText.match(/抖音号\s*[:：]?\s*([0-9A-Za-z_.\-]+)/) || [])[1] || '',
    nick: (document.title || '').replace('的抖音 - 抖音', '').replace(' - 抖音', '')
  }))

  // 白名单强校验：抖音号必须完全匹配，否则绝不动手
  if (!identity.id.startsWith(entry.DouyinId || entry.douyinId)) {
    log(`  跳过：抖音号不符（页面=${identity.id}，名单=${entry.DouyinId || entry.douyinId}）`)

    return
  }

  const btn = page.locator('button', { hasText: '私信' }).first()
  if (!(await btn.count())) {
    log('  跳过：没有私信入口')

    return
  }

  await btn.click({ force: true })
  await sleep(3000)

  // 面板是异步渲染的，最多等 10 秒
  let bubbles = []
  for (let i = 0; i < 10; i++) {
    bubbles = await readBubbles()
    if (bubbles.length) break
    await sleep(1000)
  }

  if (!bubbles.length) {
    log('  跳过：没有对话')

    return
  }

  const last = bubbles[bubbles.length - 1]
  if (last.fromMe) {
    log('  跳过：最后一条是我方')

    return
  }

  // 非文本（图片/语音/视频）→ 转人工，AI 不猜
  if (last.hasMedia && !last.text) {
    log('  转人工：对方发的是图片/语音/视频')

    await createHandoff(token, {
      ChatId: entry.DouyinId || entry.douyinId,
      Channel: 'douyin',
      NickName: entryNick,
      Category: '非文本消息',
      Reason: '对方发了图片/语音/视频，需人工查看',
      LastMsg: '[非文本消息]',
      Contact: '',
      Urgency: 2,
      Status: 0
    })

    return
  }

  const key = `${last.text.slice(0, 40)}`

  if (ledger[entry.DouyinId || entry.douyinId] === key) {
    log('  跳过：这条已经回过了')

    return
  }

  const history = bubbles
    .slice(-8)
    .map(b => `${b.fromMe ? '我' : '对方'}: ${b.text}`)
    .join('\n')

  log(`  对方说：${last.text.slice(0, 40)}`)

  // ① 对方留了手机号/微信号 → 建「转微信」工单，并回一句确认
  const contact = extractContact(last.text)

  if (contact) {
    log(`  发现联系方式：${contact} → 建转微信工单`)

    await createHandoff(token, {
      ChatId: entry.DouyinId || entry.douyinId,
      Channel: 'douyin',
      NickName: entryNick,
      Category: '转微信',
      Reason: '对方在抖音留了联系方式，需微信跟进',
      LastMsg: last.text.slice(0, 400),
      Contact: contact,
      Urgency: 3,
      Status: 0
    })

    // 回填到线索池：有了号码就升级成 A1
    try {
      await apiPost(token, '/lead/applyLeadContact', { NickName: entryNick, Contact: contact })
      log(`  已回填线索：${entryNick} → ${contact}`)
    } catch {
      // 线索不在池子里就算了
    }

    const ack = '好，我加您微信，微信上细聊'
    const ackResult = await sendReply(ack)

    if (ackResult.sent) {
      ledger[entry.DouyinId || entry.douyinId] = key
      writeJson(LEDGER, ledger)
      log('  已回确认（转微信）')
    }

    return
  }

  // ② 打款/合同/看现场等 → 转人工，AI 不动
  const intent = await checkIntent(token, last.text, history)

  if (intent.NeedHuman) {
    log(`  转人工（${intent.Category}）：${intent.Reason}`)

    await createHandoff(token, {
      ChatId: entry.DouyinId || entry.douyinId,
      Channel: 'douyin',
      NickName: entryNick,
      Category: intent.Category || '需人工',
      Reason: intent.Reason || '',
      LastMsg: last.text.slice(0, 400),
      Contact: '',
      Urgency: intent.Urgency || 2,
      Status: 0
    })

    return
  }

  const reply = await genReply(token, entryNick, last.text, history)
  log(`  AI：${reply}`)

  const result = await sendReply(reply)

  if (result.sent) {
    ledger[entry.DouyinId || entry.douyinId] = key
    writeJson(LEDGER, ledger)
    log('  已发送')
    await addFollowup(token, entryNick, entry.DouyinId || entry.douyinId, '自动回复', `AI 回复：${reply}`)
  } else {
    log('  未发送（dry-run）')
  }
}

const token = (await login().catch(() => '')) || ''
const device = await loadDevice(token)

log(`账号 #${device.Id} ${device.Name}（抖音号 ${device.DouyinId}，端口 ${device.CdpPort}）`)

await connect(device)

let entries = await loadWhitelist(token)

if (!entries.length) {
  log('该账号白名单为空，先空转，等后台添加')
}

log(`白名单 ${entries.length} 人，模式 ${doSend ? '真实发送' : 'dry-run'}`)

let round = 0
let lastReport = 0
let lastInboxCount = 0

do {
  round++
  const ledger = readJson(LEDGER, {})

  try {
    const ok = await ensureInbox()

    if (!ok) {
      log('打不开收件箱')
    } else {
      const allRows = await readInbox()
      lastInboxCount = allRows.length
      const names = entries.map(e => e.Nickname || e.nickname)

      // 抖音有同名账号（比如两个「庸」），只保留最上面那条，避免快照来回翻转
      const seen = new Set()
      const rows = allRows.filter(r => {
        if (seen.has(r.nick)) return false
        seen.add(r.nick)

        return true
      })

      // 触发条件：只看「会话预览是否变化」
      // 未读角标不可靠——从主页进会话不会清未读，会导致每分钟空跑
      const state = readJson(INBOX_STATE, {})
      const now = Date.now()

      const pending = rows.filter(r => {
        if (!names.some(n => r.nick.includes(n))) return false

        const prev = state[r.nick]
        if (!prev) return false                      // 首次只记录，不触发

        if (prev.preview === r.preview) return false // 内容没变，不动

        // 冷却：5 分钟内对同一人只尝试一次，避免我们自己的回复触发反复访问
        if (prev.lastAttempt && now - prev.lastAttempt < 5 * 60 * 1000) return false

        return true
      })

      // 更新快照
      for (const r of rows) {
        if (!names.some(n => r.nick.includes(n))) continue

        const prev = state[r.nick] || {}
        state[r.nick] = {
          preview: r.preview,
          lastAttempt: pending.some(p => p.nick === r.nick) ? now : prev.lastAttempt || 0
        }
      }
      writeJson(INBOX_STATE, state)

      if (!pending.length) {
        if (once) log(`收件箱 ${rows.length} 个会话，名单内无未读`)
      } else {
        log(`名单内有未读：${pending.map(p => p.nick).join('、')}`)

        for (const entry of entries) {
          if (!pending.some(p => p.nick.includes(entryNick))) continue

          try {
            await handleContact(entry, token, ledger)
          } catch (e) {
            log(`  ${entryNick} 处理失败: ${String(e).slice(0, 80)}`)
          }
        }
      }
    }
  } catch (e) {
    const msg = String(e)

    log(`本轮异常: ${msg.slice(0, 80)}`)

    // 浏览器/页面被关：重连一次
    if (/closed|Target page|Browser has been closed|disconnected/i.test(msg)) {
      try {
        log('  尝试重连浏览器…')
        await connect(device)
        await sleep(2000)
      } catch (err2) {
        log(`  重连失败: ${String(err2).slice(0, 60)}`)
      }
    }
  }

  try {
    fs.writeFileSync(`E:\\ai\\data\\dy_heartbeat-${deviceId}.txt`, new Date().toLocaleString('sv-SE'), 'utf-8')
  } catch {
    // 心跳失败不影响主流程
  }

  // 每轮刷新白名单（后台改完不用重启）
  if (!once) {
    try {
      entries = await loadWhitelist(token)
      await touchDevice(token)
    } catch {
      // 后端暂时不可用时沿用旧名单
    }
  }

  if (!once && Date.now() - lastReport > 600000) {
    log(`监听中：第 ${round} 轮，白名单 ${entries.length} 人，收件箱 ${lastInboxCount} 个会话`)
    lastReport = Date.now()
  }

  if (once) break

  await sleep(jitter(baseInterval))
} while (true)

await browser.close().catch(() => {})
