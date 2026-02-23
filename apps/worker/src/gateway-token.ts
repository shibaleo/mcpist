/**
 * Gateway Token — Worker → Go Server 間の JWT 認証
 *
 * Worker が Ed25519 秘密鍵で短寿命 JWT を署名し、
 * Go Server が /.well-known/jwks.json の公開鍵で検証する。
 */

import * as jose from "jose";

const KID = "mcpist-gateway-v1";

// Ed25519 PKCS8 DER prefix (16 bytes)
// SEQUENCE(46) { INTEGER(0), SEQUENCE { OID(1.3.101.112) }, OCTET_STRING(34) { OCTET_STRING(32) { seed } } }
const ED25519_PKCS8_PREFIX = new Uint8Array([
  0x30, 0x2e, 0x02, 0x01, 0x00, 0x30, 0x05, 0x06,
  0x03, 0x2b, 0x65, 0x70, 0x04, 0x22, 0x04, 0x20,
]);

let cached: { privateKey: jose.KeyLike; publicJwk: jose.JWK } | null = null;

async function getKeyPair(signingKeyBase64: string): Promise<{ privateKey: jose.KeyLike; publicJwk: jose.JWK }> {
  if (cached) return cached;

  const seed = Uint8Array.from(atob(signingKeyBase64), (c) => c.charCodeAt(0));
  if (seed.length !== 32) {
    throw new Error(`Invalid GATEWAY_SIGNING_KEY seed length: ${seed.length} (expected 32)`);
  }

  // Build PKCS8 DER: 16-byte prefix + 32-byte seed
  const pkcs8 = new Uint8Array(48);
  pkcs8.set(ED25519_PKCS8_PREFIX, 0);
  pkcs8.set(seed, 16);

  const pem = `-----BEGIN PRIVATE KEY-----\n${btoa(String.fromCharCode(...pkcs8))}\n-----END PRIVATE KEY-----`;
  const privateKey = await jose.importPKCS8(pem, "EdDSA", { extractable: true });

  // Export full JWK (includes d + x), strip d to get public JWK
  const fullJwk = await jose.exportJWK(privateKey);
  const publicJwk: jose.JWK = {
    kty: fullJwk.kty,
    crv: fullJwk.crv,
    x: fullJwk.x,
    kid: KID,
    use: "sig",
    alg: "EdDSA",
  };

  cached = { privateKey, publicJwk };
  return cached;
}

/** Gateway JWT に含める claims */
export interface GatewayTokenClaims {
  /** Internal mcpist UUID (API key auth) */
  user_id?: string;
  /** Clerk user ID (Clerk JWT auth) */
  clerk_id?: string;
  /** User email (optional) */
  email?: string;
}

/**
 * Gateway JWT を署名して返す。
 * 有効期限 30 秒の短寿命トークン。
 */
export async function signGatewayToken(
  signingKeyBase64: string,
  claims: GatewayTokenClaims
): Promise<string> {
  const { privateKey } = await getKeyPair(signingKeyBase64);

  // Filter out undefined values
  const payload: Record<string, string> = {};
  if (claims.user_id) payload.user_id = claims.user_id;
  if (claims.clerk_id) payload.clerk_id = claims.clerk_id;
  if (claims.email) payload.email = claims.email;

  return new jose.SignJWT(payload)
    .setProtectedHeader({ alg: "EdDSA", kid: KID })
    .setIssuer("mcpist-gateway")
    .setIssuedAt()
    .setExpirationTime("30s")
    .sign(privateKey);
}

/**
 * JWKS JSON レスポンスを返す（公開鍵のみ）。
 * GET /.well-known/jwks.json で使用。
 */
export async function getJwksResponse(signingKeyBase64: string): Promise<Response> {
  const { publicJwk } = await getKeyPair(signingKeyBase64);
  return new Response(JSON.stringify({ keys: [publicJwk] }), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=3600",
    },
  });
}
