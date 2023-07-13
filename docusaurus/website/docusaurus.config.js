// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require("prism-react-renderer/themes/github");
const darkCodeTheme = require("prism-react-renderer/themes/dracula");
// const remark = require("remark");
// const stripHTML = require("remark-strip-html");

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "grafana-apiserver",
  tagline: "A Kubernetes API Server for Grafana Resources",
  url: "https://grafana.github.io/",
  baseUrl: "grafana-apiserver/",
  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",
  favicon: "img/favicon.png",
  organizationName: "grafana",
  projectName: "grafana-apiserver",
  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },
  plugins: [],
  presets: [
    [
      "classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          routeBasePath: "/",
          path: "../docs/",
          sidebarPath: require.resolve("./sidebars.js"),
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            "https://github.com/grafana/grafana-apiserver/edit/main/docusaurus/website",
        },
        theme: {
          customCss: require.resolve("./src/css/custom.css"),
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: "grafana-apiserver",
        logo: {
          alt: "Grafana Logo",
          src: "img/logo.svg",
        },
        items: [
          {
            type: "doc",
            docId: "api-overview",
            position: "right",
            label: "Docs",
          },
          {
            href: "https://www.github.com/grafana/grafana-apiserver",
            label: "GitHub",
            position: "right",
          },
        ],
      },
      footer: {
        style: "dark",
        links: [
          {
            title: "Community",
            items: [
              {
                label: "GitHub",
                href: "https://www.github.com/grafana/grafana-apiserver",
              },
              {
                label: "Github Issues",
                href: "https://www.github.com/grafana/grafana-apiserver/issues",
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Grafana Labs. Built with Docusaurus.`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
      },
    }),
};

module.exports = config;
