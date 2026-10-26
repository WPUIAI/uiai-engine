import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({ getVersion: vi.fn(), toast: vi.fn() }));
vi.mock('@tauri-apps/api/app', () => ({ getVersion: mocks.getVersion }));
vi.mock('@tauri-apps/plugin-process', () => ({ relaunch: vi.fn() }));
vi.mock('@tauri-apps/plugin-updater', () => ({ check: vi.fn() }));
vi.mock('$lib/ui/toast', () => ({ pushToast: mocks.toast }));
import { COCKPIT_UPDATE_RECEIPT_KEY, reportCompletedCockpitUpdate } from '$lib/updater';

const receipt = { schema: 'uiai.cockpit_ota_activation_receipt.v1', status: 'installed_relaunching', version: '0.3.1', activated_at: '2026-10-26T16:00:00Z', activation: 'signed_updater_install_and_relaunch' };
let storage: Map<string, string>;
beforeEach(() => {
 vi.resetAllMocks(); storage = new Map([[COCKPIT_UPDATE_RECEIPT_KEY, JSON.stringify(receipt)]]);
 vi.stubGlobal('window', { __TAURI_INTERNALS__: {}, localStorage: { getItem: (key: string) => storage.get(key) ?? null, removeItem: (key: string) => storage.delete(key) } });
});
afterEach(() => vi.unstubAllGlobals());
const successes = () => mocks.toast.mock.calls.filter(([toast]) => toast.level === 'success');

describe('OTA activation requires native running-version evidence', () => {
 it('does not announce activation or lose the receipt when relaunch left the old process running', async () => {
  mocks.getVersion.mockResolvedValue('0.3.0');
  expect(await reportCompletedCockpitUpdate()).toBeUndefined();
  expect(mocks.getVersion).toHaveBeenCalledOnce();
  expect(successes()).toHaveLength(0);
  expect(storage.has(COCKPIT_UPDATE_RECEIPT_KEY)).toBe(true);
 });
 it('waits for the native version query before announcing success', async () => {
  let resolve!: (value: string) => void;
  mocks.getVersion.mockReturnValue(new Promise<string>((done) => { resolve = done; }));
  const pending = reportCompletedCockpitUpdate();
  expect(successes()).toHaveLength(0);
  expect(storage.has(COCKPIT_UPDATE_RECEIPT_KEY)).toBe(true);
  resolve('0.3.1');
  expect(await pending).toEqual(receipt);
  expect(successes()).toHaveLength(1);
  expect(storage.has(COCKPIT_UPDATE_RECEIPT_KEY)).toBe(false);
 });
 it('preserves pending activation when the native query is unavailable', async () => {
  mocks.getVersion.mockRejectedValue(new Error('fixture native query unavailable'));
  expect(await reportCompletedCockpitUpdate()).toBeUndefined();
  expect(successes()).toHaveLength(0);
  expect(storage.has(COCKPIT_UPDATE_RECEIPT_KEY)).toBe(true);
 });
 it('never treats a web-only window as a native relaunch', async () => {
  delete (window as unknown as Record<string, unknown>).__TAURI_INTERNALS__;
  expect(await reportCompletedCockpitUpdate()).toBeUndefined();
  expect(mocks.getVersion).not.toHaveBeenCalled();
  expect(successes()).toHaveLength(0);
  expect(storage.has(COCKPIT_UPDATE_RECEIPT_KEY)).toBe(true);
 });
 it("preserves a newer pending update while the native version query is in flight", async () => {
  let resolve!: (value: string) => void;
  mocks.getVersion.mockReturnValue(new Promise<string>((done) => { resolve = done; }));
  const pending = reportCompletedCockpitUpdate();
  const newer = JSON.stringify({ ...receipt, version: "0.3.2" });
  storage.set(COCKPIT_UPDATE_RECEIPT_KEY, newer);
  resolve("0.3.1");
  expect(await pending).toBeUndefined();
  expect(successes()).toHaveLength(0);
  expect(storage.get(COCKPIT_UPDATE_RECEIPT_KEY)).toBe(newer);
 });
 it.each([null, { ...receipt, schema: 'unknown' }, { ...receipt, status: 'unknown' }, { ...receipt, version: 123 }, { ...receipt, version: '' }, { ...receipt, version: 'x'.repeat(257) }])('does not verify malformed or unknown receipts', async (invalid) => {
  storage.set(COCKPIT_UPDATE_RECEIPT_KEY, JSON.stringify(invalid));
  expect(await reportCompletedCockpitUpdate()).toBeUndefined();
  expect(mocks.getVersion).not.toHaveBeenCalled();
  expect(successes()).toHaveLength(0);
 });
});
