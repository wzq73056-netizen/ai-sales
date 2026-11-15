import { chromium } from 'playwright-core'
// 一个我们从未私信过的陌生人主页
const stranger = 'https://www.douyin.com/user/MS4wLjABAAAAiR9EFsgiFfjU3dIGzCEYAUimakqs7irpWROIAlA2JNk'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.goto(stranger, { waitUntil: 'domcontentloaded', timeout: 60000 })
await page.waitForTimeout(6000)

const nick = await page.evaluate(() => document.title.replace('的抖音 - 抖音', ''))
const dmBtn = page.locator('button', { hasText: '私信' }).first()
const hasBtn = await dmBtn.count()
console.log('陌生人:', nick, '| 有私信按钮:', hasBtn > 0)

if (hasBtn) {
  await dmBtn.click()
  await page.waitForTimeout(4000)
  const state = await page.evaluate(() => {
    const input = document.querySelector('div[contenteditable="true"], textarea')
    const bubbles = document.querySelectorAll('div[class*="messageMessageBoxcontentBox"]').length
    return { inputReady: !!input, bubbles }
  })
  console.log('会话面板:', JSON.stringify(state))
  await page.screenshot({ path: 'E:\\ai\\shots\\stranger-dm.png' })
}
await browser.close()
