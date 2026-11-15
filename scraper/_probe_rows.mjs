import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)
const rows = await page.evaluate(() => {
  const out = []
  const seenY = new Set()
  for (const el of document.querySelectorAll('div, li')) {
    const r = el.getBoundingClientRect()
    if (r.x < 900 || r.width < 180 || r.height < 50 || r.height > 95) continue
    const imgs = el.querySelectorAll('img')
    if (imgs.length < 1 || imgs.length > 2) continue          // 只有头像
    const kids = el.children.length
    if (kids > 6) continue                                    // 排除整个列表容器
    const text = (el.innerText || '').replace(/\s+/g, ' ').trim()
    if (text.length < 4) continue
    const bucket = Math.round(r.y / 12)
    if (seenY.has(bucket)) continue
    seenY.add(bucket)
    const badge = el.querySelector('[class*="badge"]')
    out.push({ y: Math.round(r.y), x: Math.round(r.x + r.width/2), cy: Math.round(r.y + r.height/2), text: text.slice(0, 46), unread: badge ? (badge.innerText||'').trim() : '' })
  }
  return out.sort((a,b) => a.y - b.y)
})
console.log(JSON.stringify(rows, null, 2))
await browser.close()
