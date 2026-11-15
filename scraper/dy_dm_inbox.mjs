/**
 * 抖音私信收件箱监听（只读）
 *
 * 用法：
 *   node dy_dm_inbox.mjs --once                 # 读一次会话列表
 *   node dy_dm_inbox.mjs --interval 60          # 常驻，每分钟读一次
 *   node dy_dm_inbox.mjs --once --nick 上林弘康   # 点开指定会话看完整内容
 *   node dy_dm_inbox.mjs --once --open          # 自动点开有变化的会话
 *
 * 视图模型（实测）：
 *   点主页「私信」→ 打开消息面板，可能同时渲染「列表 + 某个会话详情」
 *   标题栏左区（StackLayoutStackTitleBarleftArea）是返回键，点了气泡清空、只剩列表
 *   会话行 class 含 conversationConversationItemwrapper
 *   气泡 class 含 messageMessageBoxcontentBox，含 isFromMe 表示我方
 *
 * 只读：不发送、不回复。
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

const interval = Number(opt('interval', 60)) * 1000
const once = args.includes('--once')
const doOpen = args.includes('--open')
const nickFilter = opt('nick', '')
const maxOpen = Number(opt('max-open', 3))

const STATE_FILE = 'E:\\ai\\data\\dm_inbox_state.json'
const CHANGE_LOG = 'E:\\ai\\data\\dm_inbox_changes.json'
const OPENER =
  opt('opener', '') ||
  (() => {
    try {
      return fs.readFileSync('E:\\ai\\data\\dm_opener.txt', 'utf-8').trim()
    } catch {
      return 'https://www.douyin.com/jingxuan'
    }
  })()

const sleep = ms => new Promise(r => setTimeout(r, ms))
const stamp = () => new Date().toLocaleTimeString('zh-CN', { hour12: false })

let browser = null
let page = null

async function ensurePage() {
  if (!browser || !browser.isConnected()) {
    browser = trackBrowser(await chromium.connectOverCDP(process.env.CDP || 'http://127.0.0.1:9333'))
  }

  const ctx = browser.contexts()[0]
  if (!page || page.isClosed()) {
    page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
  }

  return page
}

/** 读可见的会话行 */
async function readList() {
  return page.evaluate(() => {
    const rows = []

    for (const el of document.querySelectorAll('div[class*="conversationConversationItemwrapper"]')) {
      const r = el.getBoundingClientRect()

      // 列表被移出视口时仍有尺寸，必须判可见
      if (!r.width || !r.height) continue
      if (r.right < 0 || r.left > window.innerWidth || r.bottom < 0 || r.top > window.innerHeight) continue
      if (typeof el.checkVisibility === 'function' && !el.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true })) continue

      const lines = (el.innerText || '').split('\n').map(s => s.trim()).filter(Boolean)
      if (!lines.length) continue

      const badge = el.querySelector('[class*="badge"]')

      rows.push({
        nick: lines[0] || '',
        time: lines[1] || '',
        preview: (lines[2] || '').trim(),
        unread: badge ? (badge.innerText || '').trim() : '',
        x: Math.round(r.left + r.width / 2),
        y: Math.round(r.top + r.height / 2)
      })
    }

    return rows.sort((a, b) => a.y - b.y)
  })
}

/** 读当前会话气泡 */
async function readBubbles() {
  return page.evaluate(() => {
    const out = []

    for (const el of document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]')) {
      const r = el.getBoundingClientRect()
      if (!r.width || !r.height) continue

      const text = (el.innerText || '').replace(/\s+/g, ' ').replace(/\s*已读$/, '').trim()
      if (!text) continue

      out.push({ fromMe: /isFromMe/.test((el.className || '').toString()), text: text.slice(0, 200) })
    }

    return out
  })
}

/** 点标题栏左区返回列表（实测：点完气泡清空） */
async function backToList() {
  const box = await page.evaluate(() => {
    const el = document.querySelector('[class*="StackLayoutStackTitleBarleftArea"]')
    if (!el) return null

    const r = el.getBoundingClientRect()
    return { x: r.left + Math.min(15, r.width / 2), y: r.top + r.height / 2 }
  })

  if (!box) return false

  await page.mouse.click(box.x, box.y)
  await page.waitForTimeout(2000)

  return (await readBubbles()).length === 0
}

/** 打开消息面板并确保停在列表视图 */
async function openPanelToList() {
  // 面板已经开着且停在列表，直接复用，避免反复导航导致的点击失败
  if ((await readList()).length > 0) {
    if ((await readBubbles()).length > 0) await backToList()
    return (await readList()).length > 0
  }

  // 顶部「消息」图标直接打开收件箱列表（不需要借道别人主页）
  await page.goto(OPENER, { waitUntil: 'domcontentloaded', timeout: 60000 })
  await page.waitForTimeout(6000)

  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(400)

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

  // 面板若直接进了某个会话，先返回列表
  if ((await readBubbles()).length > 0) {
    await backToList()
  }

  return (await readList()).length > 0
}

