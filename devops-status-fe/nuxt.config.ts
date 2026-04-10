export default defineNuxtConfig({
  compatibilityDate: '2025-05-01',
  devtools: { enabled: true },

  runtimeConfig: {
    public: {
      apiBase: '',
    },
  },

  css: ['~/assets/css/main.css'],

  app: {
    head: {
      title: 'DOS — DevOpsStatus',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'DevOps Services Status Dashboard' },
      ],
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' },
      ],
    },
  },
})
