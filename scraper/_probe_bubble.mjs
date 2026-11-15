import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.goto('https://www.douyin.com/user/MS4wLjABAAAAvmHH8fB9Yy0gEjeQ3FR4pVP10bl032l9QUGejnuJ7Co', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(6000)
const dm = page.locator('button', { hasText: '私信' }).first()
if (await dm.count()) await dm.click()
await page.waitForTimeout(5000)
const info = await page.evaluate(() => {
  const nodes = [...document.querySelectorAll('[class*="messageMsg"], [class*="msgItem"], [class*="message-item"]')]
    .filter(el => el.offsetParent !== null)
  const sample = nodes.slice(-8).map(el => {
    const r = el.getBoundingClientRect()
    return { cls: (el.className || '').toString().slice(0, 70), text: (el.innerText || '').replace(/\s+/g, ' ').slice(0, 40), x: Math.round(r.x) }
  })
  return { count: nodes.length, sample }
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
