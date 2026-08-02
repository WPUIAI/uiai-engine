import { relaunch } from '@tauri-apps/plugin-process';
import { check } from '@tauri-apps/plugin-updater';

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

function isTauriRuntime(): boolean {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;
}

function report(reporter: CockpitUpdateReporter | undefined, result: CockpitUpdateResult) {
  reporter?.(result);
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('uiai-cockpit-update', { detail: result }));
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

/** Check shortly after startup and install signed releases without blocking the UI. */
export function startAutomaticCockpitUpdate(reporter?: CockpitUpdateReporter): () => void {
  const timer = window.setTimeout(() => {
    void runCockpitUpdate({ install: true, reporter });
  }, 2500);
  return () => window.clearTimeout(timer);
}
