import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)
const info = await page.evaluate(() => {
  const entry = [...document.querySelectorAll('*')].find(el => (el.innerText || '').trim() === '消息' && el.getBoundingClientRect().width > 0)
  if (!entry) return { found: false }
  let box = entry
  for (let i = 0; i < 3 && box.parentElement; i++) box = box.parentElement
  return { found: true, html: box.outerHTML.slice(0, 900) }
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
