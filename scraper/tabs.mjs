/**
 * 浏览器标签页卫生（抖音/小红书共用）
 *
 * 背景：脚本用 connectOverCDP 连的是用户正在用的 Edge。
 * Playwright 的 browser.close() 在 CDP 连接下只断开连接，**不会关闭页面**，
 * 于是每跑一次采集/私信，浏览器里就多留一个标签页。
 *
 * 这里提供两件事：
 *   trackPages(ctx)  记录本进程开的标签，close 时自动关掉
 *   tidyTabs(ctx)    清理历史残留标签（保留监听用的那几个）
 */
import { chromium } from 'playwright-core'

export const ENDPOINT = process.env.CDP || 'http://127.0.0.1:9333'

/** 监听脚本长期占用的页面，不能关 */
const KEEP_URLS = ['douyin.com/jingxuan', 'douyin.com/chat', 'xiaohongshu.com/chat']

function isKeepPage(page) {
  const url = page.url()

  return KEEP_URLS.some(item => url.includes(item))
}

/**
 * 连接浏览器，并让 browser.close() 顺手关掉本进程开的标签
 */
export async function connectTracked() {
  const browser = await chromium.connectOverCDP(ENDPOINT)
  const ctx = browser.contexts()[0]
  const owned = []

  const newPage = async () => {
    const page = await ctx.newPage()

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

  return { browser, ctx, newPage }
}

/**
 * 清理残留标签：同域名只保留监听页 + 最新一个，其余关掉
 */
export async function tidyTabs(ctx, { douyin = 1, xhs = 1 } = {}) {
  const closed = []
  const pages = ctx.pages()

  const dyPages = pages.filter(p => p.url().includes('douyin.com') && !isKeepPage(p))
  const xhsPages = pages.filter(p => p.url().includes('xiaohongshu.com') && !isKeepPage(p))

  for (const page of dyPages.slice(0, Math.max(0, dyPages.length - douyin))) {
    await page.close().catch(() => {})

    closed.push(page.url())
  }

  for (const page of xhsPages.slice(0, Math.max(0, xhsPages.length - xhs))) {
    await page.close().catch(() => {})

    closed.push(page.url())
  }

  // 空白页也一并收掉
  for (const page of pages.filter(p => p.url() === 'about:blank')) {
    await page.close().catch(() => {})

    closed.push('about:blank')
  }

  return closed
}
