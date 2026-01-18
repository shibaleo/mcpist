/**
 * JWT utilities for MCPist Auth Server
 *
 * Uses RS256 (RSA-SHA256) for signing JWTs.
 * - Production: Keys from environment variables (AUTH_PRIVATE_KEY, AUTH_PUBLIC_KEY)
 * - Development: Auto-generates keys on first use and saves to files
 */

import { SignJWT, importPKCS8, importSPKI, exportJWK, jwtVerify, generateKeyPair, exportPKCS8, exportSPKI } from 'jose'
import { writeFileSync, readFileSync, existsSync } from 'fs'
import { join } from 'path'

// Key ID for JWKS
const KEY_ID = 'mcpist-auth-key-1'

// Development key file paths (relative to console app root)
const DEV_KEYS_DIR = process.cwd()
const DEV_PRIVATE_KEY_PATH = join(DEV_KEYS_DIR, '.auth-private-key.pem')
const DEV_PUBLIC_KEY_PATH = join(DEV_KEYS_DIR, '.auth-public-key.pem')

// Cached keys
let cachedPrivateKey: CryptoKey | null = null
let cachedPublicKey: CryptoKey | null = null
let keysLoaded = false

/**
 * Generate RSA key pair for development
 */
async function generateDevKeys(): Promise<{ privateKeyPEM: string; publicKeyPEM: string }> {
  console.log('[Auth] Generating RSA key pair for development...')
  const { privateKey, publicKey } = await generateKeyPair('RS256', { extractable: true })

  const privateKeyPEM = await exportPKCS8(privateKey)
  const publicKeyPEM = await exportSPKI(publicKey)

  // Save to files for persistence across restarts
  try {
    writeFileSync(DEV_PRIVATE_KEY_PATH, privateKeyPEM, { mode: 0o600 })
    writeFileSync(DEV_PUBLIC_KEY_PATH, publicKeyPEM)
    console.log(`[Auth] Keys saved to ${DEV_PRIVATE_KEY_PATH}`)
  } catch (e) {
    console.warn('[Auth] Could not save keys to files:', e)
  }

  return { privateKeyPEM, publicKeyPEM }
}

/**
 * Load keys from environment or generate for development
 */
async function loadKeys(): Promise<{ privateKeyPEM: string; publicKeyPEM: string }> {
  // 1. Try environment variables (production)
  const envPrivateKey = process.env.AUTH_PRIVATE_KEY
  const envPublicKey = process.env.AUTH_PUBLIC_KEY

  if (envPrivateKey && envPublicKey) {
    console.log('[Auth] Using keys from environment variables')
    return {
      privateKeyPEM: envPrivateKey.replace(/\\n/g, '\n'),
      publicKeyPEM: envPublicKey.replace(/\\n/g, '\n'),
    }
  }

  // 2. Development: Try to load from files
  try {
    if (existsSync(DEV_PRIVATE_KEY_PATH) && existsSync(DEV_PUBLIC_KEY_PATH)) {
      console.log('[Auth] Loading keys from files')
      return {
        privateKeyPEM: readFileSync(DEV_PRIVATE_KEY_PATH, 'utf-8'),
        publicKeyPEM: readFileSync(DEV_PUBLIC_KEY_PATH, 'utf-8'),
      }
    }
  } catch (e) {
    console.warn('[Auth] Could not load keys from files:', e)
  }

  // 3. Development: Generate new keys
  if (process.env.NODE_ENV !== 'production') {
    return generateDevKeys()
  }

  throw new Error('AUTH_PRIVATE_KEY and AUTH_PUBLIC_KEY environment variables are required in production')
}

/**
 * Ensure keys are loaded (call once at startup)
 */
async function ensureKeysLoaded(): Promise<void> {
  if (keysLoaded) return

  const { privateKeyPEM, publicKeyPEM } = await loadKeys()
  cachedPrivateKey = await importPKCS8(privateKeyPEM, 'RS256')
  cachedPublicKey = await importSPKI(publicKeyPEM, 'RS256')
  keysLoaded = true
}

async function getPrivateKey(): Promise<CryptoKey> {
  await ensureKeysLoaded()
  return cachedPrivateKey!
}

async function getPublicKey(): Promise<CryptoKey> {
  await ensureKeysLoaded()
  return cachedPublicKey!
}

export interface JWTPayload {
  sub: string        // user_id
  aud: string        // audience (e.g., "https://mcp.mcpist.app")
  scope?: string     // granted scopes
  email?: string     // user email
}

/**
 * Sign a JWT for MCP access
 */
export async function signJWT(payload: JWTPayload): Promise<string> {
  const privateKey = await getPrivateKey()
  const issuer = process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'

  const jwt = await new SignJWT({
    ...payload,
  })
    .setProtectedHeader({ alg: 'RS256', kid: KEY_ID })
    .setIssuer(issuer)
    .setAudience(payload.aud)
    .setSubject(payload.sub)
    .setIssuedAt()
    .setExpirationTime('1h')
    .sign(privateKey)

  return jwt
}

/**
 * Verify a JWT
 */
export async function verifyJWT(token: string): Promise<JWTPayload | null> {
  try {
    const publicKey = await getPublicKey()
    const issuer = process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'

    const { payload } = await jwtVerify(token, publicKey, {
      issuer,
    })

    return {
      sub: payload.sub as string,
      aud: payload.aud as string,
      scope: payload.scope as string | undefined,
      email: payload.email as string | undefined,
    }
  } catch {
    return null
  }
}

/**
 * Get JWKS (JSON Web Key Set) for public key
 */
export async function getJWKS() {
  const publicKey = await getPublicKey()
  const jwk = await exportJWK(publicKey)

  return {
    keys: [
      {
        ...jwk,
        kid: KEY_ID,
        use: 'sig',
        alg: 'RS256',
      },
    ],
  }
}
