import { createRouter, createWebHistory } from 'vue-router'
import TerminalView from "@/views/TerminalView.vue";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/terminal',
      // route level code-splitting
      // this generates a separate chunk (About.[hash].js) for this route
      // which is lazy-loaded when the route is visited.
      component: TerminalView,
    },
  ],
})

export default router
