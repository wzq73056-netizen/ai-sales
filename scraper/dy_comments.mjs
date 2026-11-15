/**
 * 视频评论只读采集（带移动端同款过滤）
 *
 * 用法：
 *   node dy_comments.mjs <videoId> [--city 上海] [--days 90] [--limit 10]
 *
 * 过滤规则（对齐移动端 TiktokCommentAccess）：
 *   1. 跳过视频作者本人的评论
 *   2. 目标地区：评论 IP 必须包含 --city（不传则不限制）
 *   3. 时间窗口：评论超过 --days 天则跳过（0 表示不限制）
 *   4. 数量上限：每个视频最多 --limit 条（默认 10，对齐 maxcommentNum）
 *
 * 只读：不点赞、不关注、不发私信。
 */
import { chromium } from 'playwright-core'
import fs from 'node:fs'
import path from 'node:path'
import { parseCommentTime, ensureDataDir, detectBlock } from './dy_lib.mjs'
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
const videoId = args.find(a => !a.startsWith('--'))
const opt = (name, def) => {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? args[i + 1] : def
}

const city = opt('city', '')
const maxDays = Number(opt('days', 0))
const limit = Number(opt('limit', 10))
const excludeWords = opt('exclude', '')
  .split(/[,;，；]/)
  .map(s => s.trim())
  .filter(Boolean)

if (!videoId) {
  console.log('用法: node dy_comments.mjs <videoId> [--city 上海] [--days 90] [--limit 10]')
  process.exit(1)
}

ensureDataDir()
const sleep = ms => new Promise(r => setTimeout(r, ms))

const browser = trackBrowser(await chromium.connectOverCDP(process.env.CDP || 'http://127.0.0.1:9333'))
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

await page.goto(`https://www.douyin.com/video/${videoId}`, { waitUntil: 'domcontentloaded', timeout: 60000 })
await humanDelay('read')
await page.waitForTimeout(9000)

const guard = await detectBlock(page)
if (guard.blocked) {
  console.log(JSON.stringify({ ok: false, videoId, reason: guard.reason, url: guard.url, sample: guard.text }, null, 2))
  await browser.close()
  process.exit(3)
}

const videoTitle = await page.evaluate(() => {
  const sel = ['[data-e2e="video-desc"]', 'h1', '[data-e2e="detail-video-desc"]']

  for (const s of sel) {
    const el = document.querySelector(s)
    const t = (el?.innerText || '').trim()
    if (t) return t.slice(0, 200)
  }

  return document.title.replace(' - 抖音', '').slice(0, 200)
})

const authorOfVideo = await page.evaluate(() => {
  const blacklist = ['我的', '朋友', '关注', '推荐', '精选', '首页']
  const link = [...document.querySelectorAll('a[href*="/user/"]')].find(a => {
    const t = (a.innerText || '').trim()
    return t && t.length > 1 && !blacklist.includes(t)
  })
  return (link?.innerText || '').trim()
})

for (let i = 0; i < 6; i++) {
  const moved = await page.evaluate(() => {
    const item = document.querySelector('[data-e2e="comment-item"]')
    if (!item) return false

    let el = item.parentElement
    while (el) {
      if (el.scrollHeight > el.clientHeight + 50) {
        const before = el.scrollTop
        el.scrollTop += el.clientHeight
        return el.scrollTop !== before
      }
      el = el.parentElement
    }
    return false
  })

  if (!moved) break
  await sleep(1500 + Math.random() * 1200)
}

const raw = await page.evaluate(() => {
  const nodes = [...document.querySelectorAll('[data-e2e="comment-item"]')]
  const seen = new Set()
  const out = []
  const timeRe = /((\d+(秒|分钟|小时|天|周|月|年)前)|昨天|今天|(\d{1,2}-\d{1,2})|(\d{4}-\d{1,2}-\d{1,2}))([·•].*)?$/

  for (const node of nodes) {
    const profile = node.querySelector('a[href*="/user/"]')?.getAttribute('href') ?? ''
    const avatars = [...node.querySelectorAll('img')].map(i => i.src || '').filter(Boolean)
    const avatar = avatars.find(s => s.includes('aweme-avatar') || s.includes('300x300')) || avatars[0] || ''
    const lines = (node.innerText || '')
      .split('\n')
      .map(s => s.trim())
      .filter(Boolean)

    if (lines.length < 2) continue

    const author = lines[0]
    const timeIdx = lines.findIndex((l, i) => i > 0 && timeRe.test(l))
    const content = lines
      .slice(1, timeIdx > 0 ? timeIdx : undefined)
      .filter(l => l !== '...' && l !== '展开' && !/^\d+$/.test(l) && l !== '分享' && l !== '回复')
      .join(' ')
      .replace(/展开\d+条回复/g, '')
      .trim()

    if (!content) continue

    const key = profile || `${author}|${content}`
    if (seen.has(key)) continue
    seen.add(key)

    if (!takeQuota('scrape', 'douyin')) { console.log('今日抖音抓取配额已满，停止'); break }

    out.push({ author, profile, avatar, time: timeIdx > 0 ? lines[timeIdx] : '', content: content.slice(0, 300) })
  }

  return out
})

const stats = { raw: raw.length, skipAuthor: 0, skipRegion: 0, skipDays: 0, skipExclude: 0, kept: 0, truncated: 0 }
const kept = []

for (const c of raw) {
  if (authorOfVideo && c.author === authorOfVideo) {
    stats.skipAuthor++
    continue
  }

  const { days, region } = parseCommentTime(c.time)

  if (city && region && !region.includes(city)) {
    stats.skipRegion++
    continue
  }

  if (maxDays > 0 && days !== null && days > maxDays) {
    stats.skipDays++
    continue
  }

  if (excludeWords.length && excludeWords.some(w => c.content.includes(w))) {
    stats.skipExclude++
    continue
  }

  kept.push({ ...c, region, days })
}

const limited = kept.slice(0, limit)
stats.kept = limited.length
stats.truncated = Math.max(0, kept.length - limited.length)

const payload = {
  videoId,
  url: `https://www.douyin.com/video/${videoId}`,
  authorOfVideo,
  videoTitle,
  filters: { city, maxDays, limit, excludeWords },
  collectedAt: new Date().toISOString(),
  stats,
  comments: limited
}

const file = path.join('E:\\ai\\data', `dy_comments_${videoId}.json`)
fs.writeFileSync(file, JSON.stringify(payload, null, 2), 'utf-8')

console.log(JSON.stringify({ ok: true, videoId, authorOfVideo, stats, file }, null, 2))
console.log(JSON.stringify(limited.slice(0, 4), null, 2))

await browser.close()
