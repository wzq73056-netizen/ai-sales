<script setup lang="ts">
import { computed } from 'vue';

defineOptions({ name: 'MediaContent' });

interface Props {
  content?: string | null;
  /** 语音音频地址（微信语音转成 wav 后由后台托管） */
  mediaUrl?: string | null;
}

const props = withDefaults(defineProps<Props>(), { content: '', mediaUrl: '' });

/** 微信的图片/语音在库里是「[图片] xxx」「[语音] 转写内容」这样的文本，这里按类型渲染 */
const parsed = computed(() => {
  const raw = (props.content || '').trim();

  if (!raw) return { kind: 'text' as const, text: '' };

  if (raw.startsWith('[图片]') || raw.startsWith('[视频]')) {
    return { kind: 'image' as const, text: raw.replace(/^\[[^\]]+\]\s*/, '') };
  }

  if (raw.startsWith('[语音]')) {
    return { kind: 'voice' as const, text: raw.replace(/^\[语音\]\s*/, '') };
  }

  return { kind: 'text' as const, text: raw };
});

/** 音频完整地址（后台 8888 静态目录） */
const audioSrc = computed(() => {
  if (!props.mediaUrl) return '';

  return props.mediaUrl.startsWith('http') ? props.mediaUrl : `${import.meta.env.VITE_SERVICE_BASE_URL || ''}${props.mediaUrl}`;
});
</script>

<template>
  <div v-if="parsed.kind === 'text'" class="text-13px whitespace-pre-wrap">{{ parsed.text }}</div>

  <div v-else-if="parsed.kind === 'image'" class="inline-flex flex-col gap-4px">
    <div class="h-80px w-80px flex-center rounded-6px bg-gray-100 text-22px op-40 dark:bg-gray-800">
      <SvgIcon icon="mdi:image-outline" />
    </div>
    <span class="text-11px op-40">图片消息，原图在微信里</span>
  </div>

  <div v-else class="flex flex-col gap-6px">
    <div class="inline-flex w-fit items-center gap-6px rounded-999px bg-primary/8 px-10px py-4px text-12px text-primary">
      <SvgIcon icon="mdi:waveform" class="text-14px" />
      <span>语音</span>
      <span class="op-60">{{ audioSrc ? '可播放' : '已转写' }}</span>
    </div>
    <audio v-if="audioSrc" :src="audioSrc" controls class="h-32px w-220px" />
    <div class="text-13px whitespace-pre-wrap">{{ parsed.text || '（转写失败，需在微信里收听）' }}</div>
  </div>
</template>
