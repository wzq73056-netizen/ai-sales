import { defineConfig } from 'vitest/config';

/**
 * 单元测试专用配置。
 *
 * 不加载应用的 vite.config（含大量插件/别名），只跑纯逻辑测试，
 * 避免测试结束后 Vite server 挂住不退出。
 */
export default defineConfig({
  test: {
    environment: 'node',
    include: ['src/**/__tests__/**/*.test.ts'],
    pool: 'forks',
    watch: false
  }
});
