import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.goto('https://www.douyin.com/jingxuan', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(5000)

const box = await page.evaluate(() => {
  const el = document.querySelector('[data-e2e="im-entry"]')
  const r = el.getBoundingClientRect()
  return { x: r.x + r.width / 2, y: r.y + r.height / 2 }
})
console.log('消息入口坐标:', JSON.stringify(box))

await page.mouse.move(box.x, box.y)
await page.waitForTimeout(800)
await page.mouse.click(box.x, box.y)
await page.waitForTimeout(5000)

const after = await page.evaluate(() => ({
  url: location.href,
  panels: [...document.querySelectorAll('div')].filter(d => /panel|Panel|drawer|Drawer/i.test((d.className || '').toString()) && d.getBoundingClientRect().width > 300 && d.getBoundingClientRect().height > 300).length,
  text: document.body.innerText.replace(/\s+/g, ' ').slice(0, 150)
}))
console.log(JSON.stringify(after, null, 2))
await page.screenshot({ path: 'E:\\ai\\shots\\im-mouse.png' })
await browser.close()
