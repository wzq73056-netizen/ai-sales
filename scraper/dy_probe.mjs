import { chromium } from 'playwright-core'

const browser = await chromium.launch({
  channel: 'chrome',
  headless: true,
  args: ['--disable-blink-features=AutomationControlled']
})
const ctx = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  locale: 'zh-CN',
  userAgent:
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36'
})
const page = await ctx.newPage()

const kw = encodeURIComponent('代理记账')
await page.goto(`https://www.douyin.com/search/${kw}?type=video`, { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(8000)

const info = await page.evaluate(() => {
  const text = document.body.innerText.slice(0, 600)
  const links = [...document.querySelectorAll('a[href*="/video/"]')].length
  const captcha = /验证|滑块|拖动|安全验证/.test(document.body.innerText)
  return { title: document.title, url: location.href, videoLinks: links, captcha, text }
})
console.log(JSON.stringify(info, null, 2))
await page.screenshot({ path: 'E:\\ai\\shots\\dy-probe.png' })
await browser.close()
