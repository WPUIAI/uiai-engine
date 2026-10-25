import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';

const lock: { packages: Record<string, { version: string }> } = JSON.parse(readFileSync(new URL('../../package-lock.json', import.meta.url), 'utf8'));
function atLeast(actual: string, minimum: number[]) {
  const parts = actual.split('.').map(Number);
  for (let index = 0; index < minimum.length; index++) {
    if (parts[index] > minimum[index]) return true;
    if (parts[index] < minimum[index] || !Number.isFinite(parts[index])) return false;
  }
  return true;
}

describe('release dependency security baseline', () => {
  for (const [name, minimum] of [['vite', [8, 3, 5]], ['vitest', [5, 0, 5]], ['zod', [4, 6, 6]]] as const) {
    it(`${name} retains the patched resolved version`, () => {
      const entries = Object.entries(lock.packages).filter(([path]) => path.endsWith(`/node_modules/${name}`) || path === `node_modules/${name}`);
      expect(entries.length).toBeGreaterThan(0);
      for (const [, entry] of entries) expect(atLeast((entry as { version: string }).version, [...minimum])).toBe(true);
    });
  }
});
