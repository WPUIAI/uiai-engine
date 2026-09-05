# Filesystem publication boundary

Artifact and delivery stores share this platform adapter. Callers flush file contents before publication and validate/sync changed parent directories afterwards.

- Unix: filesystem rename followed by parent-directory `fsync`.
- Windows: `MoveFileEx` with `REPLACE_EXISTING | WRITE_THROUGH`, without cross-volume copy fallback. Directory validation replaces the unsupported POSIX directory flush operation; file flushing remains mandatory.

Windows write-through is not a claim of POSIX-equivalent power-loss durability. Native Windows store/replay/restore tests and independent crash acceptance remain required; cross-compilation alone proves neither.
