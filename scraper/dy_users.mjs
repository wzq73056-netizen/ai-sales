/**
 * 账号获客（只读）：搜索「用户」→ 抽取账号卡片 → 打分 → 本地合并 + 推送后端
 *
 * 用法：node dy_users.mjs <关键词> [--limit 20]
 * 产物：E:\ai\data\dy_users_<kw>.json；同时写入 ai_douyin_account
 *
 * 只读：不点关注、不发私信、不进主页。
 */
import { chromium } from 'playwright-core'
import fs from 'node:fs'
import path from 'node:path'
import { mergeLeads, ensureDataDir, detectBlock } from './dy_lib.mjs'
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
const keyword = args.find(a => !a.startsWith('--'))
const limitIdx = args.indexOf('--limit')
const limit = limitIdx >= 0 ? Number(args[limitIdx + 1]) : 20
const API = process.env.API || 'http://127.0.0.1:8888'

if (!keyword) {
  console.log('用法: node dy_users.mjs <关键词> [--limit 20]')
  process.exit(1)
}

ensureDataDir()
const sleep = ms => new Promise(r => setTimeout(r, ms))

/** "1.2万" → 12000 */
function parseFans(text) {
  if (!text) return 0
  const m = String(text).match(/([\d.]+)\s*(万|亿)?/)
  if (!m) return 0
  const n = Number(m[1])
  if (m[2] === '万') return Math.round(n * 10000)
  if (m[2] === '亿') return Math.round(n * 100000000)
  return Math.round(n)
}

const browser = trackBrowser(await chromium.connectOverCDP(process.env.CDP || 'http://127.0.0.1:9333'))
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

await page.goto(`https://www.douyin.com/search/${encodeURIComponent(keyword)}?type=user`, {
  waitUntil: 'domcontentloaded',
  timeout: 60000
})
await page.waitForTimeout(6000)

const guard = await detectBlock(page)
if (guard.blocked) {
  console.log(JSON.stringify({ ok: false, keyword, reason: guard.reason, url: guard.url, sample: guard.text }, null, 2))
  await browser.close()
  process.exit(3)
}

const needLogin = await page.evaluate(() => /扫码登录|登录后即可/.test(document.body.innerText))
if (needLogin) {
  console.log(JSON.stringify({ ok: false, reason: '未登录' }))
  await browser.close()
  process.exit(2)
}

for (let i = 0; i < 4; i++) {
  await page.mouse.wheel(0, 1000 + Math.random() * 500)
  await sleep(1500 + Math.random() * 1000)
}

const users = await page.evaluate(() => {
  const out = []
  const seen = new Set()

  for (const a of document.querySelectorAll('a[href*="/user/MS4w"]')) {
    const href = a.getAttribute('href') ?? ''
    const uid = (href.match(/\/user\/([^?]+)/) || [])[1]
    if (!uid || seen.has(uid)) continue
    seen.add(uid)

    const cardText = (a.closest('li')?.innerText || a.innerText || '').replace(/\s+/g, ' ').trim()
    if (!cardText) continue

    const nickname = (a.innerText || '').trim().split('\n')[0] || cardText.slice(0, 20)
    const douyinId = (cardText.match(/抖音号[:：]\s*([^\s]+)/) || [])[1] || ''
    const fansText =
      (cardText.match(/粉丝\s*[:：]?\s*([\d.]+[万亿]?)/) || [])[1] ||
      (cardText.match(/([\d.]+[万亿]?)\s*粉丝/) || [])[1] ||
      ''
    const bio = cardText.replace(/关注|私信|抖音号[:：][^\s]+/g, '').trim().slice(0, 120)

    out.push({ nickname, douyinId, fansText, bio, profile: href.split('?')[0] })
  }

  return out
})

const COMPETITOR = /财税|代账|代理记账|会计|企服|企业管理|财务咨询|代办|事务所|记账公司|注册公司|企业服务/

const leads = []
const accounts = []

for (const u of users) {
  let score = 20
  const tags = ['账号获客']

  if (u.douyinId) score += 5
  if (u.fansText) score += 5
  if (u.bio) score += 5

  const isCompetitor = COMPETITOR.test(`${u.nickname} ${u.bio}`)
  if (isCompetitor) {
    score -= 30
    tags.push('疑似同行')
  }

  if (score <= 0) continue

  leads.push({
    author: u.nickname,
    profile: u.profile,
    time: '',
    region: '',
    days: null,
    content: `抖音号:${u.douyinId} 粉丝:${u.fansText} 简介:${u.bio}`,
    score: Math.min(score, 100),
    tags
  })

  accounts.push({
    Keyword: keyword,
    Nickname: u.nickname,
    DouyinId: u.douyinId,
    Fans: parseFans(u.fansText),
    Bio: u.bio,
    ProfileUrl: u.profile,
    IsCompetitor: isCompetitor ? 1 : 0,
    Score: Math.min(score, 100),
    Status: 0
  })
}

leads.sort((a, b) => b.score - a.score)
const limitedLeads = leads.slice(0, limit)
const keepProfiles = new Set(limitedLeads.map(l => l.profile))
const limitedAccounts = accounts.filter(a => keepProfiles.has(a.ProfileUrl)).slice(0, limit)

const file = path.join('E:\\ai\\data', `dy_users_${keyword}.json`)
fs.writeFileSync(
  file,
  JSON.stringify(
    {
      keyword,
      mode: 'account',
      collectedAt: new Date().toISOString(),
      found: users.length,
      kept: limitedAccounts.length,
      accounts: limitedAccounts
    },
    null,
    2
  ),
  'utf-8'
)

const merged = mergeLeads(limitedLeads, keyword)

let pushed = null
try {
  const loginRes = await fetch(`${API}/base/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: '123456', captcha: '', captchaId: '' })
  })
  const loginJson = await loginRes.json()

  if (loginJson.code === 0) {
    const res = await fetch(`${API}/lead/saveDouyinAccounts`, {
      method: 'POST',
      headers: { 'x-token': loginJson.data.token, 'Content-Type': 'application/json' },
      body: JSON.stringify({ Accounts: limitedAccounts })
    })
    pushed = await res.json()
    // 顺便同步进统一线索池
    const syncRes = await fetch(`${API}/lead/syncAccountsToLeads`, {
      method: 'POST',
      headers: { 'x-token': loginJson.data.token, 'Content-Type': 'application/json' }
    })
    pushed.synced = await syncRes.json()
  } else {
    pushed = loginJson
  }
} catch (e) {
  pushed = { ok: false, msg: String(e) }
}

console.log(
  JSON.stringify(
    { ok: true, keyword, found: users.length, kept: limitedAccounts.length, file, store: merged, pushed },
    null,
    2
  )
)
console.log(JSON.stringify(limitedAccounts.slice(0, 3), null, 2))

await browser.close()
