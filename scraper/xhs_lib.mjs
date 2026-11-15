/**
 * 小红书公共工具（对齐抖音那套）
 */
import { chromium } from 'playwright-core'
import { applyStealth } from './humanize.mjs'

// 账号隔离：小红书采集号独立浏览器（9334），不复用抖音的 9333
export const ENDPOINT = process.env.CDP || process.env.CDP_XHS || 'http://127.0.0.1:9334'

/**
 * 连接浏览器
 * fresh=true 时开一个独立标签页，避免和私信监听抢同一个页面
 */
export async function connect(fresh = false) {
  const browser = await chromium.connectOverCDP(ENDPOINT)
  const ctx = browser.contexts()[0]

  // 我们开出来的标签记在这里：browser.close() 只断开 CDP 连接，
  // 不关页面，所以每个脚本跑完都会在浏览器里留下一个标签页。
  const owned = []

  const openPage = async () => {
    const page = await ctx.newPage()

    // 每个新页面都先注入指纹补丁（必须在 goto 之前）
    await applyStealth(page)

    owned.push(page)

    return page
  }

  const originalClose = browser.close.bind(browser)

  browser.close = async () => {
    for (const page of owned) {
      await page.close().catch(() => {})
    }

    await originalClose()
  }

  if (fresh) {
    const page = await openPage()

    return { browser, page, ctx }
  }

  const existing = ctx.pages().find(p => p.url().includes('xiaohongshu.com'))
  const page = existing ?? (await openPage())

  return { browser, page, ctx }
}

/** 风控 / 未登录检测 */
export async function detectBlock(page) {
  return page.evaluate(() => {
    const text = document.body ? document.body.innerText : ''
    const url = location.href

    if (/captcha|verify/i.test(url)) return { blocked: true, reason: 'captcha' }
    if (/页面不见了|你访问的页面不存在/.test(text)) return { blocked: true, reason: 'not_found' }
    if (/访问频次异常|操作过于频繁|请稍后再试/.test(text)) return { blocked: true, reason: 'rate_limited' }
    if (text.includes('扫码登录') || text.includes('手机号登录')) return { blocked: true, reason: 'not_logged_in' }

    return { blocked: false, reason: '' }
  })
}

/** 小红书时间 → 天数（几天前），解析不了返回 -1 */
export function daysAgo(text) {
  const value = String(text || '').trim()

  const today = new Date()

  const md = value.match(/^(\d{1,2})-(\d{1,2})$/)

  if (md) {
    const month = Number(md[1])
    const day = Number(md[2])
    let year = today.getFullYear()
    const guess = new Date(year, month - 1, day)

    if (guess > today) year -= 1

    const diff = (today - new Date(year, month - 1, day)) / 86400000

    return Math.max(0, Math.round(diff))
  }

  const rel = value.match(/^(\d+)\s*(分钟|小时|天)前$/)

  if (rel) {
    const n = Number(rel[1])
    if (rel[2] === '分钟' || rel[2] === '小时') return 0

    return n
  }

  if (value === '昨天') return 1
  if (value === '前天') return 2

  return -1
}

/** 服务端推送线索 */
export async function pushLeads(api, token, leads) {
  let ok = 0
  let failed = 0

  for (const lead of leads) {
    const res = await fetch(`${api}/lead/createLead`, {
      method: 'POST',
      headers: { 'x-token': token, 'Content-Type': 'application/json' },
      body: JSON.stringify(lead)
    })
    const json = await res.json()

    if (json.code === 0) ok++
    else failed++
  }

  return { ok, failed }
}

export async function login(api) {
  const res = await fetch(`${api}/base/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: '123456', captcha: '', captchaId: '' })
  })
  const json = await res.json()

  if (json.code !== 0) throw new Error(`登录失败: ${json.msg}`)

  return json.data.token
}
