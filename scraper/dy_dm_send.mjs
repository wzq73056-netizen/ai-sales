/**
 * 抖音私信发送（在收件箱里回复指定会话）
 *
 * 用法：
 *   node dy_dm_send.mjs --nick "六安市叶集" --message "内容"           # 只输入不发送
 *   node dy_dm_send.mjs --nick "六安市叶集" --message "内容" --send    # 真实发送
 *   node dy_dm_send.mjs --dry-run-list                              # 只看有哪些会话
 *
 * 安全：
 *   - 默认 dry-run，只有 --send 才点发送
 *   - 发送前会打印目标会话昵称 + 最后一条消息，便于核对
 *   - 复用收件箱入口（顶部消息图标），不依赖个人主页
 *
 * 说明：发送按钮是圆形图标（无 text），只能按坐标点；输入框取读回校验。
 */
import { chromium } from 'playwright-core'
import fs from 'node:fs'
import { applyStealth, humanDelay, humanType, takeQuota } from './humanize.mjs'
/** CDP 连接装饰：记录本进程开的标签，browser.close() 时一起关掉（否则每跑一次留一个标签） */
function trackBrowser(browser) {
  const ctx = browser.contexts()[0]
  const owned = []
  const newPage = ctx.newPage.bind(ctx)
  ctx.newPage = async () => {
    const page = await newPage()

  // 反检测注入（必须在 goto 之前）
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


const args = process.argv.slice(2)
const opt = (name, def) => {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? args[i + 1] : def
}

const nick = opt('nick', '')
const message = opt('message', '')
const doSend = args.includes('--send')
const listOnly = args.includes('--dry-run-list')

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

if (!nick && !listOnly) {
  console.log('用法: node dy_dm_send.mjs --nick <昵称> --message "内容" [--send]')
  process.exit(1)
}

const browser = trackBrowser(await chromium.connectOverCDP(process.env.CDP || 'http://127.0.0.1:9333'))
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

/** 读可见会话行 */
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

      rows.push({
        nick: lines[0] || '',
        time: lines[1] || '',
        preview: lines[2] || '',
        x: Math.round(r.left + r.width / 2),
        y: Math.round(r.top + r.height / 2)
      })
    }

    return rows.sort((a, b) => a.y - b.y)
  })
}

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

/** 打开收件箱列表 */
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

if (!(await ensureInbox())) {
  console.log('打不开消息面板')
  await browser.close()
  process.exit(2)
}

const rows = await readList()

if (listOnly) {
  console.log(JSON.stringify(rows.map(r => ({ nick: r.nick, time: r.time, preview: r.preview })), null, 2))
  await browser.close()
  process.exit(0)
}

const row = rows.find(r => r.nick.includes(nick))
if (!row) {
  console.log(`列表里没有包含「${nick}」的会话`)
  await browser.close()
  process.exit(3)
}

await page.mouse.click(row.x, row.y)
await page.waitForTimeout(2500)

const before = await readBubbles()
const lastFromThem = [...before].reverse().find(b => !b.fromMe)

console.log(`[${stamp()}] 目标: ${row.nick}`)
console.log(`  最后一条对方消息: ${lastFromThem ? lastFromThem.text.slice(0, 60) : '(无)'}`)

if (!message) {
  console.log('  未提供 --message，仅查看')
  await browser.close()
  process.exit(0)
}

// 输入
const input = page.locator('div[contenteditable="true"], textarea').first()
if (!(await input.count())) {
  console.log('未找到输入框')
  await browser.close()
  process.exit(4)
}

await input.click()
await page.waitForTimeout(400)
await input.fill(message)
await page.waitForTimeout(600)

const typed = await page.evaluate(() => {
  const el = document.querySelector('div[contenteditable="true"], textarea')
  return el ? (el.innerText || el.value || '').replace(/\u200b/g, '').trim() : ''
})

if (!typed.includes(message.slice(0, 10))) {
  console.log('文本注入失败:', typed.slice(0, 60))
  await browser.close()
  process.exit(5)
}

if (!doSend) {
  // 清空输入框，保证不留草稿
  await page.keyboard.press('Control+A')
  await page.keyboard.press('Delete')
  console.log('dry-run：已输入并校验，未发送（已清空输入框）')
  await browser.close()
  process.exit(0)
}

// 发送：工具栏最右图标
const sendPos = await page.evaluate(() => {
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

if (!sendPos) {
  console.log('没找到发送按钮')
  await browser.close()
  process.exit(6)
}

await page.mouse.click(sendPos.x, sendPos.y)
await page.waitForTimeout(2500)

const appears = await page.evaluate(msg => document.body.innerText.includes(msg), message)

console.log(JSON.stringify({ ok: appears, sent: appears, nick: row.nick, message: message.slice(0, 40) }, null, 2))

await backToList()
await browser.close()
