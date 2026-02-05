/**
 * Environment utilities
 */

export function getMcpServerUrl(): string {
  const url = process.env.NEXT_PUBLIC_MCP_SERVER_URL
  if (!url) {
    throw new Error("NEXT_PUBLIC_MCP_SERVER_URL is not set")
  }
  return url
}
