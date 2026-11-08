// path: frontend/next.config.js
/** @type {import('next').NextConfig} */
const isGitHubPages = process.env.GITHUB_PAGES === 'true';
const basePath = isGitHubPages ? '/artisan-market' : '';

const nextConfig = {
  reactStrictMode: true,
  output: isGitHubPages ? 'export' : 'standalone',
  ...(isGitHubPages && {
    basePath,
    assetPrefix: `${basePath}/`,
    trailingSlash: true,
  }),
};

module.exports = nextConfig;
