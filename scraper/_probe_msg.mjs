import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.goto('https://www.douyin.com/messages', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(7000)
const info = await page.evaluate(() => {
  const body = document.body.innerText.slice(0, 600)
  const items = [...document.querySelectorAll('[class*="conversation"], [class*="session"], [class*="chat-item"]')]
    .slice(0, 6)
    .map(el => ({ cls: (el.className || '').toString().slice(0, 50), text: (el.innerText || '').replace(/\s+/g, ' ').slice(0, 60) }))
  return { url: location.href, snippet: body.replace(/\s+/g, ' ').slice(0, 200), items }
})
console.log(JSON.stringify(info, null, 2))
await page.screenshot({ path: 'E:\\ai\\shots\\dm-list.png' })
await browser.close()
