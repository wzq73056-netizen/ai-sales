/**
 * 抖音私信 AI 自动回复
 *
 * 用法：
 *   node dy_dm_auto.mjs --dry-run              # 只生成回复，打印出来，不发
 *   node dy_dm_auto.mjs --send                 # 生成并真实发送
 *   node dy_dm_auto.mjs --send --limit 3       # 一次最多处理 3 个会话
 *   node dy_dm_auto.mjs --distance              # 只处理"对方最后一条消息"的会话
 *
 * 判定逻辑：
 *   打开会话 → 按气泡的纵坐标取最下面一条 → 不含 isFromMe 说明是对方说的 → 需要回复
 *
 * 安全：
 *   默认 dry-run；--send 才真的点发送；每个会话每次运行只回一条。
 */
import { chromium } from 'playwright-core'
import fs from 'node:fs'
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


const args = process.argv.slice(2)
const opt = (name, def) => {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? args[i + 1] : def
}

const API = process.env.API || 'http://127.0.0.1:8888'
const doSend = args.includes('--send')
const onlyIncoming = !args.includes('--all')
const limit = Number(opt('limit', 3))
const onlyNick = opt('nick', '')

const OPENER =
  (() => {
    try {
      return fs.readFileSync('E:\\ai\\data\\dm_opener.txt', 'utf-8').trim()
    } catch {
      return 'https://www.douyin.com/jingxuan'
    }
  })()

const sleep = ms => new Promise(r => setTimeout(r, ms))
const stamp = () => new Date().toLocaleTimeString('zh-CN', { hour12: false })

