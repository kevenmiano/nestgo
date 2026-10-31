// @ts-check
const config = {
  title: 'NestGo',
  tagline: 'Structured applications. Native Go.',
  favicon: 'img/favicon.svg',
  url: 'https://kevenmiano.github.io',
  baseUrl: '/nestgo/',
  organizationName: 'kevenmiano',
  projectName: 'nestgo',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  i18n: {defaultLocale: 'en', locales: ['en']},
  presets: [['classic', {
    docs: {sidebarPath: require.resolve('./sidebars.js'), routeBasePath: 'docs', editUrl: 'https://github.com/kevenmiano/nestgo/edit/main/website/'},
    blog: false,
    theme: {customCss: require.resolve('./src/css/custom.css')}
  }]],
  themeConfig: {
    image: 'img/favicon.svg',
    navbar: {
      title: 'NestGo',
      
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
