import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.goto('https://www.douyin.com/jingxuan', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(5000)
await page.locator('[data-e2e="im-entry"]').first().hover()
await page.waitForTimeout(3500)
const info = await page.evaluate(() => {
  const box = document.querySelector('#IMpushListBoxId')
  const items = box ? [...box.querySelectorAll('*')].filter(e => e.children.length === 0 && (e.innerText || '').trim()).map(e => (e.innerText || '').trim().slice(0, 40)) : []
  return { visible: !!box && box.getBoundingClientRect().height > 0, texts: items.slice(0, 12), html: box ? box.innerHTML.slice(0, 400) : '' }
})
console.log(JSON.stringify(info, null, 2))
await page.screenshot({ path: 'E:\\ai\\shots\\im-hover.png' })
await browser.close()
