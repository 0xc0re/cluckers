# Phase 5: Containers / AppImage - Context

**Gathered:** 2026-02-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Package cluckers as an AppImage that bundles all dependencies including Proton-GE, so Linux users can go from download to playing with zero system-level dependency management. Static CLI binary continues to ship alongside for headless/CLI-only use.

</domain>

<decisions>
## Implementation Decisions

### Packaging Format
- AppImage as the primary packaging format
- Static CLI binary remains available alongside AppImage in GitHub releases
- Target Ubuntu 22.04 LTS (glibc 2.35) as the build baseline
- Include AppImageUpdate (zsync) support for delta self-updates

### Dependency Bundling
- Bundle everything: Fyne graphics libs + Proton-GE — true zero-dependency experience
- Proton-GE variant for best game compatibility (DXVK, fsync, game patches baked in)
- Wine prefix stays external at ~/.cluckers/prefix/ (not inside AppImage data dir)

### Build Pipeline
- AppImage built in GitHub Actions alongside existing CI
- Local build script (scripts/build-appimage.sh) for dev iteration and testing
- Must produce AppImage artifact attached to the same GitHub release as the static binary

### Distribution Channels
- GitHub releases only (no Flathub, AUR, or other repos)
- Default install instructions (curl one-liner) point to AppImage download
- Standard AppImage naming: Cluckers-x86_64.AppImage
- Auto-integrate .desktop file and icon on first run (standard AppImage desktop integration)

### Claude's Discretion
- Goreleaser integration approach (nfpm plugin vs post-build step)
- Whether Proton-GE is baked in at build time or downloaded on first run (tradeoff: ~1GB artifact vs setup step)
- Wine/Proton-GE update strategy (new AppImage release vs separate download)
- Exact Proton-GE version to pin
- AppImage internal directory structure
- zsync file generation details

</decisions>

<specifics>
## Specific Ideas

- User explicitly wants Wine bundled because "Wine seems to be an issue" — the whole point is removing Wine management from the user
- zsync delta updates keep the large AppImage manageable for updates
- The AppImage should feel like a native app — download, chmod +x, run

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-containers-appimage*
*Context gathered: 2026-02-24*
