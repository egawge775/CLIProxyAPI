// Copies the single-file Vite bundle to dist/management.html, the asset name the
// Go backend embeds and serves. Mirrors the upstream release workflow, which
// renames dist/index.html to management.html before publishing.
//
// Runs as part of `bun run build`; uses only Node built-ins so it works under
// Bun and Node alike.
import { copyFileSync, existsSync, statSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const source = join(root, 'dist', 'index.html');
const target = join(root, 'dist', 'management.html');

if (!existsSync(source)) {
  console.error(`emit-management-asset: missing build output ${source}`);
  process.exit(1);
}

copyFileSync(source, target);
console.log(
  `emit-management-asset: wrote ${target} (${statSync(target).size} bytes)`
);
