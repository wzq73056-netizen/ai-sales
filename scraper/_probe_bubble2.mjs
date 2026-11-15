import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)
const info = await page.evaluate(() => {
  const out = []
  for (const el of document.querySelectorAll('div')) {
    const r = el.getBoundingClientRect()
    if (r.x < 850 || r.width < 40 || r.height < 14) continue
    const t = (el.innerText || '').replace(/\s+/g, ' ').trim()
    if (!t || t.length > 60) continue
    out.push({ cls: (el.className || '').toString().slice(0, 55), x: Math.round(r.x), w: Math.round(r.width), t: t.slice(0, 40) })
  }
  return out.slice(-16)
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
