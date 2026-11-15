import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

await page.goto('https://www.douyin.com/user/self?showTab=following', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(9000)
for (let i = 0; i < 4; i++) { await page.mouse.wheel(0, 700); await page.waitForTimeout(1500) }

const data = await page.evaluate(() => {
  const body = document.body.innerText
  const users = []
  for (const a of document.querySelectorAll('a[href*="/user/MS4w"]')) {
    const t = (a.innerText || '').trim().split('\n')[0]
    if (t) users.push({ name: t, href: a.getAttribute('href').split('?')[0] })
  }
  const uniq = [...new Map(users.map(u => [u.href, u])).values()]
  const idx = body.indexOf('关注')
  return { url: location.href, 用户数: uniq.length, 列表: uniq.slice(0, 10), 正文片段: body.slice(idx, idx + 200) }
})
console.log(JSON.stringify(data, null, 2))
await page.screenshot({ path: 'E:\\ai\\shots\\dy-following3.png' })
await browser.close()
