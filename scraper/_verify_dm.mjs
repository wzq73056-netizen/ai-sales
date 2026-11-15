import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)
const info = await page.evaluate(() => {
  const text = document.body.innerText
  const lines = text.split('\n').filter(Boolean)
  const idx = lines.findIndex(l => l.includes('李老板，在吗'))
  return { found: idx >= 0, context: idx >= 0 ? lines.slice(Math.max(0,idx-4), idx+3) : lines.slice(-10) }
})
console.log(JSON.stringify(info, null, 2))
try { await page.screenshot({ path: 'E:\\ai\\shots\\dy-dm-sent2.png', timeout: 20000 }) } catch(e) { console.log('截图超时') }
await browser.close()
