import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)

const badge = await page.evaluate(() => {
  const el = document.querySelector('[data-e2e="im-entry"] .semi-badge-count')
  return el ? el.innerText.trim() : '(无角标)'
})
console.log('当前角标:', badge)
console.log('当前页:', page.url())
console.log('标签页数:', ctx.pages().length)

// 点击消息入口，看是否开新标签
const before = ctx.pages().length
await page.locator('[data-e2e="im-entry"]').first().click({ force: true }).catch(() => {})
await page.waitForTimeout(4000)
const after = ctx.pages()
console.log('点击后标签页数:', after.length, '新增:', after.length - before)
for (const p of after) console.log('  tab:', p.url().slice(0, 90))
