import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import HealthCheckView from '../views/HealthCheckView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: DashboardView
    },
    {
      path: '/health',
      name: 'health',
      component: HealthCheckView
    }
  ]
})

export default router

