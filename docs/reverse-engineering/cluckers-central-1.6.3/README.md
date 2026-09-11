# Cluckers Central 1.6.3 reverse-engineering notes

Static analysis of the official Project Crown Windows launcher (Tauri 2 / Rust,
`LAUNCHER_VERSION=1.6.3`), captured on 2026-09-11 while migrating cluckers to
the v1.6.3 gateway protocol.

- `protocol.md` — gateway endpoints, request/response fields, headers, login /
  link / PIN-gate flow, websocket, Tauri command surface.
- `launch-and-updater.md` — exact game command line, shared-memory mechanism,
  install layout, updater API (version.json, manifest, repair index, minisign),
  INI presets, self-update.
- `cluckers-code-map.md` — map of our Go code at commit f51ec81 with file:line
  references for every gateway call site, plus live gateway probe results.
- `frontend-index.js` — the launcher's frontend bundle, decompressed from the
  brotli asset table in the binary. Vanilla TypeScript built by Vite, minified.

The launcher binary itself is not committed.
