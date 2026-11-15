/**
 * 视频评论只读探查：打开一个视频页，看看评论 DOM 长什么样。
 * 用法：node dy_probe_comments.mjs <videoId>
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


const id = process.argv[2]
if (!id) {
  console.log('用法: node dy_probe_comments.mjs <videoId>')
  process.exit(1)
}

const browser = trackBrowser(await chromium.connectOverCDP(process.env.CDP || 'http://127.0.0.1:9333'))
const ctx = browser.contexts()[0]
let page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

await page.goto(`https://www.douyin.com/video/${id}`, { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(6000)

const info = await page.evaluate(() => {
  const pick = sel => [...document.querySelectorAll(sel)].slice(0, 3).map(el => (el.innerText || '').replace(/\s+/g, ' ').slice(0, 120))

  const guesses = {}
  for (const sel of [
    '[data-e2e="comment-item"]',
    '[class*="comment-item"]',
    '[class*="CommentItem"]',
    '[class*="comment"]'
  ]) {
    guesses[sel] = document.querySelectorAll(sel).length
  }

  return {
    url: location.href,
    title: document.title.slice(0, 80),
    counts: guesses,
    samples: pick('[class*="comment"]').slice(0, 3),
    bodySnippet: document.body.innerText.slice(0, 300)
  }
})

console.log(JSON.stringify(info, null, 2))
await page.screenshot({ path: 'E:\\ai\\shots\\dy-comments-probe.png' })
await browser.close()
