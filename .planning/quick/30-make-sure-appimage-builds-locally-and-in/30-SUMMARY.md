---
phase: quick-30
plan: 01
subsystem: infra
tags: [appimage, linuxdeploy, appimagetool, proton-ge, ci, fuse]

# Dependency graph
requires:
  - phase: 05-containers-appimage
    provides: AppImage build script, CI release workflow, deploy assets
provides:
  - Working local AppImage build pipeline producing Cluckers-x86_64.AppImage
  - CI release workflow fixed for FUSE-less ubuntu-22.04 runners
affects: [release, appimage, ci]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "NO_STRIP=true for linuxdeploy on modern distros with .relr.dyn ELF sections"
    - "APPIMAGE_EXTRACT_AND_RUN=1 for CI runners without FUSE"
    - "cd into OUTPUT_DIR before appimagetool to control zsync file placement"

key-files:
  created: []
  modified:
    - scripts/build-appimage.sh
    - .github/workflows/release.yaml
    - .gitignore

key-decisions:
  - "NO_STRIP=true added to linuxdeploy: bundled strip binary too old for .relr.dyn ELF sections on modern distros"
  - "APPIMAGE_EXTRACT_AND_RUN=1 for CI: ubuntu-22.04 runners lack FUSE, AppImage tools need extract-and-run mode"
  - "appimagetool run from dist/ directory to ensure zsync file lands next to AppImage output"
  - "build/ directory added to .gitignore for AppImage build artifacts (Proton cache, AppDir)"

patterns-established:
  - "AppImage build script validated end-to-end: binary build, Proton bundle, linuxdeploy, appimagetool"

requirements-completed: [VERIFY-APPIMAGE-LOCAL, VERIFY-APPIMAGE-CI]

# Metrics
duration: 5min
completed: 2026-02-25
---

# Quick Task 30: AppImage Build Verification Summary

**Fixed linuxdeploy strip compatibility, zsync output path, and CI FUSE workaround for working AppImage pipeline**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-25T01:06:15Z
- **Completed:** 2026-02-25T01:11:09Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Local AppImage build completes successfully producing 444MB Cluckers-x86_64.AppImage with bundled Proton-GE
- Fixed linuxdeploy strip failure on modern distros (Arch, Ubuntu 24+) with .relr.dyn ELF sections
- Fixed zsync file output location (was landing in project root instead of dist/)
- Added FUSE workaround for CI runners (APPIMAGE_EXTRACT_AND_RUN=1)
- AppImage responds to --version and is executable

## Task Commits

Each task was committed atomically:

1. **Task 1: Install prerequisites and run local AppImage build** - `4631220` (fix)
2. **Task 2: Audit and fix CI release workflow for AppImage build** - `cea7ee9` (ci)

## Files Created/Modified
- `scripts/build-appimage.sh` - Added NO_STRIP=true for linuxdeploy, fixed appimagetool cwd for zsync output
- `.github/workflows/release.yaml` - Added APPIMAGE_EXTRACT_AND_RUN=1 env var for Build AppImage step
- `.gitignore` - Added build/ directory to exclude AppImage build artifacts

## Decisions Made
- **NO_STRIP=true**: linuxdeploy's bundled strip binary (from its embedded AppImage toolchain) cannot handle newer ELF `.relr.dyn` section types. Since the Go binary is already built with `-s -w` ldflags and system libraries are pre-stripped by package managers, skipping strip is safe and correct.
- **APPIMAGE_EXTRACT_AND_RUN=1**: Ubuntu 22.04 CI runners do not have FUSE available. This env var tells AppImage-format tools (linuxdeploy, appimagetool) to extract themselves to a temp directory before running, bypassing the FUSE requirement.
- **appimagetool cwd**: appimagetool always writes the zsync file to the current working directory, not next to the output file. Running it from dist/ via a subshell ensures both outputs land together.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed linuxdeploy strip failure on modern ELF binaries**
- **Found during:** Task 1 (local AppImage build)
- **Issue:** linuxdeploy's bundled strip binary fails on .relr.dyn ELF sections from Arch Linux's newer libraries, causing non-zero exit code that kills the build script
- **Fix:** Added NO_STRIP=true environment variable to linuxdeploy invocation in bundle_libraries()
- **Files modified:** scripts/build-appimage.sh
- **Verification:** Full build completes successfully, libraries copied and rpath set correctly
- **Committed in:** 4631220

**2. [Rule 1 - Bug] Fixed zsync file output location**
- **Found during:** Task 1 (local AppImage build)
- **Issue:** appimagetool writes zsync file to cwd, not next to the output AppImage. The zsync file ended up in the project root instead of dist/
- **Fix:** Run appimagetool from OUTPUT_DIR via subshell: `(cd "$OUTPUT_DIR" && appimagetool ...)`
- **Files modified:** scripts/build-appimage.sh
- **Verification:** Both Cluckers-x86_64.AppImage and .zsync appear in dist/ after build
- **Committed in:** 4631220

**3. [Rule 2 - Missing Critical] Added build/ to .gitignore**
- **Found during:** Task 2 (CI audit)
- **Issue:** build/appimage/ directory (containing 1.4GB Proton cache and AppDir) was not gitignored, could be accidentally committed
- **Fix:** Added `build/` to .gitignore
- **Files modified:** .gitignore
- **Verification:** `git status` no longer shows build/ directory
- **Committed in:** cea7ee9

---

**Total deviations:** 3 auto-fixed (2 bugs, 1 missing critical)
**Impact on plan:** All auto-fixes necessary for build correctness. No scope creep.

## Issues Encountered
- zsync not pre-installed on system; installed via Homebrew (sudo not available for pacman). Homebrew provides zsyncmake from the zsync package.
- linuxdeploy and appimagetool not pre-installed; downloaded from GitHub continuous releases to ~/.local/bin/

## Next Phase Readiness
- AppImage build pipeline is verified and ready for next tag push
- CI release workflow has all necessary workarounds (FUSE, strip)
- Proton-GE cache saves ~2GB download on subsequent builds

---
*Quick Task: 30*
*Completed: 2026-02-25*
