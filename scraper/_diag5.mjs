import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

const stat = () => page.evaluate(() => ({
  rows: [...document.querySelectorAll('div[class*="conversationConversationItemwrapper"]')].filter(e => { const r = e.getBoundingClientRect(); return r.width > 0 && r.left > 0 }).length,
  bubbles: document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]').length,
  panel: !!document.querySelector('[class*="StackLayoutStackTitleBar"]')
}))

await page.goto('https://www.douyin.com/jingxuan', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(6000)
console.log('新页面:', JSON.stringify(await stat()))

const entry = page.locator('[data-e2e="im-entry"]').first()
console.log('入口存在:', await entry.count())

// 逐个候选点击目标试
const targets = [
  ['im-entry 整体', '[data-e2e="im-entry"]'],
  ['内部按钮', '[data-e2e="something-button"]'],
  ['图标 i', '[data-e2e="im-entry"] i']
]

for (const [name, sel] of targets) {
  const el = page.locator(sel).first()
  if (!(await el.count())) { console.log(name, '不存在'); continue }
  try {
    await el.click({ force: true, timeout: 5000 })
  } catch (e) {
    console.log(name, '点击异常', String(e).slice(0, 40))
  }
  await page.waitForTimeout(3500)
  console.log(name, '→', JSON.stringify(await stat()))
  if ((await stat()).rows > 0) break
}
await browser.close()
