> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462492569

## Operator requirement amendment — portability + responsive contract

The Browser Proof PWA must be portable and mobile-first.

### Mandatory default viewport matrix
- Mobile: **375px**
- Tablet: **768px**
- Small desktop: **1024px**
- Large desktop: **1440px**

Publishers may add custom viewports, but proof-mode defaults must always include these four unless explicitly overridden and recorded in the manifest.

### Fluid layout requirements
- Start from the 375px/mobile layout.
- Use CSS Grid/Flexbox and `clamp()` for typography, spacing, card dimensions, and media sizing between breakpoints.
- No breakpoint-only cliff layouts; intermediate widths must remain usable without clipping or horizontal page overflow.
- Screenshots may scroll inside their own bounded frame, but the report page itself must remain fluid.

### Server portability
- No hard-coded UIAI hostname, port, CDN, or deployment path.
- PWA HTML, manifest, JSON proof manifest, and image assets use same-origin/relative routes.
- API response derives an absolute operator-facing `artifact_url` from trusted request/proxy origin configuration while preserving a relative canonical path.
- Storage derives only from configured UIAI data/artifact directories.
- Works unchanged on localhost, LAN, tailnet, reverse proxy, tunnel, and public remote deployments.
- Dependency-free renderer; no external JS/CSS availability requirement.

### Additional acceptance
1. Visual tests at 375/768/1024/1440 plus one intermediate fluid width.
2. No page-level horizontal overflow at any required width.
3. Published link works on both a loopback-only test server and a reverse-proxy-origin test.
4. Artifact remains usable after browser session closure and engine restart.
