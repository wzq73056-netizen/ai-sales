import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.goto('https://www.douyin.com/', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(6000)
const info = await page.evaluate(() => {
  // 顶部导航的红点/未读数
  const badges = []
  for (const el of document.querySelectorAll('[class*="badge"], [class*="Badge"], [class*="dot"], [class*="count"]')) {
    const t = (el.innerText || '').trim()
    const r = el.getBoundingClientRect()
    if (r.width > 0 && r.width < 40) badges.push({ cls: (el.className || '').toString().slice(0, 40), text: t.slice(0, 6), x: Math.round(r.x), y: Math.round(r.y) })
  }
  // 消息入口
  const msgEntry = [...document.querySelectorAll('*')].find(el => (el.innerText || '').trim() === '消息' && el.getBoundingClientRect().width > 0)
  return { badgeCount: badges.length, badges: badges.slice(0, 8), hasMsgEntry: !!msgEntry }
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
