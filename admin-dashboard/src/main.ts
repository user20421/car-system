import { createApp } from 'vue'
import { createPinia } from 'pinia'
import '@fontsource-variable/manrope'
import './assets/main.css'
import App from './App.vue'
import { router } from './router'
import { configureHttpAuth } from './stores/auth'

const app = createApp(App)
app.use(createPinia())
configureHttpAuth()
app.use(router)
void router.isReady().then(() => app.mount('#app'))
