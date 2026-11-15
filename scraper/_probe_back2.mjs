import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)
const info = await page.evaluate(() => {
  const out = []
  for (const el of document.querySelectorAll('*')) {
    const r = el.getBoundingClientRect()
    if (r.x < 880 || r.x > 1010 || r.y < 50 || r.y > 130) continue
    if (!r.width || r.width > 60) continue
    out.push({ tag: el.tagName.toLowerCase(), cls: (el.className||'').toString().slice(0,40), box: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)] })
  }
  return out
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
