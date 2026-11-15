import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1000)
const info = await page.evaluate(() => {
  // 找「上林弘康」文本节点，向上找行容器
  const all = [...document.querySelectorAll('*')]
  const node = all.find(el => el.children.length === 0 && (el.innerText || '').trim().startsWith('上林弘康'))
  if (!node) return { found: false }
  const chain = []
  let el = node
  for (let i = 0; i < 6 && el; i++) {
    const r = el.getBoundingClientRect()
    chain.push({
      tag: el.tagName.toLowerCase(),
      cls: (el.className || '').toString().slice(0, 45),
      box: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)],
      imgs: el.querySelectorAll('img').length,
      kids: el.children.length
    })
    el = el.parentElement
  }
  return { found: true, chain }
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
