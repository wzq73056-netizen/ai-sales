import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)
const rows = await page.evaluate(() => {
  const out = []
  for (const el of document.querySelectorAll('div[class*="conversationConversationItemwrapper"]')) {
    const r = el.getBoundingClientRect()
    if (!r.width) continue
    const badge = el.querySelector('[class*="badge"]')
    out.push({
      nick: (el.innerText||'').split('\n')[0],
      badgeCls: badge ? (badge.className||'').toString().slice(0,40) : '(none)',
      badgeText: badge ? (badge.innerText||'').trim() : '',
      unreadDot: !!el.querySelector('[class*="unread"], [class*="Unread"]')
    })
  }
  return out.slice(0, 4)
})
console.log(JSON.stringify(rows, null, 2))
await browser.close()
