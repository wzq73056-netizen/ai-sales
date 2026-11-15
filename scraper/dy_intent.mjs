/**
 * 评论意向判定 + 落库
 *
 * 优先用后端大模型判定（/lead/classifyComments），失败时退回本地正则规则。
 *
 * 用法：node dy_intent.mjs <commentsJson>
 */
import fs from 'node:fs'
import { mergeLeads } from './dy_lib.mjs'

const API = process.env.API || 'http://127.0.0.1:8888'
const KEYWORD = process.env.DY_KEYWORD || ''
const file = process.argv[2]
if (!file) {
  console.log('用法: node dy_intent.mjs <commentsJson>')
  process.exit(1)
}

const data = JSON.parse(fs.readFileSync(file, 'utf-8'))
const comments = data.comments || []
const videoTitle = data.videoTitle || ''

/** 本地正则兜底 */
const RULES = [
  { w: 40, tag: '问价', all: [/多少钱|价格|收费|怎么收|贵不贵|费用|价位|便宜吗|贵吗/] },
  { w: 35, tag: '求推荐', all: [/哪里|哪儿|有没有/, /靠谱|好的|推荐|便宜|介绍/] },
  { w: 35, tag: '求助', all: [/怎么办|怎么弄|如何|要不要|需不需要|该不该|求助/] },
  { w: 30, tag: '找服务', all: [/找谁|联系|咨询|推荐/] },
  { w: 30, tag: '自述需求', all: [/我(也|想|要|这边|公司|自己)/, /(做|找|办|注册|记账|代账)/] },
  { w: -35, tag: '疑似软广', all: [/效率超高|省心又省力|一站式服务|全额退款|专业团队|欢迎咨询|私信我|加我|找我办|承接|一条龙/] }
]

function regexScore(text) {
  let score = 0
  const tags = []

  for (const r of RULES) {
    if (r.all.every(re => re.test(text))) {
      score += r.w
      tags.push(r.tag)
    }
  }

  if (text.length > 120) score -= 10

  return { score, tags }
}

/** 调后端批量判定 */
async function classifyByLLM(items) {
  const loginRes = await fetch(`${API}/base/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: '123456', captcha: '', captchaId: '' })
  })
  const loginJson = await loginRes.json()
  if (loginJson.code !== 0) throw new Error(loginJson.msg)

  const token = loginJson.data.token
  const out = []

  for (let i = 0; i < items.length; i += 15) {
    const chunk = items.slice(i, i + 15)
    const res = await fetch(`${API}/lead/classifyComments`, {
      method: 'POST',
      headers: { 'x-token': token, 'Content-Type': 'application/json' },
      body: JSON.stringify({ Items: chunk })
    })
    const json = await res.json()
    if (json.code !== 0) throw new Error(json.msg)
    out.push(...json.data)
  }

  return out
}

let leads = []
let mode = 'regex'

try {
  const items = comments.map((c, i) => ({ id: String(i), content: c.content || '' }))
  const results = await classifyByLLM(items)
  mode = 'llm'

  const byId = new Map(results.map(r => [r.id, r]))

  for (let i = 0; i < comments.length; i++) {
    const c = comments[i]
    const r = byId.get(String(i))
    if (!r || r.label !== 'customer' || r.score <= 0) continue

    leads.push({
      author: c.author,
      profile: c.profile,
      time: c.time,
      region: c.region ?? '',
      days: c.days ?? null,
      content: c.content,
      score: Math.min(r.score, 100),
      tags: ['AI判定', '潜在客户'],
      avatar: c.avatar || '',
      videoTitle,
      keyword: KEYWORD,
      reason: r.reason || ''
    })
  }
} catch (e) {
  console.log('大模型判定不可用，退回正则：' + String(e).slice(0, 80))
  mode = 'regex'

  for (const c of comments) {
    const { score, tags } = regexScore(c.content || '')
    if (score <= 0) continue

    leads.push({
      author: c.author,
      profile: c.profile,
      time: c.time,
      region: c.region ?? '',
      days: c.days ?? null,
      content: c.content,
      score: Math.min(score, 100),
      tags,
      avatar: c.avatar || '',
      videoTitle,
      keyword: KEYWORD
    })
  }
}

leads.sort((a, b) => b.score - a.score)

const out = { videoId: data.videoId, mode, total: comments.length, candidates: leads.length, leads }
const outFile = file.replace(/\.json$/, '.leads.json')
fs.writeFileSync(outFile, JSON.stringify(out, null, 2), 'utf-8')

const merged = mergeLeads(leads, data.videoId)

console.log(
  JSON.stringify(
    { ok: true, mode, total: comments.length, candidates: leads.length, file: outFile, store: merged },
    null,
    2
  )
)
console.log(JSON.stringify(leads.slice(0, 4), null, 2))