/** 点开某个会话，读气泡，然后返回列表 */
async function openConversation(nick, preview) {
  const rows = await readList()

  const row =
    rows.find(r => r.nick === nick && preview && r.preview.startsWith(preview.slice(0, 6))) ||
    rows.find(r => r.nick === nick)

  if (!row) return null

  await page.mouse.click(row.x, row.y)
  await page.waitForTimeout(1500)

  const key = (preview || '').slice(0, 6)
  let bubbles = []

  for (let i = 0; i < 10; i++) {
    await page.waitForTimeout(1000)
    bubbles = await readBubbles()

    if (key && bubbles.some(b => b.text.includes(key))) break
    if (!key && bubbles.length) break
  }

  await backToList()

  return bubbles
}

function readState() {
  try {
    return JSON.parse(fs.readFileSync(STATE_FILE, 'utf-8'))
  } catch {
    return { conversations: {} }
  }
}

async function tick(round) {
  await ensurePage()

  const opened = await openPanelToList()
  if (!opened) {
    console.log(`[${stamp()}] 打不开消息面板`)
    return
  }

  const allRows = await readList()
  if (!allRows.length) {
    console.log(`[${stamp()}] 列表为空`)
    return
  }

  // 同名会话（比如两个「庸」）只保留最上面那条（列表按最新排序），避免反复判"新增"
  const seenNick = new Set()
  const rows = []

  for (const r of allRows) {
    if (seenNick.has(r.nick)) continue
    seenNick.add(r.nick)
    rows.push(r)
  }

  if (nickFilter) {
    const hit = rows.find(r => r.nick.includes(nickFilter))
    if (!hit) {
      console.log(`[${stamp()}] 没有包含「${nickFilter}」的会话`)
      return
    }

    const bubbles = await openConversation(hit.nick, hit.preview)
    if (!bubbles) {
      console.log(`[${stamp()}] 点开失败`)
      return
    }

    console.log(`[${stamp()}] ${hit.nick}（${bubbles.length} 条）`)
    for (const b of bubbles) console.log(`  ${b.fromMe ? '我方' : '对方'}: ${b.text}`)

    return
  }

  const state = readState()
  const prev = state.conversations || {}
  const changes = []

  for (const row of rows) {
    if (!row.nick) continue

    const old = prev[row.nick]
    const isNew = !old
    const changed = old && old.preview !== row.preview

    if (isNew || changed || Number(row.unread) > 0) {
      changes.push({ ...row, isNew, changed })
    }

    prev[row.nick] = { preview: row.preview, time: row.time, unread: row.unread }
  }

  if (!changes.length) {
    console.log(`[${stamp()}] 第 ${round} 轮：${rows.length} 个会话，无变化`)
  } else {
    console.log(`[${stamp()}] 第 ${round} 轮：${rows.length} 个会话，${changes.length} 条变化`)

    for (const c of changes) {
      console.log(`  ${c.unread ? '[未读] ' : '[新] '}${c.nick} | ${c.time} | ${c.preview.slice(0, 40)}`)
    }

    if (doOpen) {
      for (const c of changes.slice(0, maxOpen)) {
        const bubbles = await openConversation(c.nick, c.preview)
        if (!bubbles) continue

        const theirs = bubbles.filter(b => !b.fromMe)
        console.log(`    ${c.nick}: ${bubbles.length} 条，对方 ${theirs.length} 条`)
        if (theirs.length) console.log(`      对方最后一句：「${theirs[theirs.length - 1].text}」`)
      }
    }
  }

  fs.writeFileSync(STATE_FILE, JSON.stringify({ conversations: prev, updatedAt: new Date().toISOString() }, null, 2), 'utf-8')

  if (changes.length) {
    let log = []

    try {
      log = JSON.parse(fs.readFileSync(CHANGE_LOG, 'utf-8'))
    } catch {}

    const now = new Date().toLocaleString('sv-SE')
    log.push(...changes.map(c => ({ at: now, nick: c.nick, time: c.time, preview: c.preview, unread: c.unread })))
    fs.writeFileSync(CHANGE_LOG, JSON.stringify(log.slice(-500), null, 2), 'utf-8')
  }
}

console.log(`[${stamp()}] 开始监听会话列表（间隔 ${once ? '-' : interval / 1000 + ' 秒'}）`)

let round = 0
do {
  round++

  try {
    await tick(round)
  } catch (e) {
    console.log(`[${stamp()}] 第 ${round} 轮异常: ${String(e).slice(0, 90)}`)
    page = null
  }

  if (once) break
  await sleep(interval)
} while (true)

if (browser) await browser.close().catch(() => {})
console.log(`[${stamp()}] 结束`)
