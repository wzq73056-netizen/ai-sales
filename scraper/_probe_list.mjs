import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)

const list = await page.evaluate(() => {
  // 会话列表在消息面板里，找带未读角标或昵称的行
  const rows = []
  const seen = new Set()
  for (const el of document.querySelectorAll('div, li')) {
    const r = el.getBoundingClientRect()
    if (r.x < 900 || r.width < 200 || r.height < 40 || r.height > 90) continue
    const t = (el.innerText || '').replace(/\s+/g, ' ').trim()
    if (!t || t.length < 3) continue
    if (seen.has(t)) continue
    seen.add(t)
    // 是否带未读角标
    const badge = el.querySelector('[class*="badge"]')
    rows.push({ t: t.slice(0, 50), unread: badge ? (badge.innerText || '').trim() : '', y: Math.round(r.y) })
  }
  return rows.slice(0, 22)
})
console.log(JSON.stringify(list, null, 2))
await browser.close()
