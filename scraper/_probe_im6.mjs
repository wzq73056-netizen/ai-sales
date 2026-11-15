import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.goto('https://www.douyin.com/user/MS4wLjABAAAAvmHH8fB9Yy0gEjeQ3FR4pVP10bl032l9QUGejnuJ7Co', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(6000)

const box = await page.evaluate(() => {
  const el = document.querySelector('[data-e2e="im-entry"]')
  if (!el) return null
  const r = el.getBoundingClientRect()
  return { x: r.x + r.width / 2, y: r.y + r.height / 2, badge: (el.querySelector('.semi-badge-count')?.innerText || '').trim() }
})
console.log('入口:', JSON.stringify(box))
if (box) {
  await page.mouse.move(box.x, box.y)
  await page.waitForTimeout(2500)
  await page.mouse.click(box.x, box.y)
  await page.waitForTimeout(5000)
  const after = await page.evaluate(() => ({
    url: location.href,
    dialogCount: document.querySelectorAll('[role="dialog"], .semi-modal, [class*="drawer"]').length,
    text: document.body.innerText.replace(/\s+/g, ' ').slice(0, 120)
  }))
  console.log(JSON.stringify(after, null, 2))
  await page.screenshot({ path: 'E:\\ai\\shots\\im-profile-click.png' })
}
await browser.close()
