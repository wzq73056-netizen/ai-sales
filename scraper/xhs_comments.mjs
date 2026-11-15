/**
 * 小红书笔记评论采集
 *
 * 用法：node xhs_comments.mjs <笔记id> [--days 30] [--limit 50] [--exclude 同行,招聘]
 * 输出：E:\ai\data\xhs_comments_<id>.json
 *
 * 关键点：小红书笔记页必须带 xsec_token，直接访问会「页面不见了」，
 * 所以统一从搜索结果点进去。
 */
import fs from 'node:fs'
import { humanDelay, scrollLikeHuman, takeQuota } from './humanize.mjs'
import { connect, detectBlock, daysAgo } from './xhs_lib.mjs'

const noteId = process.argv[2]
if (!noteId) {
  console.error('缺少笔记 id')
  process.exit(1)
}

const args = process.argv.slice(2)
const maxDays = Number((args[args.indexOf('--days') + 1] || 30))
const limit = Number((args[args.indexOf('--limit') + 1] || 50))
const exclude = (args[args.indexOf('--exclude') + 1] || '')
  .split(/[,;]/)
  .map(s => s.trim())
  .filter(Boolean)

const keyword = process.env.XHS_KEYWORD || ''
const { browser, page } = await connect(true)

const hrefArg = args[args.indexOf('--href') + 1] || ''

// 优先用采集时存下来的带 token 链接
if (hrefArg && hrefArg.includes('xsec_token')) {
  await page.goto(`https://www.xiaohongshu.com${hrefArg}`, { waitUntil: 'domcontentloaded', timeout: 60000 })
await humanDelay('read')
await scrollLikeHuman(page, 2)
  await page.waitForTimeout(4500)
} else if (keyword) {
  await page.goto(`https://www.xiaohongshu.com/search_result?keyword=${encodeURIComponent(keyword)}`, { waitUntil: 'domcontentloaded', timeout: 60000 })
await humanDelay('read')
await scrollLikeHuman(page, 3)
  await page.waitForTimeout(3500)

  const clicked = await page.evaluate(id => {
    const links = [...document.querySelectorAll('a[href*="' + id + '"]')]
    const link = links.find(a => (a.getAttribute('href') || '').includes('/search_result/')) || links[0]

    if (!link) return false

    link.click()

    return true
  }, noteId)

  if (!clicked) {
    console.log(JSON.stringify({ ok: false, reason: 'note_not_in_search' }))
    await browser.close()
    process.exit(4)
  }
} else {
  await page.goto(`https://www.xiaohongshu.com/explore/${noteId}`, { waitUntil: 'domcontentloaded', timeout: 60000 })
}

await page.waitForTimeout(4500)

let block = await detectBlock(page)
if (block.blocked) {
  console.log(JSON.stringify({ ok: false, reason: block.reason }))
  await browser.close()
  process.exit(3)
}

// 往下滚两屏，加载更多评论
for (let i = 0; i < 3; i++) {
  await page.mouse.wheel(0, 900)
  await page.waitForTimeout(1200)
}

// 笔记作者（发笔记的商家/供应商，也算线索）
const noteAuthor = await page.evaluate(() => {
  const pick = sel => { const el = document.querySelector(sel); return el ? (el.innerText || '').replace(/\s+/g, ' ').trim() : '' }
  const link = document.querySelector('.author-container a[href*="/user/profile/"], .author-wrapper a[href*="/user/profile/"]')

  return {
    name: pick('.author-container .username') || pick('.author-wrapper .name') || pick('[class*="username"]'),
    profile: link ? link.getAttribute('href') : '',
    title: pick('#detail-title') || pick('[class*="title"]')
  }
})

const raw = await page.evaluate(() => {
  const out = []
  const nodes = document.querySelectorAll('.comment-item, [class*="comment-item"]')

  for (const node of nodes) {
    const pick = sel => {
      const el = node.querySelector(sel)

      return el ? (el.innerText || '').replace(/\s+/g, ' ').trim() : ''
    }

    const author =
      pick('.author .name') || pick('.name') || pick('.author-wrapper') || ''

    const content = pick('.content .note-text') || pick('.content') || ''

    const date = pick('.info .date') || ''
    const location = pick('.info .location') || ''
    const isAuthor = !!node.querySelector('.tag')

    const profileLink = node.querySelector('a[href*="/user/profile/"]')
    const profile = profileLink ? profileLink.getAttribute('href') : ''

    if (!takeQuota('scrape', 'xhs')) { console.log('今日小红书抓取配额已满，停止'); break }

    out.push({
      author: author.replace(/\s*作者\s*$/, '').trim(),
      content,
      date,
      location,
      isAuthor,
      profile
    })
  }

  return out
})

const stats = { raw: raw.length, kept: 0, skippedOld: 0, skippedExclude: 0, skippedAuthor: 0, skippedIrrelevant: 0 }

// 业务相关词：评论里得有买卖迹象，否则不进线索池
const BUSINESS_WORDS = ['蛋', '鸡', '批发', '价格', '价', '多少钱', '多少', '报价', '供货', '货源', '拿货', '进货', '采购', '求购', '要货', '一件', '一箱', '斤', '吨', '运费', '包邮']
const items = []

for (const item of raw) {
  if (!item.content || !item.author) continue
  if (item.isAuthor) {
    stats.skippedAuthor++
    continue
  }

  if (exclude.some(word => item.content.includes(word))) {
    stats.skippedExclude++
    continue
  }

  if (!BUSINESS_WORDS.some(word => item.content.includes(word))) {
    stats.skippedIrrelevant++
    continue
  }

  const days = daysAgo(item.date)
  if (maxDays > 0 && days >= 0 && days > maxDays) {
    stats.skippedOld++
    continue
  }

  items.push({
    noteId,
    author: item.author,
    profile: item.profile,
    content: item.content.slice(0, 300),
    location: item.location.replace(/^[\d-]+\s*/, '').trim(),
    date: item.date,
    days
  })

  if (items.length >= limit) break
}

stats.kept = items.length

const file = `E:\\ai\\data\\xhs_comments_${noteId}.json`
fs.writeFileSync(file, JSON.stringify({ noteId, keyword, noteAuthor, stats, items, fetchedAt: new Date().toLocaleString('sv-SE') }, null, 2), 'utf-8')

console.log(JSON.stringify({ ok: true, noteId, stats, file }))
await browser.close()
