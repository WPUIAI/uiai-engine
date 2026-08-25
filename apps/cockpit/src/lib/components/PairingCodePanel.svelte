<script lang="ts">
  import type { PairingPresentationV1 } from "$lib/pairing/pairing-presentation";
  let { presentation }: { presentation: PairingPresentationV1 } = $props();
</script>

<section class="pairing-code" aria-labelledby="pairing-code-title">
  <h2 id="pairing-code-title">Connect Cockpit to Focusa</h2>
  <p>Open the secure pairing page on your phone or VPS, then confirm this code.</p>
  <output aria-label="Pairing code">{presentation.display_code}</output>
  <a href={presentation.pair_href} rel="noreferrer noopener" target="_blank">Open pairing page</a>
  <p class="expiry">This request expires at {presentation.expires_at}.</p>
</section>

<style>
  .pairing-code{display:grid;gap:.75rem;max-width:34rem;padding:1rem;border:1px solid var(--border-color,#444);border-radius:.75rem}
  output{font:700 1.5rem/1.2 ui-monospace,monospace;letter-spacing:.12em;overflow-wrap:anywhere}
  .expiry{font-size:.875rem;opacity:.75}
</style>
