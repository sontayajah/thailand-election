import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // Allow party logo images served from the backend
  images: {
    remotePatterns: [
      {
        protocol: 'http',
        hostname: 'localhost',
        port: '8080',
        pathname: '/static/**',
      },
    ],
  },

  // Rewrite /api/v1/* → backend (server-side proxy — avoids CORS in browser).
  // In Docker: API_INTERNAL_URL=http://api:8080/api/v1 (direct container-to-container).
  // Outside Docker (native dev): falls back to localhost:8080.
  async rewrites() {
    const apiTarget =
      process.env.API_INTERNAL_URL ?? 'http://localhost:8080/api/v1';
    return [
      {
        source: '/api/v1/:path*',
        destination: `${apiTarget}/:path*`,
      },
    ];
  },
};

export default nextConfig;
