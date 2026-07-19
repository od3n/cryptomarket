/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  async rewrites() {
    const apiUrl = process.env.API_URL || "http://localhost:8080";
    const realtimeUrl = process.env.REALTIME_URL || "http://localhost:8081";

    return [
      {
        source: "/api/:path*",
        destination: `${apiUrl}/:path*`,
      },
      {
        source: "/events/:path*",
        destination: `${realtimeUrl}/events/:path*`,
      },
    ];
  },
};

export default nextConfig;
