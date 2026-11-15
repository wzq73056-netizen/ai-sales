import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.goto('https://www.douyin.com/video/7547901950851894585', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(6000)
const info = await page.evaluate(() => {
  const items = [...document.querySelectorAll('[data-e2e="comment-item"]')]
  const one = items[0]
  const kids = one ? [...one.querySelectorAll('*')].slice(0, 25).map(el => ({
    tag: el.tagName.toLowerCase(),
    cls: (el.className || '').toString().slice(0, 60),
    text: (el.innerText || '').replace(/\s+/g,' ').slice(0, 40)
  })) : []
  return { itemCount: items.length, firstItemText: one?.innerText, kids }
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
