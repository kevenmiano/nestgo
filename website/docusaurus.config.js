// @ts-check
const config = {
  title: 'NestGo',
  tagline: 'Structured applications. Native Go.',
  favicon: 'img/nestgo-mascot.svg',
  url: 'https://kevenmiano.github.io',
  baseUrl: '/nestgo/',
  trailingSlash: false,
  customFields: { releaseVersion: process.env.NESTGO_RELEASE_TAG || null },
  organizationName: 'kevenmiano',
  projectName: 'nestgo',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  i18n: {defaultLocale: 'en', locales: ['en']},
  markdown: {mermaid: true},
  themes: ['@docusaurus/theme-mermaid'],
  presets: [['classic', {
    docs: {sidebarPath: require.resolve('./sidebars.js'), routeBasePath: 'docs', editUrl: 'https://github.com/kevenmiano/nestgo/edit/main/website/'},
    blog: false,
    theme: {customCss: require.resolve('./src/css/custom.css')}
  }]],
  themeConfig: {
    image: 'img/nestgo-mascot.svg',
    navbar: {
      title: 'NestGo',
      logo: {alt: 'NestGo mascot', src: 'img/nestgo-mascot.svg'},
      items: [
        {to: '/docs/intro', label: 'Documentation', position: 'left'},
        {to: '/docs/roadmap', label: 'Roadmap', position: 'left'},
        {href: 'https://github.com/kevenmiano/nestgo', label: 'GitHub', position: 'right'}
      ]
    },
    footer: {style: 'dark', links: [{title: 'Explore', items: [{label: 'Get started', to: '/docs/intro'}, {label: 'Roadmap', to: '/docs/roadmap'}]}, {title: 'Community', items: [{label: 'GitHub', href: 'https://github.com/kevenmiano/nestgo'}]}], copyright: 'NestGo — experimental open-source project'},
    prism: {additionalLanguages: ['go', 'bash']}
  }
};
module.exports = config;
