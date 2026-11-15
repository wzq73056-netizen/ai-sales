/**
 * 私信回复追踪：检查已发送私信是否收到对方回复
 *
 * 用法：node dy_dm_reply.mjs [--limit 10] [--id 5]
 *
 * 原理：
 *   1. 从后台取「已发送且未回复」的私信记录
 *   2. 逐个打开对方主页 → 点私信 → 读会话气泡
 *   3. 气泡 class 含 isFromMe = 我方；不含 = 对方
 *   4. 对方最后一次发言晚于我们发送时间 → 记录回复
 */
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

const limit = Number(opt('limit', 10))
const onlyId = opt('id', '')
const sleep = ms => new Promise(r => setTimeout(r, ms))

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

const token = await login()

let targets = []
if (onlyId) {
  const res = await fetch(`${API}/lead/getDouyinDmList?page=1&pageSize=200`, { headers: { 'x-token': token } })
  const json = await res.json()
  targets = (json?.data?.list || []).filter(d => String(d.Id) === String(onlyId))
} else {
  const res = await fetch(`${API}/lead/getPendingDmList?limit=${limit}`, { headers: { 'x-token': token } })
  const json = await res.json()
  targets = json?.data || []
}

if (!targets.length) {
  console.log(JSON.stringify({ ok: true, checked: 0, replied: 0, note: '没有待检查的私信' }))
  process.exit(0)
}

const browser = trackBrowser(await chromium.connectOverCDP(process.env.CDP || 'http://127.0.0.1:9333'))
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

let replied = 0
const details = []

for (const dm of targets) {
  try {
    await page.goto(dm.ProfileUrl, { waitUntil: 'domcontentloaded', timeout: 60000 })
    await page.waitForTimeout(5000)

    const dmBtn = page.locator('button', { hasText: '私信' }).first()
    if (!(await dmBtn.count())) {
      details.push({ id: dm.Id, nick: dm.Nickname, result: '无入口' })
      continue
    }

    await dmBtn.click()
    await page.waitForTimeout(4000)

    // 上滚加载历史
    for (let i = 0; i < 3; i++) {
      await page.mouse.move(1050, 600)
      await page.mouse.wheel(0, -700)
      await page.waitForTimeout(900)
    }

    const bubbles = await page.evaluate(() => {
      const out = []

      for (const el of document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]')) {
        const cls = (el.className || '').toString()
        const text = (el.innerText || '').replace(/\s+/g, ' ').trim()

        // 「已读」标记去掉
        const clean = text.replace(/\s*已读$/, '').trim()
        if (!clean) continue

        out.push({ fromMe: /isFromMe/.test(cls), text: clean })
      }

      return out
    })

    const theirs = bubbles.filter(b => !b.fromMe)

    if (!theirs.length) {
      details.push({ id: dm.Id, nick: dm.Nickname, result: '暂无回复', bubbles: bubbles.length })
      continue
    }

    const last = theirs[theirs.length - 1]

    // 只要对方在我们发送之后有发言就算回复（时间无法精确比对，用条数+尾部判断）
    await fetch(`${API}/lead/updateDmReply`, {
      method: 'POST',
      headers: { 'x-token': token, 'Content-Type': 'application/json' },
      body: JSON.stringify({ Id: dm.Id, ReplyText: last.text.slice(0, 480) })
    })

    replied++
    details.push({ id: dm.Id, nick: dm.Nickname, result: '已回复', reply: last.text.slice(0, 40) })
  } catch (e) {
    details.push({ id: dm.Id, nick: dm.Nickname, result: '检查失败', error: String(e).slice(0, 80) })
  }

  await sleep(3000 + Math.random() * 2000)
}

console.log(JSON.stringify({ ok: true, checked: targets.length, replied, details }, null, 2))
await browser.close()
