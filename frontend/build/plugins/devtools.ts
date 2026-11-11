import type { Plugin } from 'vite';
import VueDevtools from 'vite-plugin-vue-devtools';

export function setupDevtoolsPlugin(viteEnv: Env.ImportMeta): Plugin {
  const { VITE_DEVTOOLS_LAUNCH_EDITOR, VITE_DEVTOOLS_ENABLE } = viteEnv;

  /** 需要 Vue DevTools 悬浮球时把 VITE_DEVTOOLS_ENABLE 设为 true，默认关闭以免挡住移动端底部导航 */
  if (VITE_DEVTOOLS_ENABLE !== 'true') {
    return { name: 'skip-vue-devtools' } as Plugin;
  }

  return VueDevtools({
    launchEditor: VITE_DEVTOOLS_LAUNCH_EDITOR
  });
}
