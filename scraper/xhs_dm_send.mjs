/**
 * 小红书主动私信（只发给白名单，默认仅冬冬瓜咚）
 *
 * 用法：
 *   node xhs_dm_send.mjs --nick 冬冬瓜咚 --message "你好" [--send]
 *   node xhs_dm_send.mjs --nick 冬冬瓜咚 --ai "咨询鸡蛋批发" [--send]
 */
import fs from 'node:fs'
import { connect, detectBlock, login } from './xhs_lib.mjs'

const API = process.env.API || 'http://127.0.0.1:8888'
const args = process.argv.slice(2)

const pick = flag => {
  const i = args.indexOf(flag)

  return i >= 0 ? args[i + 1] : ''
}

const nick = pick('--nick')
const profile = pick('--profile')
const message = pick('--message')
const aiHint = pick('--ai')
const doSend = args.includes('--send')

if (!nick) {
  console.log(JSON.stringify({ ok: false, reason: '缺少 --nick' }))
  process.exit(1)
}

const log = msg => console.log(`[${new Date().toLocaleTimeString('zh-CN', { hour12: false })}] ${msg}`)

// 频率限制：两次发送至少间隔 30 秒，每天最多 20 条
const LIMIT_FILE = 'E:\\ai\\data\\xhs_dm_limit.json'

function checkLimit() {
  let state = { lastAt: 0, today: '', count: 0 }

  try {
    state = { ...state, ...JSON.parse(fs.readFileSync(LIMIT_FILE, 'utf-8')) }
  } catch {}

  const today = new Date().toLocaleDateString('sv-SE')

  if (state.today !== today) state = { lastAt: state.lastAt, today, count: 0 }

  const gap = Date.now() - state.lastAt

  if (gap < 30000) return { ok: false, reason: `发太快了，等 ${Math.ceil((30000 - gap) / 1000)} 秒` }
  if (state.count >= 20) return { ok: false, reason: '今天已发 20 条，上限了' }

  return { ok: true, state }
}

function recordSend(state) {
  state.lastAt = Date.now()
  state.count = (state.count || 0) + 1
  fs.writeFileSync(LIMIT_FILE, JSON.stringify(state), 'utf-8')
}

/** 只允许发给白名单里的人 */
async function checkWhitelist(token) {
  const res = await fetch(`${API}/lead/getDouyinWhitelist?platform=xhs`, { headers: { 'x-token': token } })
  const json = await res.json()
  const list = json?.data?.list || []

  return list.find(item => (item.Nickname || '').includes(nick) && Number(item.Enabled) === 1)
}

async function genMessage(token, hint) {
  const res = await fetch(`${API}/lead/genDmReply`, {
    method: 'POST',
    headers: { 'x-token': token, 'Content-Type': 'application/json' },
    body: JSON.stringify({ Nick: nick, LastMessage: hint || '第一次打招呼，想了解鸡蛋批发价格', History: '' , Channel: 'xhs'})
  })
  const json = await res.json()

  if (json.code !== 0) throw new Error(json.msg)

  return json.data.Reply
}

const token = await login(API)
const allowed = await checkWhitelist(token)

if (!allowed) {
  console.log(JSON.stringify({ ok: false, reason: `${nick} 不在白名单，拒绝发送` }))
  process.exit(2)
}

const text = message || (aiHint ? await genMessage(token, aiHint) : '')

if (!text) {
  console.log(JSON.stringify({ ok: false, reason: '没有内容可发' }))
  process.exit(1)
}

const { browser, page } = await connect()

await page.goto('https://www.xiaohongshu.com/chat', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(6000)

const block = await detectBlock(page)
if (block.blocked) {
  console.log(JSON.stringify({ ok: false, reason: block.reason }))
  await browser.close()
  process.exit(3)
}

const index = await page.evaluate(nick => {
  const items = [...document.querySelectorAll('.xhs-im-conv-item')]

  return items.findIndex(el => (el.innerText || '').includes(nick))
}, nick)

if (index < 0) {
  // 网页版小红书：陌生人主页没有「私信」入口，只有已有会话才能发
  if (profile) {
    await page.goto(`https://www.xiaohongshu.com${profile}`, { waitUntil: 'domcontentloaded', timeout: 60000 })
    await page.waitForTimeout(5000)

    const hasDm = await page.evaluate(() => document.body.innerText.includes('私信'))

    console.log(JSON.stringify({
      ok: false,
      reason: 'no_dm_entry',
      hasDmButton: hasDm,
      note: '小红书网页版对陌生人没有私信入口，只能等对方先发消息'
    }))
  } else {
    console.log(JSON.stringify({ ok: false, reason: `会话列表里没有 ${nick}，需要先有对话` }))
  }

  await browser.close()
  process.exit(4)
}

await page.locator('.xhs-im-conv-item').nth(index).click()
await page.waitForTimeout(2500)

const editable = page.locator('.xhs-im-input-bar-wrap [contenteditable="true"]').first()

if (!(await editable.count())) {
  console.log(JSON.stringify({ ok: false, reason: '找不到输入框' }))
  await browser.close()
  process.exit(5)
}

await editable.click()
await page.waitForTimeout(300)
await page.keyboard.type(text, { delay: 12 })
await page.waitForTimeout(500)

const typed = await page.evaluate(() => {
  const el = document.querySelector('.xhs-im-input-bar-wrap [contenteditable="true"]')

  return el ? (el.innerText || '').trim() : ''
})

if (!typed) {
  console.log(JSON.stringify({ ok: false, reason: '输入失败' }))
  await browser.close()
  process.exit(6)
}

if (!doSend) {
  await page.keyboard.press('Control+A')
  await page.keyboard.press('Delete')
  console.log(JSON.stringify({ ok: true, dryRun: true, nick, message: text }))
  await browser.close()
  process.exit(0)
}

const limit = checkLimit()

if (!limit.ok) {
  console.log(JSON.stringify({ ok: false, reason: limit.reason }))
  await browser.close()
  process.exit(7)
}

await page.keyboard.press('Enter')
await page.waitForTimeout(2200)
recordSend(limit.state)

const sent = await page.evaluate(m => document.body.innerText.includes(m.slice(0, 12)), text)

console.log(JSON.stringify({ ok: sent, sent, nick, message: text }))
await browser.close()
