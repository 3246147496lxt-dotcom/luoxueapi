import { createApp } from 'vue'
import { createI18n } from 'vue-i18n'
import App from './App.vue'
import { messages } from './i18n'
import './styles/main.css'

const savedLocale = localStorage.getItem('luoxue-desktop-locale')
const locale = savedLocale === 'en' ? 'en' : 'zh-CN'

const i18n = createI18n({
  legacy: false,
  locale,
  fallbackLocale: 'zh-CN',
  messages
})

createApp(App).use(i18n).mount('#app')
