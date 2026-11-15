/**
 * 清理浏览器里堆下来的标签页
 *
 * 用法：node tabs_tidy.mjs
 * 保留：抖音监听页、小红书私信页、每平台各一个工作页
 */
import { chromium } from 'playwright-core'
import { ENDPOINT, tidyTabs } from './tabs.mjs'

const browser = await chromium.connectOverCDP(ENDPOINT)
const ctx = browser.contexts()[0]

try {
  const closed = await tidyTabs(ctx)

  console.log(JSON.stringify({ ok: true, closed: closed.length, urls: closed.slice(0, 10) }))
} catch (error) {
  console.log(JSON.stringify({ ok: false, error: String(error && error.message) }))
} finally {
  await browser.close().catch(() => {})
}
