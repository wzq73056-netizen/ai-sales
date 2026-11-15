import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)

// 当前应该在某个会话详情；找左上角可点击元素
const info = await page.evaluate(() => {
  const out = []
  for (const el of document.querySelectorAll('*')) {
    const r = el.getBoundingClientRect()
    if (r.left < 880 || r.left > 980 || r.top < 55 || r.top > 120) continue
    if (!r.width || r.width > 70 || r.height > 45) continue
    const cs = getComputedStyle(el)
    out.push({
      tag: el.tagName.toLowerCase(),
      cls: (el.className || '').toString().slice(0, 42),
      box: [Math.round(r.left), Math.round(r.top), Math.round(r.width), Math.round(r.height)],
      cursor: cs.cursor,
      text: (el.innerText || '').trim().slice(0, 10)
    })
  }
  return { viewport: [window.innerWidth, window.innerHeight], items: out }
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
