import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

await page.goto('https://www.douyin.com/user/MS4wLjABAAAAvmHH8fB9Yy0gEjeQ3FR4pVP10bl032l9QUGejnuJ7Co', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(6000)
const dm = page.locator('button', { hasText: '私信' }).first()
if (await dm.count()) await dm.click()
await page.waitForTimeout(4000)

const info = await page.evaluate(() => {
  const buttons = [...document.querySelectorAll('button')]
    .filter(b => b.offsetParent !== null)
    .map(b => {
      const r = b.getBoundingClientRect()
      return {
        text: (b.innerText || '').trim().slice(0, 10),
        e2e: b.getAttribute('data-e2e') || '',
        cls: (b.className || '').toString().slice(0, 60),
        title: b.getAttribute('title') || '',
        aria: b.getAttribute('aria-label') || '',
        box: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)]
      }
    })
  return { total: buttons.length, buttons: buttons.slice(-14) }
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
