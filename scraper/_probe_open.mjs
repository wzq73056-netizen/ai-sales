import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)
// 面板应该还开着；先确认列表在
const before = await page.evaluate(() => document.body.innerText.includes('上林弘康'))
console.log('列表可见:', before)

// 点击「上林弘康」那一行
const target = await page.evaluate(() => {
  const all = [...document.querySelectorAll('div, li')]
  const row = all.find(el => {
    const r = el.getBoundingClientRect()
    return r.x > 900 && r.height > 40 && r.height < 100 && (el.innerText || '').includes('上林弘康')
  })
  if (!row) return null
  const r = row.getBoundingClientRect()
  return { x: r.x + r.width / 2, y: r.y + r.height / 2, text: row.innerText.replace(/\s+/g, ' ').slice(0, 60) }
})
console.log('目标行:', JSON.stringify(target))

if (target) {
  await page.mouse.click(target.x, target.y)
  await page.waitForTimeout(4000)
  const conv = await page.evaluate(() => {
    const bubbles = [...document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]')].map(el => ({
      fromMe: /isFromMe/.test((el.className || '').toString()),
      text: (el.innerText || '').replace(/\s+/g, ' ').replace(/\s*已读$/, '').trim().slice(0, 60)
    }))
    return { count: bubbles.length, bubbles: bubbles.slice(-6) }
  })
  console.log('会话内容:', JSON.stringify(conv, null, 2))
  await page.screenshot({ path: 'E:\\ai\\shots\\dm-opened.png' })
}
await browser.close()
