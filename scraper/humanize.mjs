/**
 * 反检测 + 拟人行为 + 配额
 *
 * 目的：把「脚本操作」的痕迹压到最低。参考业内做法（xiaohongshu-skills 的 stealth.py /
 * human.py / cdp.py）：指纹伪装 + 行为随机 + 真实输入事件。
 *
 * 用法：
 *   import { applyStealth, humanDelay, humanType, humanClick, scrollLikeHuman, quota } from './humanize.mjs'
 *
 *   const page = await ctx.newPage()
 *   await applyStealth(page)          // 必须在 page.goto 之前
 *   await humanDelay('read')          // 读页面前随机等
 */

/** 浏览器环境适配脚本（在页面脚本之前执行，保证环境一致性） */
const STEALTH_SCRIPT = `
(() => {
  // 1. 环境一致性：webdriver 标记在正常浏览器里是 undefined
  try {
    const proto = Object.getPrototypeOf(navigator);
    const desc = Object.getOwnPropertyDescriptor(proto, 'webdriver');
    if (desc) {
      Object.defineProperty(proto, 'webdriver', {
        get: () => undefined,
        configurable: true,
      });
    } else {
      Object.defineProperty(navigator, 'webdriver', { get: () => undefined, configurable: true });
    }
  } catch (e) {}

  // 2. 补齐 headless 环境里缺失的 chrome 对象
  try {
    if (!window.chrome) window.chrome = {};
    if (!window.chrome.runtime) window.chrome.runtime = {};
    if (!window.chrome.app) {
      window.chrome.app = {
        isInstalled: false,
        InstallState: { DISABLED: 'disabled', INSTALLED: 'installed', NOT_INSTALLED: 'not_installed' },
        RunningState: { CANNOT_RUN: 'cannot_run', READY_TO_RUN: 'ready_to_run', RUNNING: 'running' },
      };
    }
    if (!window.chrome.csi) window.chrome.csi = () => ({});
    if (!window.chrome.loadTimes) window.chrome.loadTimes = () => ({});
  } catch (e) {}

  // 3. 语言与平台，和 UA 保持一致（中文 Windows）
  try {
    Object.defineProperty(navigator, 'languages', { get: () => ['zh-CN', 'zh', 'en'], configurable: true });
    Object.defineProperty(navigator, 'vendor', { get: () => 'Google Inc.', configurable: true });
  } catch (e) {}

  // 4. WebGL vendor / renderer 与常见硬件保持一致
  try {
    const patchGL = (proto) => {
      if (!proto) return;
      const orig = proto.getParameter;
      proto.getParameter = function (p) {
        if (p === 37445) return 'Intel Inc.';
        if (p === 37446) return 'Intel Iris OpenGL Engine';
        return orig.apply(this, arguments);
      };
    };
    if (window.WebGLRenderingContext) patchGL(WebGLRenderingContext.prototype);
    if (window.WebGL2RenderingContext) patchGL(WebGL2RenderingContext.prototype);
  } catch (e) {}

  // 5. 硬件参数：给一组真实机器的常见值（每次加载在区间内随机，别每次都是一样）
  try {
    const pick = (a, b) => a + Math.floor(Math.random() * (b - a + 1));
    Object.defineProperty(navigator, 'hardwareConcurrency', { get: () => pick(8, 16), configurable: true });
    Object.defineProperty(navigator, 'deviceMemory', { get: () => 8, configurable: true });
  } catch (e) {}

  // 6. permissions.query：headless 里 notification 状态是 denied，真浏览器是 prompt
  try {
    const origQuery = window.navigator.permissions && window.navigator.permissions.query;
    if (origQuery) {
      window.navigator.permissions.query = (params) =>
        params && params.name === 'notifications'
          ? Promise.resolve({ state: Notification.permission })
          : origQuery.call(window.navigator.permissions, params);
    }
  } catch (e) {}
})();
`;

/** 注入指纹补丁：必须在 page.goto 之前调用 */
export async function applyStealth(page) {
  // CDP 层面注入，比 addInitScript 更早、更接近真实浏览器
  try {
    const ctx = page.context();
    const client = await ctx.newCDPSession(page);

    await client.send('Page.addScriptToEvaluateOnNewDocument', { source: STEALTH_SCRIPT });
    await client.send('Page.enable').catch(() => {});
  } catch (e) {
    // CDP 不可用时退回 Playwright 的注入方式
    await page.addInitScript({ content: STEALTH_SCRIPT }).catch(() => {});
  }

  // 鼠标事件走 CDP 真实事件通道（isTrusted=true）
  try {
    page.__cdp = await page.context().newCDPSession(page);
  } catch (e) {}

  return page;
}

const rand = (min, max) => min + Math.random() * (max - min);

/** 拟人等待：不同场景用不同区间，带 ± 抖动 */
export async function humanDelay(kind = 'op') {
  const ranges = {
    read: [1200, 2600], // 打开页面后"看"一会
    op: [600, 1400], // 两次操作之间
    click: [200, 450], // 点击前
    type: [0, 0], // 打字自己带延迟
    scroll: [300, 900]
  };

  const [min, max] = ranges[kind] || ranges.op;

  if (max > 0) await new Promise(r => setTimeout(r, rand(min, max)));
}

