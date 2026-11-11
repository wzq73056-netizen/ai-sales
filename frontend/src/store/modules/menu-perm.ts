import { ref } from 'vue';
import { defineStore } from 'pinia';

/**
 * 后端菜单权限：决定侧边栏显示哪些菜单
 *
 * - `loaded=false`：还没从后端拿到，默认全放行（避免接口抖动导致整个菜单消失）
 * - `loaded=true`：严格按白名单过滤
 */
export const useMenuPermStore = defineStore('menu-perm', () => {
  const allowedKeys = ref<string[]>([]);
  const loaded = ref(false);

  function setAllowed(keys: string[]) {
    allowedKeys.value = keys;
    loaded.value = true;
  }

  function isAllowed(key: string) {
    if (!loaded.value) return true;

    return allowedKeys.value.includes(key);
  }

  function reset() {
    allowedKeys.value = [];
    loaded.value = false;
  }

  return { allowedKeys, loaded, setAllowed, isAllowed, reset };
});
