import type { NextConfig } from "next";

// Debug: log env vars at config load time
console.log('[next.config] NEXT_PUBLIC_SUPABASE_URL:', process.env.NEXT_PUBLIC_SUPABASE_URL);

const nextConfig: NextConfig = {
  reactStrictMode: true,
  env: {
    // Expose OAuth Server URL to client-side
    NEXT_PUBLIC_OAUTH_SERVER_URL: process.env.OAUTH_SERVER_URL || 'http://oauth.localhost',
  },
};

export default nextConfig;
