import { chromium } from 'playwright-core'
const OPENER = 'https://www.douyin.com/user/MS4wLjABAAAAvmHH8fB9Yy0gEjeQ3FR4pVP10bl032l9QUGejnuJ7Co'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

const stat = () => page.evaluate(() => ({
  url: location.href.slice(-40),
  rowsVisible: [...document.querySelectorAll('div[class*="conversationConversationItemwrapper"]')].filter(el => {
    const r = el.getBoundingClientRect()
    return r.width > 0 && r.left > 0 && r.right < window.innerWidth
  }).length,
  rowsAll: document.querySelectorAll('div[class*="conversationConversationItemwrapper"]').length,
  bubbles: document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]').length,
  dmBtn: [...document.querySelectorAll('button')].filter(b => b.innerText.trim() === '私信' && b.offsetParent).length
}))

console.log('初始:', JSON.stringify(await stat()))
await page.goto(OPENER, { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(5000)
console.log('导航后:', JSON.stringify(await stat()))

await page.keyboard.press('Escape')
await page.waitForTimeout(500)
const btn = page.locator('button', { hasText: '私信' }).first()
try { await btn.click({ force: true, timeout: 6000 }) } catch(e) { console.log('点击异常:', String(e).slice(0,50)) }
await page.waitForTimeout(4000)
console.log('点私信后:', JSON.stringify(await stat()))
await browser.close()
