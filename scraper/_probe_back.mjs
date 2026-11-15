import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)
const info = await page.evaluate(() => {
  const panel = document.querySelector('[class*="RightPanel"], [class*="rightPanel"]')
  if (!panel) return { panel: false }
  const r = panel.getBoundingClientRect()
  // 面板顶部区域的元素
  const top = []
  for (const el of panel.querySelectorAll('*')) {
    const b = el.getBoundingClientRect()
    if (b.y > r.y + 60 || b.width < 10 || b.width > 60) continue
    top.push({ tag: el.tagName.toLowerCase(), cls: (el.className||'').toString().slice(0,40), box: [Math.round(b.x), Math.round(b.y), Math.round(b.width)] })
  }
  return { panel: true, panelBox: [Math.round(r.x), Math.round(r.y), Math.round(r.width)], top: top.slice(0, 8) }
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
