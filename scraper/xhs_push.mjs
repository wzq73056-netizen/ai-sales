/**
 * 小红书评论 → 线索推送（对接后端 /lead/createLead，Source=xhs）
 *
 * 用法：node xhs_push.mjs [文件或目录]
 */
import fs from 'node:fs'
import path from 'node:path'
import { login } from './xhs_lib.mjs'

const API = process.env.API || 'http://127.0.0.1:8888'
const input = process.argv[2] || 'E:\\ai\\data'

/** 抽手机号 / 微信号 */
function extractContacts(text) {
  const src = String(text || '')
  const phone = src.match(/1[3-9]\d{9}/)?.[0] || ''
  const wechat = src.match(/(?:vx|wx|微信|v信|加我)[:：]?\s*([A-Za-z][-_A-Za-z0-9]{5,19})/i)?.[1] || ''

  return { phone, wechat }
}

function collectFiles(target) {
  const stat = fs.statSync(target)

  if (stat.isFile()) return [target]

  return fs
    .readdirSync(target)
    .filter(name => name.startsWith('xhs_comments_') && name.endsWith('.json'))
    .map(name => path.join(target, name))
}

// 业务相关词：内容里得沾边，否则不进线索池
const BIZ_WORDS = ['蛋', '鸡', '批发', '价', '货源', '供货', '拿货', '进货', '采购', '求购', '运费', '斤', '吨', '一件', '一箱']

function isRelevant(text) {
  const src = String(text || '')

  return BIZ_WORDS.some(word => src.includes(word))
}

// 供货类关键词：用这些词采到的号多半是供货方（养殖场、批发商），方向记 supplier
const SUPPLIER_KEYWORDS = ['货源', '养殖场', '厂家', '供应', '一手货', '养殖']

/** 按采集关键词判断这条线索的方向 */
function sideOfKeyword(keyword) {
  const src = String(keyword || '')

  return SUPPLIER_KEYWORDS.some(word => src.includes(word)) ? 'supplier' : 'buyer'
}

// 买方意图词：只有这些才算「想买」
const BUYER_WORDS = ['求购', '我需要', '我想买', '想进货', '哪里拿货', '怎么批发', '批发价多少', '找货源', '长期采购', '要货', '一件代发吗']

// 卖方话术：养殖场/厂家推销，不是客户
const SELLER_WORDS = ['需要鸡蛋吗', '还要鸡蛋吗', '养殖场', '厂家', '直供', '出货', '供应', '批发鸡蛋', '出售']

/** 有没有明确的采购意图（先排除卖家推销） */
function looksLikeBuyer(text) {
  const src = String(text || '')

  if (SELLER_WORDS.some(word => src.includes(word))) return false

  return BUYER_WORDS.some(word => src.includes(word))
}

const token = await login(API)

// 已有线索（按 source_url 去重）
const res = await fetch(`${API}/lead/getLeadList?page=1&pageSize=500`, { headers: { 'x-token': token } })
const json = await res.json()
const have = new Set((json?.data?.list || []).map(item => item.SourceUrl).filter(Boolean))
// 小红书同一个人可能在多条笔记下留言，按昵称也去重一次
const haveNames = new Set(
  (json?.data?.list || []).filter(item => item.Source === 'xhs').map(item => (item.ContactName || '').trim()).filter(Boolean)
)

let pushed = 0
let skipped = 0
let failed = 0

for (const file of collectFiles(input)) {
  let data

  try {
    data = JSON.parse(fs.readFileSync(file, 'utf-8'))
  } catch {
    continue
  }

  // 只收评论区问价的人，不再把笔记作者当线索
  //
  // 原因：作者都是发笔记的同行/卖家，正文是「发布笔记：…」，没有任何买方意图，
  // 分级永远落到 B。真正的线索是评论区问「多少钱」「能发合肥吗」的人。
  // 需要作者时用环境变量 XHS_INCLUDE_AUTHOR=1 打开。
  const author = data.noteAuthor

  if (process.env.XHS_INCLUDE_AUTHOR === '1' && author && author.name && isRelevant(author.title)) {
    const authorUrl = author.profile ? `https://www.xiaohongshu.com${author.profile}` : ''

    if (authorUrl && !have.has(authorUrl)) {
      const body = {
        Company: author.name,
        ContactName: author.name,
        City: '',
        Industry: '鸡蛋批发',
        Source: 'xhs',
        SourceUrl: authorUrl,
        Keyword: data.keyword || '小红书',
        Score: 55,
        Status: 0,
        Remark: `发布笔记：${String(author.title || '').slice(0, 60)}`,
        ExtractInfo: author.name,
        Phone: '',
        Wechat: '',
        VideoTitle: String(author.title || '').slice(0, 200),
        Avatar: '',
        CreatedAt: new Date().toLocaleString('sv-SE'),
        UpdatedAt: new Date().toLocaleString('sv-SE')
      }

      const r = await fetch(`${API}/lead/createLead`, {
        method: 'POST',
        headers: { 'x-token': token, 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      })
      const result = await r.json()

      if (result.code === 0) {
        pushed++
        have.add(authorUrl)
        haveNames.add(author.name)
      } else failed++
    } else skipped++
  }
  for (const item of data.items || []) {
    if (!isRelevant(item.content)) {
      skipped++
      continue
    }

    // 小红书评论区大多是晒价格/消费者闲聊，没有采购意图的一律不入库
    if (!looksLikeBuyer(item.content)) {
      skipped++
      continue
    }

    const profile = item.profile || ''
    const url = profile ? `https://www.xiaohongshu.com${profile}` : ''

    if (url && have.has(url)) {
      skipped++
      continue
    }

    const nick = String(item.author || '').trim()

    if (nick && haveNames.has(nick)) {
      skipped++
      continue
    }

    const contacts = extractContacts(item.content)

    const body = {
      Company: item.author,
      ContactName: item.author,
      City: item.location || '',
      Industry: '鸡蛋批发',
      Source: 'xhs',
      SourceUrl: url,
      Side: sideOfKeyword(data.keyword),
      Keyword: data.keyword || '小红书',
      // 评论里出现「多少钱 / 能发 / 怎么卖」这类问价，本身就是买方意向
      Score: Math.min(80, 45 + (contacts.phone || contacts.wechat ? 30 : 0) + (/(多少|几块|啥价|什么价|怎么卖|能发|包邮|运费|起订|一件)/.test(String(item.content || '')) ? 10 : 0)),
      Status: 0,
      Remark: String(item.content || '').slice(0, 480),
      ExtractInfo: contacts.phone || contacts.wechat || item.author,
      Phone: contacts.phone,
      Wechat: contacts.wechat,
      VideoTitle: '',
      Avatar: '',
      CreatedAt: new Date().toLocaleString('sv-SE'),
      UpdatedAt: new Date().toLocaleString('sv-SE')
    }

    const r = await fetch(`${API}/lead/createLead`, {
      method: 'POST',
      headers: { 'x-token': token, 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    const result = await r.json()

    if (result.code === 0) {
      pushed++
      if (url) have.add(url)
      if (nick) haveNames.add(nick)
    } else {
      failed++
    }
  }
}

console.log(JSON.stringify({ ok: true, pushed, skipped, failed }))