/** 登录后台拿 token */
async function login() {
  const res = await fetch(`${API}/base/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: '123456', captcha: '', captchaId: '' })
  })
  const json = await res.json()
  if (json.code !== 0) throw new Error(`登录失败: ${json.msg}`)
  return json.data.token
}

/** 调后端生成回复 */
async function genReply(token, nick, lastMessage, history) {
  const res = await fetch(`${API}/lead/genDmReply`, {
    method: 'POST',
    headers: { 'x-token': token, 'Content-Type': 'application/json' },
    body: JSON.stringify({ Nick: nick, LastMessage: lastMessage, History: history || '' , Channel: 'douyin'})
  })
  const json = await res.json()
  if (json.code !== 0) throw new Error(json.msg)

  return json.data.Reply
}

const browser = trackBrowser(await chromium.connectOverCDP(process.env.CDP || 'http://127.0.0.1:9333'))
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

async function readList() {
  return page.evaluate(() => {
    const rows = []

    for (const el of document.querySelectorAll('div[class*="conversationConversationItemwrapper"]')) {
      const r = el.getBoundingClientRect()
      if (!r.width || !r.height) continue
      if (r.right < 0 || r.left > window.innerWidth || r.bottom < 0 || r.top > window.innerHeight) continue
      if (typeof el.checkVisibility === 'function' && !el.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true })) continue

      const lines = (el.innerText || '').split('\n').map(s => s.trim()).filter(Boolean)
      if (!lines.length) continue

      // 未读角标可能在行内，也可能在父级
      let badge = el.querySelector('[class*="semi-badge-count"]')
      if (!badge && el.parentElement) badge = el.parentElement.querySelector('[class*="semi-badge-count"]')

      rows.push({
        nick: lines[0] || '',
        time: lines[1] || '',
        preview: lines[2] || '',
        unread: badge ? (badge.innerText || '').trim() : '',
        x: Math.round(r.left + r.width / 2),
        y: Math.round(r.top + r.height / 2)
      })
    }

    return rows.sort((a, b) => a.y - b.y)
  })
}

/** 气泡按纵坐标排序，取最下面一条为最新 */
async function readBubbles() {
  return page.evaluate(() => {
    const out = []

    for (const el of document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]')) {
      const r = el.getBoundingClientRect()
      if (!r.width || !r.height) continue

      const text = (el.innerText || '').replace(/\s+/g, ' ').replace(/\s*已读$/, '').trim()
      if (!text) continue

      out.push({ fromMe: /isFromMe/.test((el.className || '').toString()), text: text.slice(0, 300), top: r.top })
    }

    return out.sort((a, b) => a.top - b.top)
  })
}

async function backToList() {
  const box = await page.evaluate(() => {
    const el = document.querySelector('[class*="StackLayoutStackTitleBarleftArea"]')
    if (!el) return null
    const r = el.getBoundingClientRect()
    return { x: r.left + Math.min(15, r.width / 2), y: r.top + r.height / 2 }
  })

  if (!box) return false
  await page.mouse.click(box.x, box.y)
  await page.waitForTimeout(1800)

  return (await readBubbles()).length === 0
}

async function ensureInbox() {
  if ((await readList()).length > 0) {
    if ((await readBubbles()).length > 0) await backToList()
    return true
  }

  await page.goto(OPENER, { waitUntil: 'domcontentloaded', timeout: 60000 })
  await page.waitForTimeout(6000)
  await page.keyboard.press('Escape').catch(() => {})

  const entry = page.locator('[data-e2e="im-entry"]').first()
  if (!(await entry.count())) return false

  try {
    await entry.click({ force: true, timeout: 8000 })
  } catch {
    const b = await entry.boundingBox().catch(() => null)
    if (!b) return false
    await page.mouse.click(b.x + b.width / 2, b.y + b.height / 2)
  }

  await page.waitForTimeout(4000)

  if ((await readBubbles()).length > 0) await backToList()

  return (await readList()).length > 0
}

/** 输入并（可选）发送 */
async function sendToCurrent(reply) {
  const input = page.locator('div[contenteditable="true"], textarea').first()
  if (!(await input.count())) throw new Error('未找到输入框')

  await input.click()
  await page.waitForTimeout(400)
  await input.fill(reply)
  await page.waitForTimeout(600)

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
  await page.waitForTimeout(2200)

  const ok = await page.evaluate(m => document.body.innerText.includes(m), reply)

  return { sent: ok }
}

// ---------------- 主流程 ----------------

const token = doSend ? await login() : (await login().catch(() => '')) || ''

if (!(await ensureInbox())) {
  console.log('打不开消息面板')
  await browser.close()
  process.exit(2)
}

const rows = await readList()
console.log(`[${stamp()}] 会话 ${rows.length} 个，模式: ${doSend ? '真实发送' : 'dry-run'}`)

let handled = 0

for (const row of rows) {
  if (handled >= limit) break
  if (onlyNick && !row.nick.includes(onlyNick)) continue

  // 只处理有未读的（除非显式 --all）
  if (onlyIncoming && !Number(row.unread)) continue

  await page.mouse.click(row.x, row.y)
  await page.waitForTimeout(2500)

  const bubbles = await readBubbles()
  if (!bubbles.length) {
    await backToList()
    continue
  }

  const last = bubbles[bubbles.length - 1]
  if (last.fromMe) {
    console.log(`  ${row.nick}: 最后一条是我方，跳过`)
    await backToList()
    continue
  }

  const history = bubbles
    .slice(-6)
    .map(b => `${b.fromMe ? '我方' : '对方'}: ${b.text}`)
    .join('\n')

  console.log(`  ${row.nick} 对方说：「${last.text.slice(0, 40)}」`)

  try {
    const reply = await genReply(token, row.nick, last.text, history)
    console.log(`    AI 回复：${reply}`)

    const result = await sendToCurrent(reply)
    console.log(`    ${result.sent ? '已发送' : '未发送（dry-run）'}`)
    handled++
  } catch (e) {
    console.log(`    失败: ${String(e).slice(0, 80)}`)
  }

  await backToList()
  await sleep(2000 + Math.random() * 2000)
}

console.log(`[${stamp()}] 处理完毕，共 ${handled} 个会话`)
await browser.close()
