import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1500)
// 向上滚动聊天区加载历史
for (let i = 0; i < 5; i++) {
  await page.mouse.move(1050, 600)
  await page.mouse.wheel(0, -800)
  await page.waitForTimeout(1200)
}
const info = await page.evaluate(() => {
  const out = []
  for (const el of document.querySelectorAll('div')) {
    const c = (el.className || '').toString()
    if (!/messageMessageBox/i.test(c)) continue
    const r = el.getBoundingClientRect()
    const t = (el.innerText || '').replace(/\s+/g, ' ').trim()
    if (!t) continue
    out.push({ cls: c.slice(0, 60), x: Math.round(r.x), t: t.slice(0, 36) })
  }
  return out.slice(-14)
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
