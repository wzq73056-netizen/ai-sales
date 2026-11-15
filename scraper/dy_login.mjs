/**
 * 抖音登录器：打开有头浏览器，人工扫码登录一次，保存 storageState 供后续任务复用。
 *
 * 用法：node dy_login.mjs
 * 产物：storage/douyin.json
 */
import { chromium } from 'playwright-core'
import fs from 'node:fs'
import path from 'node:path'

const OUT_DIR = path.join(process.cwd(), 'storage')
const STATE_FILE = path.join(OUT_DIR, 'douyin.json')

fs.mkdirSync(OUT_DIR, { recursive: true })

const browser = await chromium.launch({
  channel: 'chrome',
  headless: false,
  args: ['--disable-blink-features=AutomationControlled', '--start-maximized']
})

const ctx = await browser.newContext({
  viewport: null,
  locale: 'zh-CN',
  userAgent:
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36'
})

const page = await ctx.newPage()
await page.goto('https://www.douyin.com/', { waitUntil: 'domcontentloaded', timeout: 60000 })

console.log('请在弹出的浏览器里用抖音 App 扫码登录…（最多等 5 分钟）')

const deadline = Date.now() + 5 * 60 * 1000
let loggedIn = false

while (Date.now() < deadline) {
  await page.waitForTimeout(3000)

  // 登录成功特征：出现「我的」或头像相关元素，且页面不再要求扫码
  loggedIn = await page.evaluate(() => {
    const text = document.body.innerText
    const hasLoginBtn = /扫码登录|登录后即可/.test(text)
    return !hasLoginBtn && /我的|消息/.test(text)
  })

  if (loggedIn) break
}

if (!loggedIn) {
  console.log('超时未检测到登录，退出')
  await browser.close()
  process.exit(1)
}

await ctx.storageState({ path: STATE_FILE })
console.log('登录态已保存：', STATE_FILE)

// 简单验证：登录后能否搜索
await page.goto('https://www.douyin.com/search/' + encodeURIComponent('代理记账') + '?type=video', {
  waitUntil: 'domcontentloaded',
  timeout: 60000
})
await page.waitForTimeout(6000)

const result = await page.evaluate(() => ({
  url: location.href,
  videoLinks: document.querySelectorAll('a[href*="/video/"]').length,
  snippet: document.body.innerText.slice(0, 200)
}))
console.log(JSON.stringify(result, null, 2))

await page.screenshot({ path: path.join(OUT_DIR, 'after-login.png') })
await browser.close()
