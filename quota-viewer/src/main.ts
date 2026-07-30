import { createApp } from 'vue'
import App from './App.vue'
import { markRuntime } from './lib/desktop'
import './styles/main.css'

markRuntime()
createApp(App).mount('#app')
