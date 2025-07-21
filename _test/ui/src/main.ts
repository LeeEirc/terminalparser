import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
// 引入 xterm 样式
import '@xterm/xterm/css/xterm.css';
import App from './App.vue'
import router from './router'

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')
