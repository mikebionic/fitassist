import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const dark = ref(localStorage.getItem('theme') === 'dark' || !localStorage.getItem('theme'))

  watch(dark, (val) => {
    localStorage.setItem('theme', val ? 'dark' : 'light')
    applyTheme(val)
  }, { immediate: true })

  function applyTheme(isDark: boolean) {
    document.documentElement.classList.toggle('p-dark', isDark)
    document.documentElement.style.colorScheme = isDark ? 'dark' : 'light'
  }

  function toggle() {
    dark.value = !dark.value
  }

  return { dark, toggle }
})
