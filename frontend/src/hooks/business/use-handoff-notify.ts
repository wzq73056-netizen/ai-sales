import { onBeforeUnmount, onMounted } from 'vue';
import { useNotification } from 'naive-ui';
import { request } from '@/service/request';

interface HandoffItem {
  Id: number;
  Channel: string;
  NickName: string;
  Category: string;
  LastMsg: string;
}

interface HandoffCount {
  Pending: number;
  Latest: HandoffItem[];
}

/**
 * 转人工工单全局提醒
 * 挂在布局里，任何页面有新工单都会弹通知 + 响铃 + 标题闪烁
 */
export function useHandoffNotify() {
  const notificationApi = useNotification();
  const originalTitle = document.title;

  let timer: ReturnType<typeof setInterval> | null = null;
  let flashTimer: ReturnType<typeof setInterval> | null = null;
  let lastPending = -1;

  // Naive 的 provider 在 AppProvider 里，hook 里拿到的可能是空实现，优先用全局实例
  function notifyApi() {
    return (window as any).$notification || notificationApi;
  }

  function beep() {
    try {
      const AudioCtx = window.AudioContext || (window as any).webkitAudioContext;
      const ctx = new AudioCtx();
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();

      osc.connect(gain);
      gain.connect(ctx.destination);
      osc.frequency.value = 880;
      gain.gain.value = 0.08;
      osc.start();
      osc.stop(ctx.currentTime + 0.25);
    } catch {
      // 浏览器限制自动播放，忽略
    }
  }

  function flashTitle(count: number) {
    if (flashTimer) return;

    let on = false;

    flashTimer = setInterval(() => {
      document.title = on ? `【${count} 条待接管】` : originalTitle;
      on = !on;
    }, 1200);
  }

  function stopFlash() {
    if (flashTimer) {
      clearInterval(flashTimer);
      flashTimer = null;
    }

    document.title = originalTitle;
  }

  async function check() {
    const { data } = await request<HandoffCount>({ url: '/lead/getWxHandoffCount', method: 'get' });

    if (!data) return;

    const pending = data.Pending ?? 0;

    if (pending > lastPending && lastPending >= 0) {
      const latest = (data.Latest || [])[0];

      if (latest) {
        const channel = latest.Channel === 'douyin' ? '抖音' : '微信';
        const content = `${latest.NickName}：${latest.Category}｜${latest.LastMsg || ''}`.slice(0, 90);

        notifyApi()?.warning({
          title: `新的转人工（${channel}）`,
          content,
          duration: 30000,
          keepAliveOnHover: true,
          meta: pending > 1 ? `当前共 ${pending} 条待接管` : undefined
        });

        beep();
        flashTitle(pending);
      }
    }

    if (pending === 0) stopFlash();

    lastPending = pending;
  }

  function onFocus() {
    stopFlash();
  }

  onMounted(() => {
    check();
    timer = setInterval(check, 30000);
    window.addEventListener('focus', onFocus);
  });

  onBeforeUnmount(() => {
    if (timer) clearInterval(timer);

    stopFlash();
    window.removeEventListener('focus', onFocus);
  });
}