/** 拟人打字：逐字 30–80ms，偶尔停顿，换行转真实 Enter */
export async function humanType(page, text) {
  for (const ch of text) {
    if (ch === '\n') {
      await page.keyboard.press('Enter');
    } else {
      await page.keyboard.type(ch);
    }

    await new Promise(r => setTimeout(r, rand(30, 80)));

    // 每 8–15 个字偶尔停一下，像人在想
    if (Math.random() < 0.08) await new Promise(r => setTimeout(r, rand(120, 400)));
  }
}

/** 拟人点击：坐标加 ±3px 抖动，走 CDP 输入事件 */
export async function humanClick(page, x, y) {
  await humanDelay('click');

  const jx = x + rand(-3, 3);
  const jy = y + rand(-3, 3);

  if (page.__cdp) {
    try {
      await page.__cdp.send('Input.dispatchMouseEvent', { type: 'mouseMoved', x: jx, y: jy });
      await new Promise(r => setTimeout(r, rand(60, 160)));
      await page.__cdp.send('Input.dispatchMouseEvent', { type: 'mousePressed', x: jx, y: jy, button: 'left', clickCount: 1 });
      await page.__cdp.send('Input.dispatchMouseEvent', { type: 'mouseReleased', x: jx, y: jy, button: 'left', clickCount: 1 });

      return;
    } catch (e) {}
  }

  await page.mouse.click(jx, jy);
}

/** 拟人滚动：距离和次数随机，偶尔回滚一点 */
export async function scrollLikeHuman(page, rounds = 3) {
  const total = rounds + Math.floor(rand(0, 3));

  for (let i = 0; i < total; i++) {
    const dy = Math.floor(rand(200, 700));

    await page.mouse.wheel(0, dy);
    await new Promise(r => setTimeout(r, rand(400, 1100)));

    if (Math.random() < 0.2) {
      await page.mouse.wheel(0, -Math.floor(rand(60, 180)));
      await new Promise(r => setTimeout(r, rand(200, 500)));
    }
  }
}

// ---------------- 配额：每天抓取 / 私信上限 + 时段窗口 ----------------

import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { dirname } from 'node:path';

const QUOTA_FILE = process.env.QUOTA_FILE || 'E:\\ai\\data\\quota.json';

const LIMITS = {
  scrape: Number(process.env.QUOTA_SCRAPE || 200), // 每个平台每天抓取条数上限
  dm: Number(process.env.QUOTA_DM || 30), // 每个平台每天私信条数上限
  hours: (process.env.QUOTA_HOURS || '9-23').split('-').map(Number) // 只在此时段跑
};

// 抖音和小红书分开计数：各自 200 抓取 / 30 私信，互不占用
const PLATFORMS = ['douyin', 'xhs'];

function today() {
  const d = new Date();
  const p = n => String(n).padStart(2, '0');

  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

function fresh() {
  const state = { date: today() };

  for (const p of PLATFORMS) state[p] = { scrape: 0, dm: 0 };

  return state;
}

function load() {
  try {
    const raw = JSON.parse(readFileSync(QUOTA_FILE, 'utf8'));

    if (raw.date !== today()) return fresh();

    // 兼容旧格式（没有分平台字段）
    for (const p of PLATFORMS) {
      if (!raw[p]) raw[p] = { scrape: 0, dm: 0 };
    }

    return raw;
  } catch (e) {
    return fresh();
  }
}

function save(state) {
  try {
    mkdirSync(dirname(QUOTA_FILE), { recursive: true });
    writeFileSync(QUOTA_FILE, JSON.stringify(state, null, 2));
  } catch (e) {}
}

/** 当前时段是否允许跑 */
export function inActiveHours() {
  const h = new Date().getHours();
  const [start, end] = LIMITS.hours;

  return h >= start && h < end;
}

/**
 * 申请一次配额。返回 true 表示可以执行。
 *
 * @param {'scrape'|'dm'} kind   抓取 还是 私信
 * @param {string} platform      'douyin' | 'xhs'（默认 douyin）
 * @param {number} n             数量，默认 1
 */
export function takeQuota(kind, platform = 'douyin', n = 1) {
  // 兼容旧调用 takeQuota('dm', 5) 把第二参当数量的写法
  if (typeof platform === 'number') {
    n = platform;
    platform = 'douyin';
  }

  if (!PLATFORMS.includes(platform)) platform = 'douyin';

  if (!inActiveHours()) {
    console.error(`[quota] 当前不在允许时段 ${LIMITS.hours[0]}:00-${LIMITS.hours[1]}:00，跳过`);

    return false;
  }

  const state = load();

  if (state[platform][kind] + n > LIMITS[kind]) {
    console.error(`[quota] ${platform} 的 ${kind} 已达当日上限 ${LIMITS[kind]}，今天不再执行`);

    return false;
  }

  state[platform][kind] += n;
  save(state);

  return true;
}

/** 查看剩余额度（给日志用） */
export function quotaLeft(kind, platform = 'douyin') {
  const state = load();

  return Math.max(0, LIMITS[kind] - (state[platform]?.[kind] ?? 0));
}

export { LIMITS as quotaLimits };