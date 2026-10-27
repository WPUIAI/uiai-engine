import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({ getVersion: vi.fn(), toast: vi.fn(), check: vi.fn(), relaunch: vi.fn() }));
vi.mock('@tauri-apps/api/app', () => ({ getVersion: mocks.getVersion }));
vi.mock('@tauri-apps/plugin-process', () => ({ relaunch: mocks.relaunch }));
vi.mock('@tauri-apps/plugin-updater', () => ({ check: mocks.check }));
vi.mock('$lib/ui/toast', () => ({ pushToast: mocks.toast }));
import { COCKPIT_UPDATE_RECEIPT_KEY, reportCompletedCockpitUpdate, runCockpitUpdate } from '$lib/updater';

const receipt = { schema: 'uiai.cockpit_ota_activation_receipt.v1', status: 'installed_relaunching', version: '0.3.1', activated_at: '2026-10-26T16:00:00Z', activation: 'signed_updater_install_and_relaunch' };
let storage: Map<string, string>;
beforeEach(() => {
 vi.resetAllMocks(); storage = new Map([[COCKPIT_UPDATE_RECEIPT_KEY, JSON.stringify(receipt)]]);
 vi.stubGlobal('window', { __TAURI_INTERNALS__: {}, dispatchEvent: vi.fn(), localStorage: { getItem: (key: string) => storage.get(key) ?? null, setItem: (key: string, value: string) => storage.set(key, value), removeItem: (key: string) => storage.delete(key) } });
});
afterEach(() => vi.unstubAllGlobals());
const successes = () => mocks.toast.mock.calls.filter(([toast]) => toast.level === 'success');

describe('OTA download completion is not verified installation or activation', () => {
 it('does not claim a manifest-only check has verified the artifact signature', async () => {
  const downloadAndInstall = vi.fn();
  mocks.check.mockResolvedValue({ version: '0.3.1', downloadAndInstall });
  const result = await runCockpitUpdate({ install: false });
  expect(result.phase).toBe('available');
  expect(result.message).not.toMatch(/is signed|signed and ready/i);
  expect(downloadAndInstall).not.toHaveBeenCalled();
  expect(successes()).toHaveLength(0);
 });
 it.each(['signature verification failed', 'installer failed'])('does not announce installation when %s follows Finished', async (failure) => {
  storage.clear();
  let successesAtFinished = -1;
  mocks.check.mockResolvedValue({ version: '0.3.1', downloadAndInstall: async (notify: (event: { event: string }) => void) => {
   notify({ event: 'Finished' });
   successesAtFinished = successes().length;
   throw new Error(failure);
  } });
  const result = await runCockpitUpdate({ install: true });
  expect(result.phase).toBe('error');
  expect(successesAtFinished).toBe(0);
  expect(successes()).toHaveLength(0);
  expect(storage.has(COCKPIT_UPDATE_RECEIPT_KEY)).toBe(false);
  expect(mocks.relaunch).not.toHaveBeenCalled();
 });
 it('does not announce activation if native relaunch fails after installation', async () => {
  storage.clear();
  mocks.check.mockResolvedValue({ version: '0.3.1', downloadAndInstall: async (notify: (event: { event: string }) => void) => { notify({ event: 'Finished' }); } });
  mocks.relaunch.mockRejectedValue(new Error('fixture relaunch failed'));
  expect((await runCockpitUpdate({ install: true })).phase).toBe('error');
  expect(successes()).toHaveLength(0);
  expect(JSON.parse(storage.get(COCKPIT_UPDATE_RECEIPT_KEY)!)).toMatchObject({ schema: receipt.schema, status: receipt.status, version: receipt.version, activation: receipt.activation });
 });
 it('reserves success for native running-version confirmation, not Finished or relaunch request', async () => {
  storage.clear();
  mocks.check.mockResolvedValue({ version: '0.3.1', downloadAndInstall: async (notify: (event: { event: string }) => void) => { notify({ event: 'Finished' }); } });
  mocks.relaunch.mockResolvedValue(undefined);
  expect((await runCockpitUpdate({ install: true })).phase).toBe('installing');
  expect(successes()).toHaveLength(0);
  mocks.getVersion.mockResolvedValue('0.3.1');
  expect(await reportCompletedCockpitUpdate()).toMatchObject({ version: '0.3.1' });
  expect(successes()).toHaveLength(1);
  expect(storage.has(COCKPIT_UPDATE_RECEIPT_KEY)).toBe(false);
 });
});

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
