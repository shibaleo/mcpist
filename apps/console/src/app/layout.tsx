import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "MCPist Console",
  description: "MCPist User Console",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="ja">
      <body>{children}</body>
    </html>
  );
}
