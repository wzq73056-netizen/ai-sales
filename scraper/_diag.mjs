import { chromium } from 'playwright-core'
const OPENER = 'https://www.douyin.com/user/MS4wLjABAAAAvmHH8fB9Yy0gEjeQ3FR4pVP10bl032l9QUGejnuJ7Co'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())

await page.goto(OPENER, { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(5000)
const btn = page.locator('button', { hasText: '私信' }).first()
console.log('私信按钮:', await btn.count())
await btn.click()
await page.waitForTimeout(4000)

const s1 = await page.evaluate(() => ({
  rows: document.querySelectorAll('div[class*="conversationConversationItemwrapper"]').length,
  bubbles: document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]').length,
  body: document.body.innerText.replace(/\s+/g,' ').slice(0, 120)
}))
console.log('点私信后:', JSON.stringify(s1))
await page.screenshot({ path: 'E:\\ai\\shots\\step1.png' })

// 点返回
await page.mouse.click(await page.evaluate(() => window.innerWidth - 325), 90)
await page.waitForTimeout(3000)
const s2 = await page.evaluate(() => ({
  rows: document.querySelectorAll('div[class*="conversationConversationItemwrapper"]').length,
  bubbles: document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]').length
}))
console.log('点返回后:', JSON.stringify(s2))
await page.screenshot({ path: 'E:\\ai\\shots\\step2.png' })
await browser.close()
