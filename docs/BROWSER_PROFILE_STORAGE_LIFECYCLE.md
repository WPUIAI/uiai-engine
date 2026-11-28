# Managed Chromium profile storage lifecycle

Explicit `browser.profiles.<name>.launch.user_data_dir` is operator-owned. UIAI must not delete it when closing a runtime. A browser process without an explicit directory continues to own its temporary storage. A persistent profile requires an explicit directory and an exclusive reservation, and cannot also request an ephemeral storage context. External adapters that cannot honor the directory reject the setting rather than pretending it is active.

The manager canonicalizes directory paths, including existing symlinked parents, and retains the reservation until the browser runtime finishes closing. Chromium's process-level locking remains necessary across separate engine processes. Do not point a service at a personal browser's default directory or share a profile between different users. The standard vision pool remains ephemeral and per-session isolated; it is not converted into a persistent shared pool.

Rod v0.116.2's standalone launcher Cleanup waits for the child exit and removes the current user-data directory flag. The profile runtime clears that cleanup-only flag after the browser has been launched when the directory was explicitly configured, then waits for exit normally. This does not change browser identity, challenge handling, automation disclosure, or security controls.

Regression acceptance: cookies and localStorage persist after a full close/relaunch of the same explicitly configured profile; default ephemeral isolation remains intact; directory aliases cannot obtain concurrent manager reservations; shutdown holds the reservation until the runtime returns; unmanaged command-line overrides are refused. Browser-backed tests require installed Chromium and must be verified in the Browser Reliability CI lane.
