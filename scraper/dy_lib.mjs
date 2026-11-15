/**
 * 抖音采集公共工具：时间解析、地区提取、JSON 存取、线索去重合并
 */
import fs from 'node:fs'
import path from 'node:path'

export const DATA_DIR = 'E:\\ai\\data'
export const SEEN_VIDEOS = path.join(DATA_DIR, 'seen_videos.json')
export const LEADS_STORE = path.join(DATA_DIR, 'leads.json')

export function ensureDataDir() {
  fs.mkdirSync(DATA_DIR, { recursive: true })
}

export function readJson(file, fallback) {
  try {
    return JSON.parse(fs.readFileSync(file, 'utf-8'))
  } catch {
    return fallback
  }
}

export function writeJson(file, data) {
  ensureDataDir()
  fs.writeFileSync(file, JSON.stringify(data, null, 2), 'utf-8')
}

/** "6月前·福建" → { days: 180, region: '福建' } */
export function parseCommentTime(text) {
  if (!text) return { days: null, region: '' }

  const [timePart, regionPart] = String(text).split('·')
  const region = (regionPart || '').trim()

  let days = null
  const m = (timePart || '').match(/(\d+)\s*(秒|分钟|小时|天|周|月|年)前/)

  if (m) {
    const n = Number(m[1])
    const unit = m[2]
    const factor = { 秒: 0, 分钟: 0, 小时: 0, 天: 1, 周: 7, 月: 30, 年: 365 }[unit]
    days = factor === 0 ? 0 : n * factor
  } else if (/昨天/.test(timePart)) {
    days = 1
  } else if (/今天/.test(timePart)) {
    days = 0
  } else {
    const md = timePart.match(/(\d{1,2})-(\d{1,2})/)
    if (md) {
      const now = new Date()
      const d = new Date(now.getFullYear(), Number(md[1]) - 1, Number(md[2]))
      if (d > now) d.setFullYear(now.getFullYear() - 1)
      days = Math.floor((now - d) / 86400000)
    }
  }

  return { days, region }
}

/** 风控/验证码/登录失效的页面特征 */
const BLOCK_PATTERNS = [
  { re: /请完成安全验证|拖动滑块|滑动验证|验证码|人机验证/, reason: 'captcha' },
  { re: /操作过于频繁|访问频繁|请稍后再试|请求过于频繁/, reason: 'rate_limited' },
  { re: /网络环境异常|访问异常|当前环境异常/, reason: 'env_blocked' },
  { re: /登录后即可|扫码登录/, reason: 'not_logged_in' },
  { re: /页面不存在|内容不可见|审核中/, reason: 'unavailable' }
]

/**
 * 检测页面是否被风控/需要验证。命中返回 { blocked:true, reason, text }，否则 { blocked:false }
 * 用法：const guard = await detectBlock(page); if (guard.blocked) { console.log(JSON.stringify(guard)); process.exit(3) }
 */
export async function detectBlock(page) {
  const info = await page.evaluate(() => {
    const body = (document.body?.innerText || '').slice(0, 4000)
    return { text: body, url: location.href, title: document.title }
  })

  if (/verify|captcha/i.test(info.url)) {
    return { blocked: true, reason: 'captcha', url: info.url, text: info.text.slice(0, 120) }
  }

  for (const p of BLOCK_PATTERNS) {
    if (p.re.test(info.text) || p.re.test(info.title)) {
      return { blocked: true, reason: p.reason, url: info.url, text: info.text.slice(0, 120) }
    }
  }

  return { blocked: false }
}

/** 视频去重：返回未见过的新 id，并记录 */
export function filterNewVideos(items) {
  ensureDataDir()
  const seen = new Set(readJson(SEEN_VIDEOS, []))
  const fresh = items.filter(item => !seen.has(item.id))

  items.forEach(item => seen.add(item.id))
  writeJson(SEEN_VIDEOS, [...seen])

  return fresh
}

/** 合并线索：按 profile（或 作者+内容）去重，保留较高分 */
export function mergeLeads(newLeads, source) {
  ensureDataDir()
  const store = readJson(LEADS_STORE, { updatedAt: '', total: 0, leads: [] })
  const map = new Map(store.leads.map(item => [item.key, item]))

  let added = 0

  for (const lead of newLeads) {
    const key = lead.profile || `${lead.author}|${lead.content}`
    const existing = map.get(key)

    if (existing) {
      if (lead.score > existing.score) {
        map.set(key, { ...existing, ...lead, key, score: lead.score, updatedAt: new Date().toISOString() })
      }
      continue
    }

    map.set(key, { ...lead, key, source, discoveredAt: new Date().toISOString() })
    added++
  }

  const leads = [...map.values()].sort((a, b) => b.score - a.score)
  writeJson(LEADS_STORE, { updatedAt: new Date().toISOString(), total: leads.length, leads })

  return { added, total: leads.length, file: LEADS_STORE }
}
