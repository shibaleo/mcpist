import { headers } from "next/headers"

export type Lang = "ja" | "en"

/**
 * Legal ページの表示言語を決定する。
 * 1. searchParams の lang が明示されていればそれを使う
 * 2. なければ Accept-Language ヘッダーから判定
 * 3. フォールバックは英語
 */
export async function resolveLang(langParam?: string): Promise<Lang> {
  if (langParam === "ja" || langParam === "en") return langParam

  const h = await headers()
  const accept = h.get("accept-language") ?? ""
  // Accept-Language の先頭から ja を探す (例: "ja,en-US;q=0.9,en;q=0.8")
  const preferred = accept.split(",").map((s) => s.split(";")[0].trim().toLowerCase())
  for (const tag of preferred) {
    if (tag.startsWith("ja")) return "ja"
    if (tag.startsWith("en")) return "en"
  }

  return "en"
}
