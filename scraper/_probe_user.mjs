import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.goto('https://www.douyin.com/search/' + encodeURIComponent('代理记账') + '?type=user', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(7000)
const info = await page.evaluate(() => {
  const counts = {}
  for (const sel of ['[data-e2e="search-user-item"]','[data-e2e="user-card"]','a[href*="/user/"]','[class*="user-card"]']) {
    counts[sel] = document.querySelectorAll(sel).length
  }
  const sample = [...document.querySelectorAll('a[href*="/user/"]')].slice(0,3).map(a => ({
    href: a.getAttribute('href'),
    text: (a.closest('li')?.innerText || a.innerText || '').replace(/\s+/g,' ').slice(0,100)
  }))
  return { url: location.href, counts, sample }
})
console.log(JSON.stringify(info, null, 2))
await page.screenshot({ path: 'E:\\ai\\shots\\dy-user-search.png' })
await browser.close()
