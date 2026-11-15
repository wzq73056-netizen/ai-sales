import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)

const badge = await page.evaluate(() => {
  const el = document.querySelector('[data-e2e="im-entry"] .semi-badge-count')
  return el ? el.innerText.trim() : ''
})
console.log('角标:', JSON.stringify(badge))

// 点开消息入口，看是否有会话浮层
const entry = page.locator('[data-e2e="im-entry"]').first()
if (await entry.count()) {
  await entry.click()
  await page.waitForTimeout(3000)
  const info = await page.evaluate(() => ({
    url: location.href,
    hasList: !!document.querySelector('#IMpushListBoxId, [class*="pushList"], [class*="conversation"]'),
    snippet: document.body.innerText.replace(/\s+/g, ' ').slice(0, 300)
  }))
  console.log(JSON.stringify(info, null, 2))
  await page.screenshot({ path: 'E:\\ai\\shots\\im-popup.png' })
}
await browser.close()
