import {createRouter, createWebHistory} from 'vue-router'

const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes: [
        { path: '/', redirect: '/status' },
        {
            path: '/status',
            name: 'status',
            // Lazy-loaded: the code of a page is downloaded only when the page is opened.
            component: () => import('@/views/StatusView.vue'),
        },
    ],
})

export default router