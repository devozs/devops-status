export default defineNuxtConfig({
  compatibilityDate: '2025-05-01',
  devtools: { enabled: true },

  runtimeConfig: {
    /**
     * Server-only: in-cluster backend base URL for SSR fetches (see useApi).
     * Override with NUXT_API_BASE_INTERNAL in production so Nitro does not call the public ingress URL from inside the pod.
     */
    apiBaseInternal: '',
    public: {
      apiBase: '',
    },
  },

  css: ['~/assets/css/main.css', '~/assets/css/form-controls.css', '~/assets/css/admin-shared.css'],

  app: {
    head: {
      title: 'DOS — DevOpsStatus',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'DevOps Services Status Dashboard' },
      ],
      link: [
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap',
        },
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' },
      ],
    },
  },
})
