import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.goto('https://www.douyin.com/user/self?showTab=following', { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(7000)
const info = await page.evaluate(() => {
  const text = document.body.innerText
  const links = [...document.querySelectorAll('a[href*="/user/"]')].map(a => ({ href: a.href, text: (a.innerText||'').replace(/\s+/g,' ').slice(0,40) }))
  return { url: location.href, needLogin: /扫码登录/.test(text), snippet: text.slice(0, 300), userLinks: links.slice(0, 15) }
})
console.log(JSON.stringify(info, null, 2))
await page.screenshot({ path: 'E:\\ai\\shots\\dy-following.png' })
await browser.close()
