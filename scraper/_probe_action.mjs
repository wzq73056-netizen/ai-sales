import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1500)
const info = await page.evaluate(() => {
  const action = document.querySelector('[class*="inputAction"]')
  if (!action) return { found: false }
  const out = []
  for (const el of action.querySelectorAll('*')) {
    const r = el.getBoundingClientRect()
    if (!r.width || !r.height) continue
    out.push({
      tag: el.tagName.toLowerCase(),
      cls: (el.className || '').toString().slice(0, 60),
      box: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)],
      svg: !!el.querySelector('svg')
    })
  }
  return { found: true, actionBox: (() => { const r = action.getBoundingClientRect(); return [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)] })(), nodes: out.slice(-18) }
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
