import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(800)

const area = await page.evaluate(() => {
  const el = document.querySelector('[class*="StackLayoutStackTitleBarleftArea"]')
  if (!el) return null
  const r = el.getBoundingClientRect()
  const kids = [...el.querySelectorAll('*')].slice(0, 6).map(k => {
    const b = k.getBoundingClientRect()
    return { tag: k.tagName.toLowerCase(), cls: (k.className || '').toString().slice(0, 36), box: [Math.round(b.left), Math.round(b.top), Math.round(b.width), Math.round(b.height)] }
  })
  return { box: [Math.round(r.left), Math.round(r.top), Math.round(r.width), Math.round(r.height)], kids }
})
console.log('标题栏左区:', JSON.stringify(area))

if (area) {
  const [x, y, w, h] = area.box
  const before = await page.evaluate(() => [...document.querySelectorAll('div[class*="conversationConversationItemwrapper"]')].filter(e => { const r = e.getBoundingClientRect(); return r.width > 0 }).length)
  await page.mouse.click(x + 15, y + h / 2)
  await page.waitForTimeout(2500)
  const after = await page.evaluate(() => ({
    rows: [...document.querySelectorAll('div[class*="conversationConversationItemwrapper"]')].filter(e => e.getBoundingClientRect().width > 0).length,
    bubbles: document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]').length
  }))
  console.log('点击前 rows:', before, '点击后:', JSON.stringify(after))
}
await browser.close()
