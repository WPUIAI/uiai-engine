import { relaunch } from '@tauri-apps/plugin-process';
import { check } from '@tauri-apps/plugin-updater';
import { pushToast } from '$lib/ui/toast';

export type CockpitUpdatePhase =
  | 'checking'
  | 'current'
  | 'available'
  | 'downloading'
  | 'installing'
  | 'unavailable'
  | 'error';

export interface CockpitUpdateResult {
  phase: CockpitUpdatePhase;
  message: string;
  version?: string;
  downloadedBytes?: number;
  totalBytes?: number;
}

export type CockpitUpdateReporter = (result: CockpitUpdateResult) => void;

export const COCKPIT_UPDATE_RECEIPT_KEY = 'uiai.cockpit.ota.activation.v1';

interface CockpitUpdateActivationReceipt {
  schema: 'uiai.cockpit_ota_activation_receipt.v1';
  status: 'installed_relaunching';
  version: string;
  activated_at: string;
  activation: 'signed_updater_install_and_relaunch';
}

function isTauriRuntime(): boolean {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;
}

function report(reporter: CockpitUpdateReporter | undefined, result: CockpitUpdateResult) {
  reporter?.(result);
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('uiai-cockpit-update', { detail: result }));
    if (result.phase === 'available') {
      pushToast({ id: 'cockpit-ota', level: 'info', title: `Cockpit ${result.version ?? ''} update`, message: result.message, durationMs: 0 });
    } else if (result.phase === 'downloading') {
      pushToast({ id: 'cockpit-ota', level: 'info', title: `Downloading Cockpit ${result.version ?? ''}`, message: result.message, progress: result.totalBytes ? result.downloadedBytes! / result.totalBytes : undefined, durationMs: 0 });
    } else if (result.phase === 'installing') {
      pushToast({ id: 'cockpit-ota', level: 'success', title: `Cockpit ${result.version ?? ''} installed`, message: result.message, progress: 1, durationMs: 0 });
    } else if (result.phase === 'error') {
      pushToast({ id: 'cockpit-ota', level: 'error', title: 'Cockpit update failed safely', message: result.message });
    }
  }
  return result;
}

function recordActivationReceipt(version: string): void {
  try {
    window.localStorage.setItem(
      COCKPIT_UPDATE_RECEIPT_KEY,
      JSON.stringify({
        schema: 'uiai.cockpit_ota_activation_receipt.v1',
        status: 'installed_relaunching',
        version,
        activated_at: new Date().toISOString(),
        activation: 'signed_updater_install_and_relaunch',
      }),
    );
  } catch {
    // The signed installer and relaunch remain authoritative.
  }
}

export async function runCockpitUpdate(options: {
  install: boolean;
  reporter?: CockpitUpdateReporter;
}): Promise<CockpitUpdateResult> {
  const { install, reporter } = options;
  if (!isTauriRuntime()) {
    return report(reporter, {
      phase: 'unavailable',
      message: 'Background updates are available in the installed Cockpit app.',
    });
  }

  report(reporter, { phase: 'checking', message: 'Checking for signed Cockpit updates…' });
  try {
    const update = await check();
    if (!update) {
      return report(reporter, { phase: 'current', message: 'Cockpit is up to date.' });
    }

    if (!install) {
      return report(reporter, {
        phase: 'available',
        message: `Cockpit ${update.version} is signed and ready.`,
        version: update.version,
      });
    }

    let downloadedBytes = 0;
    let totalBytes: number | undefined;
    report(reporter, {
      phase: 'available',
      message: `Cockpit ${update.version} is signed; installing in the background…`,
      version: update.version,
    });
    await update.downloadAndInstall((event) => {
      if (event.event === 'Started') {
        totalBytes = event.data.contentLength ?? undefined;
      } else if (event.event === 'Progress') {
        downloadedBytes += event.data.chunkLength;
      } else if (event.event === 'Finished') {
        report(reporter, {
          phase: 'installing',
          message: `Installing signed Cockpit ${update.version}; relaunching…`,
          version: update.version,
          downloadedBytes,
          totalBytes,
        });
      }
      if (event.event === 'Started' || event.event === 'Progress') {
        report(reporter, {
          phase: 'downloading',
          message: `Downloading signed Cockpit ${update.version}…`,
          version: update.version,
          downloadedBytes,
          totalBytes,
        });
      }
    });
    recordActivationReceipt(update.version);
    await relaunch();
    return report(reporter, {
      phase: 'installing',
      message: `Verified Cockpit ${update.version} installed; relaunch requested.`,
      version: update.version,
    });
  } catch (error) {
    return report(reporter, {
      phase: 'error',
      message: `Signed Cockpit update failed safely: ${error instanceof Error ? error.message : String(error)}`,
    });
  }
}

/** Surface the durable activation receipt after the updater relaunch completes. */
export function reportCompletedCockpitUpdate(): CockpitUpdateActivationReceipt | undefined {
  try {
    const raw = window.localStorage.getItem(COCKPIT_UPDATE_RECEIPT_KEY);
    if (!raw) return undefined;
    const receipt = JSON.parse(raw) as Partial<CockpitUpdateActivationReceipt>;
    if (receipt.schema !== 'uiai.cockpit_ota_activation_receipt.v1' || receipt.status !== 'installed_relaunching' || !receipt.version) return undefined;
    window.localStorage.removeItem(COCKPIT_UPDATE_RECEIPT_KEY);
    pushToast({
      id: 'cockpit-ota',
      level: 'success',
      title: `Cockpit ${receipt.version} updated`,
      message: 'Background update completed and the desktop app relaunched successfully.',
      progress: 1,
      durationMs: 8000,
    });
    return receipt as CockpitUpdateActivationReceipt;
  } catch {
    return undefined;
  }
}

/** Check shortly after startup, then poll the dev channel for near-immediate signed updates. */
export function startAutomaticCockpitUpdate(reporter?: CockpitUpdateReporter): () => void {
  let inFlight: Promise<CockpitUpdateResult> | undefined;
  const run = () => {
    if (inFlight) return;
    inFlight = runCockpitUpdate({ install: true, reporter }).finally(() => { inFlight = undefined; });
  };
  const startupTimer = window.setTimeout(run, 2500);
  const pollTimer = window.setInterval(run, 5_000);
  return () => {
    window.clearTimeout(startupTimer);
    window.clearInterval(pollTimer);
  };
}
