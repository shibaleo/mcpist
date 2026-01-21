#!/usr/bin/env node
/**
 * 環境変数同期スクリプト
 *
 * ルートの .env.local から各モジュールに必要な環境変数を配布する
 * - apps/worker/.dev.vars: Wranglerのシークレット
 */

const fs = require('fs');
const path = require('path');

const ROOT_DIR = path.resolve(__dirname, '..');
const ENV_LOCAL = path.join(ROOT_DIR, '.env.local');

// 各モジュールが必要とする環境変数のマッピング
const MODULE_ENV_MAP = {
  'apps/worker/.dev.vars': [
    'GATEWAY_SECRET',
    'SUPABASE_ANON_KEY',
  ],
};

function parseEnvFile(filePath) {
  if (!fs.existsSync(filePath)) {
    console.error(`Error: ${filePath} not found`);
    console.error('Please copy .env.example to .env.local and fill in the values');
    process.exit(1);
  }

  const content = fs.readFileSync(filePath, 'utf-8');
  const env = {};

  content.split('\n').forEach(line => {
    const trimmed = line.trim();
    if (trimmed && !trimmed.startsWith('#')) {
      const eqIndex = trimmed.indexOf('=');
      if (eqIndex !== -1) {
        const key = trimmed.substring(0, eqIndex);
        const value = trimmed.substring(eqIndex + 1);
        env[key] = value;
      }
    }
  });

  return env;
}

function generateEnvContent(env, keys, header = '') {
  const lines = [];
  if (header) {
    lines.push(header);
    lines.push('');
  }

  keys.forEach(key => {
    if (env[key]) {
      lines.push(`${key}=${env[key]}`);
    } else {
      console.warn(`Warning: ${key} not found in .env.local`);
    }
  });

  return lines.join('\n') + '\n';
}

function main() {
  console.log('Syncing environment variables from .env.local...\n');

  const env = parseEnvFile(ENV_LOCAL);

  for (const [targetPath, keys] of Object.entries(MODULE_ENV_MAP)) {
    const fullPath = path.join(ROOT_DIR, targetPath);
    const header = `# Auto-generated from root .env.local\n# Do not edit directly - run 'pnpm env:sync' to update`;
    const content = generateEnvContent(env, keys, header);

    fs.writeFileSync(fullPath, content);
    console.log(`✓ ${targetPath}`);
    keys.forEach(key => {
      const value = env[key];
      if (value) {
        const displayValue = value.length > 20 ? value.substring(0, 20) + '...' : value;
        console.log(`    ${key}=${displayValue}`);
      }
    });
  }

  console.log('\nEnvironment sync complete!');
}

main();
