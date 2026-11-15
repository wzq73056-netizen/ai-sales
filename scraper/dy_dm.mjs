/**
 * 抖音私信发送（带查重 + 记录 + 话术随机）
 *
 * 用法：
 *   node dy_dm.mjs --profile <主页URL> [--message "内容"] [--send] [--force]
 *
 *   --message 不传 => 从 data/dm_scripts.json 随机取一条
 *   --send            => 真实发送（默认只输入）
 *   --force           => 忽略「已发过」的查重结果
 *
 * 流程（对齐原项目 CommentToMsgHandler 的两段式）：
 *   1. 查重：同一主页已发送过则拒绝
 *   2. 打开主页 → 核对身份 → 点私信
 *   3. 注入文本（读回校验）→ 点发送
 *   4. 记录结果到 ai_douyin_dm
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
const profile = opt('profile', '')
const doSend = args.includes('--send')
const force = args.includes('--force')
const SCRIPT_FILE = 'E:\\ai\\data\\dm_scripts.json'

if (!profile) {
  console.log('用法: node dy_dm.mjs --profile <URL> [--message "内容"] [--send] [--force]')
  process.exit(1)
}

const sleep = ms => new Promise(r => setTimeout(r, ms))

let message = opt('message', '')
if (!message) {
  try {
    const scripts = JSON.parse(fs.readFileSync(SCRIPT_FILE, 'utf-8')).scripts || []
    message = scripts.length ? scripts[Math.floor(Math.random() * scripts.length)] : '你好'
  } catch {
    message = '你好'
  }
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

const token = await login()

// 1. 查重
if (token && !force) {
  const res = await fetch(`${API}/lead/checkDouyinDm`, {
    method: 'POST',
    headers: { 'x-token': token, 'Content-Type': 'application/json' },
    body: JSON.stringify({ ProfileUrl: profile })
  })
  const json = await res.json()
  const data = json?.data || {}

  if (data.Allowed === false) {
    if (data.Reason === 'interval' && args.includes('--wait')) {
      console.log(`距上次发送不足间隔，等待 ${data.WaitSec} 秒…`)
      await sleep(data.WaitSec * 1000)
    } else {
      console.log(
        JSON.stringify(
          {
            ok: false,
            reason: data.Message || data.Reason,
            code: data.Reason,
            todayCount: data.TodayCount,
            dailyLimit: data.DailyLimit,
            waitSec: data.WaitSec
          },
          null,
          2
        )
      )
      process.exit(6)
    }
  }

  console.log(`限频检查通过（今日 ${data.TodayCount}/${data.DailyLimit}，间隔 ${data.IntervalSec}s）`)
}

const browser = trackBrowser(await chromium.connectOverCDP(process.env.CDP || 'http://127.0.0.1:9333'))
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

await page.goto(profile, { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(6000)

const identity = await page.evaluate(() => {
  const text = document.body.innerText
  const douyinId = (text.match(/抖音号[:：]\s*([^\s]+)/) || [])[1] || ''
  const nick = (document.title || '').replace('的抖音 - 抖音', '').replace(' - 抖音', '')
  return { nick, douyinId }
})
console.log('目标:', JSON.stringify(identity))

const dmBtn = page.locator('button', { hasText: '私信' }).first()
if (!(await dmBtn.count())) {
  console.log(JSON.stringify({ ok: false, reason: '无「私信」按钮（可能非互关）' }))
  await browser.close()
  process.exit(2)
}

await dmBtn.click()
await sleep(4000)

const input = page.locator('div[contenteditable="true"], textarea').first()
if (!(await input.count())) {
  console.log(JSON.stringify({ ok: false, reason: '未找到输入框' }))
  await browser.close()
  process.exit(3)
}

await input.click()
await sleep(400)
await input.fill(message)
await sleep(600)

// 读回校验（对齐移动端 paste 后的 readText 校验）
const typed = await page.evaluate(() => {
  const el = document.querySelector('div[contenteditable="true"], textarea')
  return el ? (el.innerText || el.value || '').replace(/\u200b/g, '').trim() : ''
})

if (!typed.includes(message.slice(0, 10))) {
  console.log(JSON.stringify({ ok: false, reason: '文本注入失败', typed: typed.slice(0, 40) }))
  await browser.close()
  process.exit(4)
}

await page.screenshot({ path: 'E:\\ai\\shots\\dy-dm-typed.png' })
console.log('已注入文本并校验通过')

if (!doSend) {
  console.log(JSON.stringify({ ok: true, mode: 'dry-run', message }))
  await browser.close()
  process.exit(0)
}

// 点发送（工具栏最右图标）
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

await page.mouse.click(sendPos.x, sendPos.y)
await sleep(2500)

// 校验：消息出现在会话里
const appears = await page.evaluate(msg => document.body.innerText.includes(msg), message)

let status = appears ? 1 : 2
let error = appears ? '' : '发送后未在会话中找到该消息'

if (token) {
  await fetch(`${API}/lead/saveDouyinDm`, {
    method: 'POST',
    headers: { 'x-token': token, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      ProfileUrl: profile,
      Nickname: identity.nick,
      DouyinId: identity.douyinId,
      Message: message,
      Status: status,
      Error: error
    })
  })
}

console.log(JSON.stringify({ ok: appears, sent: appears, message, identity }, null, 2))
await browser.close()
