/**
 * 连接专用 Edge（CDP），检查抖音页面状态。
 * 前提：Edge 已用 --remote-debugging-port=9333 启动。
 *
 * 用法：node dy_cdp.mjs
 * 只读：不点击、不输入、不发任何内容。
 */
import { chromium } from 'playwright-core'

const ENDPOINT = process.env.CDP || 'http://127.0.0.1:9333'

const browser = await chromium.connectOverCDP(ENDPOINT)
  // CDP 下 browser.close() 不关页面，装饰一下：退出时关掉本进程开的标签
  const _ownedPages = []
  const _newPage = browser.contexts()[0].newPage.bind(browser.contexts()[0])
  browser.contexts()[0].newPage = async () => {
    const p = await _newPage()
    _ownedPages.push(p)
    return p
  }
  const _close = browser.close.bind(browser)
  browser.close = async () => {
    for (const p of _ownedPages) await p.close().catch(() => {})
    await _close()
  }
const ctx = browser.contexts()[0]
const pages = ctx.pages()

console.log('页面数:', pages.length)
for (const p of pages) console.log(' -', p.url())

const page = pages.find(p => p.url().includes('douyin.com')) ?? pages[0]
if (!page) {
  console.log('没有可用的页面')
  await browser.close()
  process.exit(0)
}

await page.waitForTimeout(2000)

const state = await page.evaluate(() => {
  const text = document.body.innerText
  return {
    url: location.href,
    title: document.title,
    needLogin: /扫码登录|登录后即可/.test(text),
    hasFeed: document.querySelectorAll('a[href*="/video/"]').length,
    snippet: text.slice(0, 160)
  }
})

console.log(JSON.stringify(state, null, 2))
await page.screenshot({ path: 'E:\\ai\\shots\\dy-cdp.png' })

// 注意：不断开浏览器进程，只断开 CDP 连接
await browser.close()
