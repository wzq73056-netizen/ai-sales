/**
 * 小红书搜笔记（按关键词）
 *
 * 用法：node xhs_collect.mjs 鸡蛋批发 [--limit 20] [--sort 综合]
 * 输出：E:\ai\data\xhs_<关键词>.json
 */
import fs from 'node:fs'
import { humanDelay, scrollLikeHuman, takeQuota } from './humanize.mjs'
import { connect, detectBlock } from './xhs_lib.mjs'

const keyword = process.argv[2] || '鸡蛋批发'
const args = process.argv.slice(2)
const limit = Number((args[args.indexOf('--limit') + 1] || 20))

const { browser, page } = await connect(true)

const url = `https://www.xiaohongshu.com/search_result?keyword=${encodeURIComponent(keyword)}&source=web_search_result_notes`
await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 60000 })
await humanDelay('read')
await scrollLikeHuman(page, 2)
await page.waitForTimeout(5000)

const block = await detectBlock(page)
if (block.blocked) {
  console.error(JSON.stringify({ ok: false, reason: block.reason }))
  await browser.close()
  process.exit(3)
}

// 滚动加载
for (let i = 0; i < 3; i++) {
  await page.mouse.wheel(0, 1200)
  await page.waitForTimeout(1500)
}

const items = await page.evaluate(() => {
  const out = []

  for (const card of document.querySelectorAll('section.note-item, .note-item, [class*="note-item"]')) {
    const link = card.querySelector('a[href*="/search_result/"], a[href*="/explore/"]')
    if (!link) continue

    // 搜索结果里带 xsec_token 的链接才能直接打开笔记
    const allLinks = [...card.querySelectorAll('a[href*="/search_result/"]')]
    const tokenLink = allLinks.find(a => (a.getAttribute('href') || '').includes('xsec_token'))
    const href = (tokenLink ? tokenLink.getAttribute('href') : link.getAttribute('href')) || ''
    const id = (href.match(/([0-9a-f]{24})/) || [])[1] || ''
    if (!id) continue

    const text = (card.innerText || '').replace(/\s+/g, ' ').trim()
    const titleEl = card.querySelector('[class*="title"], .title')
    const authorEl = card.querySelector('[class*="author"], .author')
    const likeEl = card.querySelector('[class*="like"], .like')

    if (!takeQuota('scrape', 'xhs')) { console.log('今日小红书抓取配额已满，停止'); break }

    out.push({
      id,
      href,
      title: titleEl ? (titleEl.innerText || '').trim().slice(0, 80) : text.slice(0, 60),
      author: authorEl ? (authorEl.innerText || '').trim() : '',
      likes: likeEl ? (likeEl.innerText || '').trim() : '',
      text: text.slice(0, 120)
    })
  }

  return out
})

const seen = new Set()
const unique = items.filter(item => {
  if (seen.has(item.id)) return false
  seen.add(item.id)

  return true
})

const picked = unique.slice(0, limit)
const file = `E:\\ai\\data\\xhs_${keyword.replace(/[\\/:*?"<>|]/g, '_')}.json`

fs.writeFileSync(file, JSON.stringify({ keyword, count: picked.length, total: unique.length, items: picked, fetchedAt: new Date().toLocaleString('sv-SE') }, null, 2), 'utf-8')

console.log(JSON.stringify({ ok: true, keyword, count: picked.length, total: unique.length, file }))
await browser.close()
