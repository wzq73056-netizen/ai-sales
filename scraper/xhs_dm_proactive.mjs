/**
 * 小红书主动私信（陌生线索）——默认只预览，不发送
 *
 * 用法：
 *   node xhs_dm_proactive.mjs --nick 木子李 --profile /user/profile/xxx?xsec_token=... --message "你好"
 *   node xhs_dm_proactive.mjs ... --send     # 明确要求才真发（需存在私信入口）
 *
 * 现实约束：小红书网页版对陌生人通常不渲染「私信」入口，
 * 本脚本会先检测入口是否存在，没有就如实返回 no_dm_entry，绝不盲点。
 */
import fs from 'node:fs'
import { chromium } from 'playwright-core'

const API = process.env.API || 'http://127.0.0.1:8888'
const args = process.argv.slice(2)
const opt = (name, def) => {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? args[i + 1] : def
}

const nick = opt('nick', '')
const profile = opt('profile', '')
const message = opt('message', '')
const aiHint = opt('ai', '')
const doSend = args.includes('--send')

const LIMIT_FILE = 'E:\\ai\\data\\xhs_dm_limit.json'
const sleep = ms => new Promise(r => setTimeout(r, ms))
const log = msg => console.log(`[${new Date().toLocaleTimeString('zh-CN', { hour12: false })}] ${msg}`)

if (!nick) {
  console.log(JSON.stringify({ ok: false, reason: '缺少 --nick' }))
  process.exit(1)
}

function checkLimit() {
  let state = { lastAt: 0, today: '', count: 0 }

  try {
    state = { ...state, ...JSON.parse(fs.readFileSync(LIMIT_FILE, 'utf-8')) }
  } catch {}

  const today = new Date().toLocaleDateString('sv-SE')

  if (state.today !== today) state = { lastAt: state.lastAt, today, count: 0 }

  const gap = Date.now() - state.lastAt

  if (gap < 30000) return { ok: false, reason: `发太快，等 ${Math.ceil((30000 - gap) / 1000)} 秒` }
  if (state.count >= 20) return { ok: false, reason: '今天已发 20 条，超上限' }

  return { ok: true, state }
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

async function genMessage(token) {
  const res = await fetch(`${API}/lead/genDmReply`, {
    method: 'POST',
    headers: { 'x-token': token, 'Content-Type': 'application/json' },
    body: JSON.stringify({ Nick: nick, LastMessage: aiHint || '第一次打招呼，想问问鸡蛋批发', History: '' , Channel: 'xhs'})
  })
  const json = await res.json()

  if (json.code !== 0) throw new Error(json.msg)

  return json.data.Reply
}

const token = await login()
const text = message || (aiHint ? await genMessage(token) : '')

if (!text) {
  console.log(JSON.stringify({ ok: false, reason: '没有内容可发' }))
  process.exit(1)
}

const browser = await chromium.connectOverCDP(process.env.CDP || 'http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = await ctx.newPage()

if (!profile) {
  console.log(JSON.stringify({ ok: false, reason: '缺少 --profile（需要带 xsec_token 的主页链接）' }))
  await page.close()
  process.exit(1)
}

await page.goto(`https://www.xiaohongshu.com${profile}`, { waitUntil: 'domcontentloaded', timeout: 60000 })
await sleep(6000)

// 私信入口 = 主页头像旁的气泡图标（无文字），点击会在新标签页打开 chat?openUid=xxx
const imBtn = page.locator('.xhs-user-im-btn, button[class*="user-im"], [class*="im-btn"]').first()

if (!(await imBtn.count())) {
  console.log(JSON.stringify({
    ok: false,
    reason: 'no_dm_entry',
    nick,
    note: '该用户主页没有私信按钮（可能关闭了私信）'
  }))
  await page.close()
  process.exit(4)
}

const before = new Set(ctx.pages().map(p => p.url()))

await imBtn.click()
await sleep(5000)

// 找到新打开的会话页
let chatPage = ctx.pages().find(p => p.url().includes('openUid') && !before.has(p.url()))
  || ctx.pages().find(p => p.url().includes('openUid'))

if (!chatPage) {
  // 有些情况是当前页直接跳转
  chatPage = page
}

await chatPage.waitForTimeout(2500)
log(`会话页：${chatPage.url().slice(0, 70)}`)

const input = chatPage.locator('.xhs-im-input-bar-editor, [contenteditable="true"]').first()

if (!(await input.count())) {
  console.log(JSON.stringify({ ok: false, reason: '会话页没有输入框' }))
  await chatPage.close()
  await page.close()
  process.exit(5)
}

await input.click()
await sleep(300)
await chatPage.keyboard.type(text, { delay: 12 })
await sleep(600)

const typed = await chatPage.evaluate(() => {
  const el = document.querySelector('.xhs-im-input-bar-editor, [contenteditable="true"]')

  return el ? (el.innerText || '').replace(/\u200b/g, '').trim() : ''
})

if (!typed) {
  console.log(JSON.stringify({ ok: false, reason: '输入失败' }))
  await chatPage.close()
  await page.close()
  process.exit(6)
}

if (!doSend) {
  await chatPage.keyboard.press('Control+A')
  await chatPage.keyboard.press('Delete')
  console.log(JSON.stringify({ ok: true, dryRun: true, nick, message: text, chatUrl: chatPage.url().slice(0, 80) }))
  await chatPage.close()
  await page.close()
  process.exit(0)
}
const limit = checkLimit()

if (!limit.ok) {
  await page.keyboard.press('Control+A')
  await page.keyboard.press('Delete')
  console.log(JSON.stringify({ ok: false, reason: limit.reason }))
  await page.close()
  process.exit(7)
}

await chatPage.keyboard.press('Enter')
await sleep(2500)

const sent = await chatPage.evaluate(m => document.body.innerText.includes(m.slice(0, 12)), text)

if (sent) {
  limit.state.lastAt = Date.now()
  limit.state.count = (limit.state.count || 0) + 1
  fs.writeFileSync(LIMIT_FILE, JSON.stringify(limit.state), 'utf-8')
}

console.log(JSON.stringify({ ok: sent, sent, nick, message: text }))
await chatPage.close()
await page.close()
