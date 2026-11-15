/**
 * 抖音只读采集器（不发送任何内容）
 *
 * 流程：连接专用 Edge（CDP）→ 按关键词打开搜索页 → 滚动加载 → 抽取视频卡片
 *
 * 用法：node dy_collect.mjs 代理记账
 * 产物：E:\ai\data\dy_<keyword>.json
 *
 * 安全边界：
 *   - 只读 DOM，不点击评论框、不输入、不发送；
 *   - 每个关键词最多滚动 5 次，请求间隔 1.5~3 秒随机；
 */
import { chromium } from 'playwright-core'
import fs from 'node:fs'
import { detectBlock } from './dy_lib.mjs'
import path from 'node:path'
import { filterNewVideos, SEEN_VIDEOS } from './dy_lib.mjs'
import { applyStealth, humanDelay, humanType, takeQuota } from './humanize.mjs'

const ENDPOINT = process.env.CDP || 'http://127.0.0.1:9333'
const keyword = process.argv[2] || '代理记账'
const OUT_DIR = 'E:\\ai\\data'
const SCROLLS = Number(process.env.SCROLLS || 5)

fs.mkdirSync(OUT_DIR, { recursive: true })

const sleep = ms => new Promise(r => setTimeout(r, ms))
const jitter = () => sleep(1500 + Math.random() * 1500)

const browser = await chromium.connectOverCDP(ENDPOINT)
  // CDP 下 browser.close() 不关页面，装饰一下：退出时关掉本进程开的标签
  const _ownedPages = []
  const _newPage = browser.contexts()[0].newPage.bind(browser.contexts()[0])
  browser.contexts()[0].newPage = async () => {
    const p = await _newPage()

    // 反检测注入（必须在 goto 之前）
    await applyStealth(p)
    _ownedPages.push(p)
    return p
  }
  const _close = browser.close.bind(browser)
  browser.close = async () => {
    for (const p of _ownedPages) await p.close().catch(() => {})
    await _close()
  }
const ctx = browser.contexts()[0]
let page = ctx.pages().find(p => p.url().includes('douyin.com'))

if (!page) {
  page = await ctx.newPage()
}

const searchUrl = `https://www.douyin.com/search/${encodeURIComponent(keyword)}?type=video`
await page.goto(searchUrl, { waitUntil: 'domcontentloaded', timeout: 60000 })
await humanDelay('read')
await page.waitForTimeout(4000)

const loginState = await page.evaluate(() => ({
  needLogin: /扫码登录|登录后即可/.test(document.body.innerText),
  url: location.href
}))

// 风控/验证码检测：命中就明确失败，避免"采集 0 条但显示成功"
const guard = await detectBlock(page)
if (guard.blocked) {
  console.log(JSON.stringify({ ok: false, keyword, reason: guard.reason, url: guard.url, sample: guard.text }, null, 2))
  await browser.close()
  process.exit(3)
}

if (loginState.needLogin) {
  console.log(JSON.stringify({ ok: false, reason: '未登录，无法搜索', ...loginState }, null, 2))
  await browser.close()
  process.exit(2)
}

for (let i = 0; i < SCROLLS; i++) {
  await page.mouse.wheel(0, 1200 + Math.random() * 600)
  await jitter()
}

const items = await page.evaluate(() => {
  const out = []
  const seen = new Set()

  for (const a of document.querySelectorAll('a[href*="/video/"]')) {
    const href = a.href
    const id = (href.match(/\/video\/(\d+)/) || [])[1]
    if (!id || seen.has(id)) continue
    seen.add(id)

    const card = a.closest('li') ?? a.parentElement
    const text = (card?.innerText ?? a.innerText ?? '').replace(/\s+/g, ' ').trim()

    if (!takeQuota('scrape', 'douyin')) { console.log('今日抖音抓取配额已满，停止'); break }
    out.push({ id, url: href, text: text.slice(0, 200) })
  }

  return out.slice(0, 60)
})

const payload = { keyword, collectedAt: new Date().toISOString(), count: items.length, items }
const file = path.join(OUT_DIR, `dy_${keyword}.json`)
fs.writeFileSync(file, JSON.stringify(payload, null, 2), 'utf-8')

// 视频去重：全局记录已采集过的视频，避免重复处理
const fresh = filterNewVideos(items)

console.log(
  JSON.stringify(
    {
      ok: true,
      keyword,
      total: items.length,
      fresh: fresh.length,
      alreadySeen: items.length - fresh.length,
      seenStore: SEEN_VIDEOS,
      file,
      freshSample: fresh.slice(0, 3)
    },
    null,
    2
  )
)

await page.screenshot({ path: `E:\\ai\\shots\\dy-${keyword}.png` })
await browser.close()
