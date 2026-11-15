import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

await page.goto('https://www.douyin.com/user/self', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(6000)

// 点「关注」标签
const tab = page.locator('span, div', { hasText: /^关注$/ }).first()
if (await tab.count()) { await tab.click(); }
await page.waitForTimeout(4000)
for (let i = 0; i < 3; i++) { await page.mouse.wheel(0, 800); await page.waitForTimeout(1200) }

const data = await page.evaluate(() => {
  const body = document.body.innerText
  const counts = (body.match(/关注\s*(\d+)/) || [])[1]
  const fans = (body.match(/粉丝\s*(\d+)/) || [])[1]
  const users = []
  for (const a of document.querySelectorAll('a[href*="/user/MS4w"]')) {
    const t = (a.innerText || '').trim().split('\n')[0]
    if (t) users.push({ name: t, href: a.getAttribute('href').split('?')[0] })
  }
  const uniq = [...new Map(users.map(u => [u.href, u])).values()]
  return { url: location.href, 关注数: counts, 粉丝数: fans, 列表: uniq }
})
console.log(JSON.stringify(data, null, 2))
await page.screenshot({ path: 'E:\\ai\\shots\\dy-following2.png' })
await browser.close()
