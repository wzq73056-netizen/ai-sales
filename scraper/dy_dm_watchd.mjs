/**
 * 私信自动化守护进程（由后台启动）
 *
 * 每轮读取控制文件 E:\ai\data\dm_control.json：
 *   { "listen": true, "autoReply": false, "autoSend": false, "interval": 60 }
 *
 * 行为：
 *   listen=false      → 空转（不碰浏览器）
 *   listen=true       → 读会话列表，记录变化
 *   autoReply=true    → 有变化时调用 dy_dm_auto.mjs 逐个生成回复
 *   autoSend=true     → 上面那步加 --send（真实发送），否则 dry-run
 *
 * 状态写入 E:\ai\data\dm_watchd_status.json，供后台/前端展示。
 */
import { chromium } from 'playwright-core'
import fs from 'node:fs'
import { spawn } from 'node:child_process'
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


const CONTROL = 'E:\\ai\\data\\dm_control.json'
const STATUS = 'E:\\ai\\data\\dm_watchd_status.json'
const STATE = 'E:\\ai\\data\\dm_inbox_state.json'
const CHANGE_LOG = 'E:\\ai\\data\\dm_inbox_changes.json'

const OPENER = (() => {
  try {
    return fs.readFileSync('E:\\ai\\data\\dm_opener.txt', 'utf-8').trim()
  } catch {
    return 'https://www.douyin.com/jingxuan'
  }
})()

const sleep = ms => new Promise(r => setTimeout(r, ms))
const stamp = () => new Date().toLocaleTimeString('zh-CN', { hour12: false })

function readControl() {
  try {
    const c = JSON.parse(fs.readFileSync(CONTROL, 'utf-8'))
    return {
      listen: !!c.listen,
      autoReply: !!c.autoReply,
      autoSend: !!c.autoSend,
      interval: Number(c.interval) > 0 ? Number(c.interval) : 60
    }
  } catch {
    return { listen: false, autoReply: false, autoSend: false, interval: 60 }
  }
}

function writeStatus(patch) {
  let cur = {}

  try {
    cur = JSON.parse(fs.readFileSync(STATUS, 'utf-8'))
  } catch {}

  fs.writeFileSync(STATUS, JSON.stringify({ ...cur, ...patch, updatedAt: new Date().toISOString() }, null, 2), 'utf-8')
}

function readState() {
  try {
    return JSON.parse(fs.readFileSync(STATE, 'utf-8'))
  } catch {
    return { conversations: {} }
  }
}

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

      let badge = el.querySelector('[class*="semi-badge-count"]')
      if (!badge && el.parentElement) badge = el.parentElement.querySelector('[class*="semi-badge-count"]')

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

async function readBubbles() {
  return page.evaluate(() => {
    let n = 0
    for (const el of document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]')) {
      const r = el.getBoundingClientRect()
      if (r.width && r.height && (el.innerText || '').trim()) n++
    }
    return n
  })
}

async function ensureInbox() {
  const rows = await readList()
  if (rows.length) {
    if ((await readBubbles()) > 0) {
      const box = await page.evaluate(() => {
        const el = document.querySelector('[class*="StackLayoutStackTitleBarleftArea"]')
        if (!el) return null
        const r = el.getBoundingClientRect()
        return { x: r.left + Math.min(15, r.width / 2), y: r.top + r.height / 2 }
      })
      if (box) {
        await page.mouse.click(box.x, box.y)
        await page.waitForTimeout(1800)
      }
    }
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

  return (await readList()).length > 0
}

function runAutoReply(autoSend) {
  return new Promise(resolve => {
    const argv = ['dy_dm_auto.mjs', '--limit', '1']
    argv.push(autoSend ? '--send' : '--dry-run')

    const child = spawn('node', argv, { cwd: 'E:\\ai\\scraper', stdio: ['ignore', 'pipe', 'pipe'] })

    let out = ''
    child.stdout.on('data', d => (out += d.toString()))
    child.stderr.on('data', d => (out += d.toString()))

    child.on('close', () => resolve(out.trim().split('\n').slice(-8).join('\n')))
  })
}

console.log(`[${stamp()}] daemon 启动`)
writeStatus({ running: true, startedAt: new Date().toISOString(), lastError: '' })

let rounds = 0

while (true) {
  const ctl = readControl()
  rounds++

  if (!ctl.listen) {
    writeStatus({ running: true, listening: false, rounds, lastCheckAt: new Date().toLocaleString('sv-SE') })
    await sleep(Math.max(5, ctl.interval) * 1000)
    continue
  }

  try {
    await ensurePage()

    if (!(await ensureInbox())) {
      writeStatus({ running: true, listening: true, rounds, lastError: '打不开消息面板' })
      await sleep(ctl.interval * 1000)
      continue
    }

    const rows = await readList()
    const state = readState()
    const prev = state.conversations || {}
    const changes = []

    for (const row of rows) {
      if (!row.nick) continue

      const old = prev[row.nick]
      if (!old || old.preview !== row.preview || Number(row.unread) > 0) {
        changes.push(row)
      }
      prev[row.nick] = { preview: row.preview, time: row.time, unread: row.unread }
    }

    fs.writeFileSync(STATE, JSON.stringify({ conversations: prev, updatedAt: new Date().toISOString() }, null, 2), 'utf-8')

    let autoReplyLog = ''

    if (changes.length && ctl.autoReply) {
      autoReplyLog = await runAutoReply(ctl.autoSend)
      console.log(`[${stamp()}] 自动回复(${ctl.autoSend ? '发送' : '预演'}):\n${autoReplyLog}`)
    }

    if (changes.length) {
      let log = []
      try {
        log = JSON.parse(fs.readFileSync(CHANGE_LOG, 'utf-8'))
      } catch {}

      const now = new Date().toLocaleString('sv-SE')
      log.push(...changes.map(c => ({ at: now, nick: c.nick, time: c.time, preview: c.preview, unread: c.unread })))
      fs.writeFileSync(CHANGE_LOG, JSON.stringify(log.slice(-500), null, 2), 'utf-8')
    }

    writeStatus({
      running: true,
      listening: true,
      rounds,
      conversations: rows.length,
      changes: changes.length,
      lastChanges: changes.slice(0, 8).map(c => ({ nick: c.nick, preview: c.preview, unread: c.unread })),
      lastAutoReply: autoReplyLog,
      autoReply: ctl.autoReply,
      autoSend: ctl.autoSend,
      lastCheckAt: new Date().toLocaleString('sv-SE'),
      lastError: ''
    })
  } catch (e) {
    console.log(`[${stamp()}] 异常: ${String(e).slice(0, 100)}`)
    writeStatus({ running: true, lastError: String(e).slice(0, 120) })
    page = null
  }

  await sleep(Math.max(5, ctl.interval) * 1000)
}
