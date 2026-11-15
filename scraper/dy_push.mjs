/**
 * 把抖音线索推送到现有后台（ai_lead 表）
 *
 * 用法：node dy_push.mjs [leads.json 路径]
 * 幂等：按 profile URL 去重（已在库里跳过）
 */
import fs from 'node:fs'

// 供货类关键词：采到的号多半是供货方（养殖场/批发商）
const SUPPLIER_KEYWORDS = ['货源', '养殖场', '厂家', '供应', '一手货', '养殖'];

function sideOfKeyword(keyword) {
  const src = String(keyword || '');

  return SUPPLIER_KEYWORDS.some(word => src.includes(word)) ? 'supplier' : 'buyer';
}

const API = process.env.API || 'http://127.0.0.1:8888'
const file = process.argv[2] || 'E:\\ai\\data\\leads.json'

const store = JSON.parse(fs.readFileSync(file, 'utf-8'))
const leads = store.leads || []

/** 从评论正文抽手机号/微信号 */
function extractContacts(lead) {
  const text = String(lead.content || '')

  const phone = text.match(/1[3-9]\d{9}/)?.[0] || ''
  const wx =
    text.match(/(?:vx|wx|微信|v信|加我)[:：]?\s*([A-Za-z][-_A-Za-z0-9]{5,19})/i)?.[1] || ''

  return { phone, wechat: wx }
}

/** 兜底信息：账号 ID 或昵称 */
function fallbackInfo(lead) {
  const uid = String(lead.profile || '').split('/user/')[1]?.split('?')[0] || ''
  return uid || lead.author || ''
}

async function login() {
  const res = await fetch(`${API}/base/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: '123456', captcha: '', captchaId: '' })
  })
  const json = await res.json()
  if (json.code !== 0) throw new Error(`登录失败: ${json.msg}`)
  return json.data.token
}

const token = await login()

// 已存在的（按 sourceUrl 判断，先取全量）
async function existingUrls() {
  const res = await fetch(`${API}/lead/getLeadList?page=1&pageSize=500`, {
    headers: { 'x-token': token }
  })
  const json = await res.json()
  const list = json?.data?.list ?? []
  return new Set(list.map(i => i.SourceUrl).filter(Boolean))
}

const have = await existingUrls()

let created = 0
let skipped = 0
let failed = 0

for (const lead of leads) {
  // 账号线索走 ai_douyin_account，不重复进评论线索池
  if ((lead.tags || []).includes('账号获客')) continue;
  const url = lead.profile ? `https:${lead.profile}` : ''

  if (url && have.has(url)) {
    skipped++
    continue
  }

  const remark = String(lead.content ?? "").slice(0, 480)
  const contacts = extractContacts(lead)
  const extractInfo = contacts.phone || contacts.wechat || fallbackInfo(lead)

  const body = {
    Company: lead.author || '抖音用户',
    ContactName: lead.author || '',
    City: lead.region || '',
    Industry: '财税代账',

    Source: 'douyin',
    SourceUrl: url,
    Side: sideOfKeyword(lead.keyword),
    Keyword: lead.keyword || '抖音评论',
    Score: Math.min(lead.score ?? 0, 100),
    Status: 0,
    Remark: remark,
    ExtractInfo: extractInfo,
    Phone: contacts.phone,
    Wechat: contacts.wechat,
    Avatar: (lead.avatar || '').startsWith('//') ? `https:${lead.avatar}` : lead.avatar || '',
    VideoTitle: lead.videoTitle || '',
    // 用本地时区的 ISO 形式（sv-SE 输出 YYYY-MM-DD HH:mm:ss），避免 UTC 偏差
    CreatedAt: new Date().toLocaleString('sv-SE'),
    UpdatedAt: new Date().toLocaleString('sv-SE')
  }

  const res = await fetch(`${API}/lead/createLead`, {
    method: 'POST',
    headers: { 'x-token': token, 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })
  const json = await res.json()

  if (json.code === 0) created++
  else if (/Duplicate|1062/.test(json.msg || '')) skipped++
  else {
    failed++
    console.log('失败:', lead.author, json.msg)
  }
}

console.log(JSON.stringify({ ok: true, total: leads.length, created, skipped, failed }, null, 2))
