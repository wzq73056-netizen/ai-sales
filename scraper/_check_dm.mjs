import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(2000)
const state = await page.evaluate(() => {
  const body = document.body.innerText
  const input = document.querySelector('div[contenteditable="true"], textarea')
  return {
    url: location.href,
    inputText: input ? (input.innerText || input.value || '').trim() : '(no input)',
    tail: body.slice(-400)
  }
})
console.log(JSON.stringify(state, null, 2))
try { await page.screenshot({ path: 'E:\\ai\\shots\\dy-dm-after.png', timeout: 15000 }) } catch(e) { console.log('截图失败:', String(e).slice(0,60)) }
await browser.close()
