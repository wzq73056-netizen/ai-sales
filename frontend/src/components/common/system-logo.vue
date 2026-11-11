<template>
  <div class="app-logo">
    <svg width="100%" height="100%" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="logoBg" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stop-color="var(--logo-color-500)" />
          <stop offset="1" stop-color="var(--logo-color-700)" />
        </linearGradient>
      </defs>

      <!-- 圆角底 -->
      <rect x="0" y="0" width="48" height="48" rx="12" fill="url(#logoBg)" />

      <!-- 对话气泡 -->
      <path
        d="M14 15.5h20a3.5 3.5 0 0 1 3.5 3.5v10a3.5 3.5 0 0 1-3.5 3.5H23l-6.5 5v-5H14a3.5 3.5 0 0 1-3.5-3.5V19a3.5 3.5 0 0 1 3.5-3.5Z"
        fill="#fff"
        fill-opacity="0.96"
      />

      <!-- 上升箭头（获客增长） -->
      <path
        d="M20 27.5l4.6-4.6 3.2 3.2 4.4-4.4"
        stroke="var(--logo-color-600)"
        stroke-width="2.2"
        stroke-linecap="round"
        stroke-linejoin="round"
        fill="none"
      />
      <path
        d="M29.6 21.7h3.2v3.2"
        stroke="var(--logo-color-600)"
        stroke-width="2.2"
        stroke-linecap="round"
        stroke-linejoin="round"
        fill="none"
      />
    </svg>
  </div>
</template>

<style scoped>
.app-logo {
  --logo-color-300: rgb(var(--primary-300-color));
  --logo-color-400: rgb(var(--primary-400-color));
  --logo-color-500: rgb(var(--primary-500-color));
  --logo-color-600: rgb(var(--primary-600-color));
  --logo-color-700: rgb(var(--primary-700-color));
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
