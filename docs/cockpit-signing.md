# Cockpit signing and credential authority

Cockpit uses identifier `com.wpuiai.uiaiengine.cockpit`, hardened runtime, and the entitlements in `apps/cockpit/src-tauri/entitlements.plist`.

## Credential boundary

- Cockpit's Keychain access group is `$(AppIdentifierPrefix)com.wpuiai.uiaiengine.cockpit`.
- It is intentionally distinct from Focusa Menubar credential groups.
- The shared application group `group.com.focusa.shared` carries only the non-secret `focusa.platform_hint.v1` document.
- Pairing mints a distinct `client_type=cockpit` token. Updates and rollbacks retain the same application identifier and therefore the same Cockpit-only Keychain authority.
- No build may add `com.focusa.shared-focusa` to Cockpit Keychain groups.

## Release gate

Run:

```bash
python3 scripts/audit-cockpit-entitlements.py
codesign -d --entitlements :- "UIAI Engine Cockpit.app"
```

The first command is required during development and CI. The second is required on the signed package during the platform/release gate; its effective entitlements must match the source plist. Notarization, update, and rollback artifacts must retain the same identifier and access group.
