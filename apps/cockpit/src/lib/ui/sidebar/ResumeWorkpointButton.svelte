<script lang="ts">
  import type { CockpitWorkpointResume } from "$lib/contracts/workpoint-resume";

  export let state: CockpitWorkpointResume;
</script>

{#if state.status === "resumable"}
  <a class="resume-workpoint" href={state.target.href} aria-label={`Resume Workpoint: ${state.label}`}>
    <span aria-hidden="true">↗</span>
    <span>
      <strong>Resume Workpoint</strong>
      <small>{state.label}</small>
    </span>
  </a>
{:else if state.status === "blocked"}
  <div class="resume-recovery" role="status">
    <span>{state.recovery.message}</span>
    <a href={state.recovery.href}>{state.recovery.action_label} →</a>
  </div>
{/if}

<style>
  .resume-workpoint,
  .resume-recovery {
    box-sizing: border-box;
    width: calc(100% - 20px);
    margin: 0 10px 10px;
    border: 1px solid var(--line, #d8dde8);
    border-radius: 10px;
    background: var(--surface-raised, #fff);
    color: inherit;
  }

  .resume-workpoint {
    display: flex;
    align-items: center;
    gap: 9px;
    padding: 9px 10px;
    text-decoration: none;
  }

  .resume-workpoint span:last-child {
    display: grid;
    min-width: 0;
  }

  .resume-workpoint strong,
  .resume-workpoint small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .resume-workpoint small {
    color: var(--text-muted, #667085);
  }

  .resume-workpoint:focus-visible,
  .resume-recovery a:focus-visible {
    outline: 3px solid var(--focus-ring, #4f46e5);
    outline-offset: 2px;
  }

  .resume-recovery {
    display: grid;
    gap: 5px;
    padding: 9px 10px;
    color: var(--text-muted, #667085);
    font-size: 0.78rem;
  }

  .resume-recovery a {
    color: inherit;
    font-weight: 700;
  }
</style>
