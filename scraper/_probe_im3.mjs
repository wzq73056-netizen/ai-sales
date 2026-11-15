import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.goto('https://www.douyin.com/jingxuan', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(5000)

// 点消息图标本身（不是文字）
const icon = page.locator('[data-e2e="im-entry"] i').first()
if (await icon.count()) { await icon.click({ force: true }) } else { await page.locator('[data-e2e="im-entry"]').first().click({ force: true }) }
await page.waitForTimeout(4000)
const info = await page.evaluate(() => ({
  url: location.href,
  panelText: (document.querySelector('[class*="rightPanel"], [class*="RightPanel"], #IMpushListBoxId')?.innerText || '').replace(/\s+/g, ' ').slice(0, 200),
  bodyHas: /私信|消息/.test(document.body.innerText)
}))
console.log(JSON.stringify(info, null, 2))
await page.screenshot({ path: 'E:\\ai\\shots\\im-click.png' })
await browser.close()
