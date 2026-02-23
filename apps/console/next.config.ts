import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  env: {
    // Expose OAuth Server URL to client-side
    NEXT_PUBLIC_OAUTH_SERVER_URL: process.env.OAUTH_SERVER_URL || 'http://oauth.localhost',
  },
};

export default nextConfig;
