import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

for (const kw of ['134527749', '庸']) {
  await page.goto(`https://www.douyin.com/search/${encodeURIComponent(kw)}?type=user`, { waitUntil: 'domcontentloaded', timeout: 60000 })
  await page.waitForTimeout(6000)
  const info = await page.evaluate(() => {
    const out = []
    for (const a of document.querySelectorAll('a[href*="/user/MS4w"]')) {
      const t = (a.closest('li')?.innerText || a.innerText || '').replace(/\s+/g, ' ').slice(0, 80)
      if (t) out.push({ href: a.getAttribute('href').split('?')[0], text: t })
    }
    return out.slice(0, 5)
  })
  console.log('查询:', kw)
  console.log(JSON.stringify(info, null, 2))
  await page.screenshot({ path: `E:\\ai\\shots\\dy-search-${kw}.png` })
}
await browser.close()
