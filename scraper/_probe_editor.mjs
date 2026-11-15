import { chromium } from 'playwright-core'
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333')
const ctx = browser.contexts()[0]
const page = ctx.pages().find(p => p.url().includes('douyin.com')) ?? (await ctx.newPage())
await page.waitForTimeout(1500)
const info = await page.evaluate(() => {
  const editors = [...document.querySelectorAll('div[contenteditable="true"], textarea')]
  if (!editors.length) return { found: false }
  const ed = editors[0]
  const r = ed.getBoundingClientRect()
  let box = ed.parentElement
  for (let i = 0; i < 4 && box?.parentElement; i++) box = box.parentElement
  const nodes = []
  if (box) {
    for (const el of box.querySelectorAll('*')) {
      const rr = el.getBoundingClientRect()
      if (rr.width === 0 || rr.height === 0) continue
      nodes.push({
        tag: el.tagName.toLowerCase(),
        cls: (el.className || '').toString().slice(0, 50),
        text: (el.innerText || '').trim().slice(0, 12),
        box: [Math.round(rr.x), Math.round(rr.y), Math.round(rr.width), Math.round(rr.height)]
      })
    }
  }
  return { found: true, editorBox: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)], nodes: nodes.slice(-25) }
})
console.log(JSON.stringify(info, null, 2))
await browser.close()
