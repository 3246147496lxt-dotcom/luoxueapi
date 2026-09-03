import AdminApp from './AdminApp.vue'
import router from './router/admin'
import { bootstrapApp } from '@/app/bootstrap'

void bootstrapApp(AdminApp, router, 'admin')
