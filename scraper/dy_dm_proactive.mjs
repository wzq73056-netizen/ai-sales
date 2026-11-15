/**
 * 抖音主动私信（白名单限定，默认只允许「庸」134527749）
 *
 * 用法：
 *   node dy_dm_proactive.mjs --nick 庸 --douyin-id 134527749 --message "你好" [--send]
 *   node dy_dm_proactive.mjs --nick 庸 --douyin-id 134527749 --ai "鸡蛋批发询价" [--send]
 *
 * 与被动回复的区别：这里是从对方主页点「私信」主动发起，
 * 不要求已有会话；发送前会核对主页抖音号，发错人直接中止。
 */
import fs from 'node:fs'
import { chromium } from 'playwright-core'
/** CDP 连接装饰：记录本进程开的标签，browser.close() 时一起关掉（否则每跑一次留一个标签） */
function trackBrowser(browser) {
  const ctx = browser.contexts()[0]
  const owned = []
  const newPage = ctx.newPage.bind(ctx)
  ctx.newPage = async () => {
    const page = await newPage()
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
const opt = (name, def) => {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? args[i + 1] : def
}

const nick = opt('nick', '')
const douyinId = opt('douyin-id', '')
const message = opt('message', '')
const aiHint = opt('ai', '')
const doSend = args.includes('--send')

const LIMIT_FILE = 'E:\\ai\\data\\dy_dm_limit.json'
const LEDGER = 'E:\\ai\\data\\dy_dm_sent.json'

const sleep = ms => new Promise(r => setTimeout(r, ms))
const log = msg => console.log(`[${new Date().toLocaleTimeString('zh-CN', { hour12: false })}] ${msg}`)

if (!nick) {
  console.log(JSON.stringify({ ok: false, reason: '缺少 --nick' }))
  process.exit(1)
}

/** 频率限制：60 秒一条，每天 15 条 */
function checkLimit() {
  let state = { lastAt: 0, today: '', count: 0 }

  try {
    state = { ...state, ...JSON.parse(fs.readFileSync(LIMIT_FILE, 'utf-8')) }
  } catch {}

  const today = new Date().toLocaleDateString('sv-SE')

  if (state.today !== today) state = { lastAt: state.lastAt, today, count: 0 }

  const gap = Date.now() - state.lastAt

  if (gap < 60000) return { ok: false, reason: `发太快，等 ${Math.ceil((60000 - gap) / 1000)} 秒` }
  if (state.count >= 15) return { ok: false, reason: '今天已发 15 条，超上限' }

  return { ok: true, state }
}

function recordSend(state) {
  state.lastAt = Date.now()
  state.count = (state.count || 0) + 1
  fs.writeFileSync(LIMIT_FILE, JSON.stringify(state), 'utf-8')
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

async function checkWhitelist(token) {
  const res = await fetch(`${API}/lead/getDouyinWhitelist?platform=douyin`, { headers: { 'x-token': token } })
  const json = await res.json()
  const list = json?.data?.list || []

  return list.find(item => (item.Nickname || '').includes(nick) && Number(item.Enabled) === 1)
}

async function genMessage(token) {
  const res = await fetch(`${API}/lead/genDmReply`, {
    method: 'POST',
    headers: { 'x-token': token, 'Content-Type': 'application/json' },
    body: JSON.stringify({ Nick: nick, LastMessage: aiHint || '第一次打招呼，想询个鸡蛋批发价', History: '' , Channel: 'douyin'})
  })
  const json = await res.json()

  if (json.code !== 0) throw new Error(json.msg)

  return json.data.Reply
}

const token = await login()

// 白名单不再是门槛：登记过就顺便做抖音号校验
const allowed = await checkWhitelist(token)
const expectId = douyinId || allowed?.DouyinId || ''

const text = message || (aiHint ? await genMessage(token) : '')

if (!text) {
  console.log(JSON.stringify({ ok: false, reason: '没有内容可发' }))
  process.exit(1)
}

const browser = trackBrowser(await chromium.connectOverCDP(process.env.CDP || 'http://127.0.0.1:9333'))
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

// 从自己的私信入口进去，避免直接猜主页 URL
await page.goto('https://www.douyin.com/jingxuan', { waitUntil: 'domcontentloaded', timeout: 60000 })
await sleep(6000)

const entry = page.locator('[data-e2e="im-entry"]').first()
if (!(await entry.count())) {
  console.log(JSON.stringify({ ok: false, reason: '找不到私信入口' }))
  process.exit(3)
}

await entry.click({ force: true }).catch(async () => {
  const b = await entry.boundingBox().catch(() => null)
  if (b) await page.mouse.click(b.x + b.width / 2, b.y + b.height / 2)
})
await sleep(4000)

// 在会话列表里找目标
const rowInfo = await page.evaluate(nick => {
  const rows = [...document.querySelectorAll('div[class*="conversationConversationItemwrapper"]')]
    .map(el => {
      const r = el.getBoundingClientRect()
      const lines = (el.innerText || '').split('\n').map(s => s.trim()).filter(Boolean)

      return { nick: lines[0] || '', x: Math.round(r.left + r.width / 2), y: Math.round(r.top + r.height / 2) }
    })
    .filter(r => r.nick && r.x > 0 && r.y > 0)

  return rows.find(r => r.nick.includes(nick)) || null
}, nick)

if (!rowInfo) {
  console.log(JSON.stringify({ ok: false, reason: `会话列表里没有 ${nick}，抖音网页版对陌生人需要先有对话` }))
  process.exit(4)
}

await page.mouse.click(rowInfo.x, rowInfo.y)
await sleep(3000)

const bubbles = await page.evaluate(() => {
  const out = []

  for (const el of document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]')) {
    const r = el.getBoundingClientRect()
    if (!r.width || !r.height) continue

    const text = (el.innerText || '').replace(/\s+/g, ' ').trim()
    if (text) out.push({ text: text.slice(0, 40), top: r.top })
  }

  return out.sort((a, b) => a.top - b.top)
})

log(`目标：${nick}，当前对话 ${bubbles.length} 条`)

// 输入
const input = page.locator('div[contenteditable="true"], textarea').first()
if (!(await input.count())) {
  console.log(JSON.stringify({ ok: false, reason: '没有输入框' }))
  process.exit(5)
}

await input.click()
await sleep(400)
await input.fill(text)
await sleep(700)

const typed = await page.evaluate(() => {
  const el = document.querySelector('div[contenteditable="true"], textarea')

  return el ? (el.innerText || el.value || '').replace(/\u200b/g, '').trim() : ''
})

if (!typed.includes(text.slice(0, 6))) {
  console.log(JSON.stringify({ ok: false, reason: '输入校验失败' }))
  process.exit(6)
}

if (!doSend) {
  await page.keyboard.press('Control+A')
  await page.keyboard.press('Delete')
  console.log(JSON.stringify({ ok: true, dryRun: true, nick, expectId, message: text }))
  await page.close()
  process.exit(0)
}

const limit = checkLimit()
if (!limit.ok) {
  await page.keyboard.press('Control+A')
  await page.keyboard.press('Delete')
  console.log(JSON.stringify({ ok: false, reason: limit.reason }))
  process.exit(7)
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

if (!pos) {
  console.log(JSON.stringify({ ok: false, reason: '找不到发送按钮' }))
  process.exit(8)
}

await page.mouse.click(pos.x, pos.y)
await sleep(2500)

const sent = await page.evaluate(m => document.body.innerText.includes(m), text)

if (sent) {
  recordSend(limit.state)

  const ledger = (() => {
    try {
      return JSON.parse(fs.readFileSync(LEDGER, 'utf-8'))
    } catch {
      return {}
    }
  })()

  ledger[expectId || nick] = { text: text.slice(0, 60), at: new Date().toLocaleString('sv-SE') }
  fs.writeFileSync(LEDGER, JSON.stringify(ledger, null, 2), 'utf-8')
}

console.log(JSON.stringify({ ok: sent, sent, nick, expectId, message: text }))
await page.close()
