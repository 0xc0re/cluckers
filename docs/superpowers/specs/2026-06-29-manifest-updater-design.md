# Manifest-Based Game Updater — Design

**Date:** 2026-06-29
**Status:** Approved
**Author:** Claude + chris

## Problem

`cluckers update` is broken. The updater backend
(`https://updater.realmhub.io/builds/version.json`) migrated from a single
monolithic `game.zip` download to a **per-file manifest scheme**, and the
client was never updated to match.

The current `version.json` no longer contains `zip_url`, `zip_blake3`, or
`zip_size`. The code decodes `info.ZipURL` to `""`, so the download fails with:

```
Get "": unsupported protocol scheme ""
```

The new backend serves:

- `version.json` with `base_url`, `manifest_url`, `gameversion_dat_*`, plus
  delta/repair/minisign fields.
- A per-version manifest (`manifest_url`) listing every game file with its
  relative path, BLAKE3 hash, and size (1,257 files, ~8.05 GB for v0.37.6744.0).
- Each file downloadable at `base_url + "/" + path` (verified HTTP 200).
- The old `…/game.zip` returns 404 — it is gone.

## Scope

In scope: a basic manifest-based updater that fetches the manifest, downloads
changed/missing files, verifies per-file BLAKE3, and produces an install that
exactly matches the manifest.

Out of scope (explicitly deferred): delta patching (`delta_enabled`,
`patch_threshold_bytes`), the repair index (`repair_index_url`), and minisign
signature verification (`content_sig_scheme`). These fields are left unparsed.

## Design Decisions

- **Stale files:** clean sync — delete local files under the game tree that are
  not in the manifest, so the install exactly matches the server.
- **Concurrency:** bounded parallel worker pool (6–8 concurrent downloads).
- **No new dependencies:** hand-rolled worker pool; reuse existing `blake3`,
  `progressbar`, and platform `checkDiskSpace`.

## Data Model (`internal/game/version.go`)

`VersionInfo` drops the dead zip fields and keeps the manifest-scheme fields:

```go
type VersionInfo struct {
    LatestVersion        string `json:"latest_version"`
    BaseURL              string `json:"base_url"`
    ManifestURL          string `json:"manifest_url"`
    GameVersionDatPath   string `json:"gameversion_dat_path"`
    GameVersionDatBLAKE3 string `json:"gameversion_dat_blake3"`
    GameVersionDatSize   int64  `json:"gameversion_dat_size"`
}
```

New manifest types:

```go
type Manifest struct {
    Schema  int            `json:"schema"`
    Version string         `json:"version"`
    Files   []ManifestFile `json:"files"`
}

type ManifestFile struct {
    Path string `json:"path"` // relative, forward-slash separated
    Hash string `json:"hash"` // BLAKE3 hex
    Size int64  `json:"size"`
}
```

New `FetchManifest(ctx, info) (*Manifest, error)` — GET `info.ManifestURL`,
parse JSON, validate `Schema == 1`, return a `*ui.UserError` on failure.

## Sync Engine (new `internal/game/sync.go`)

A single function replaces both `DownloadAndVerify` and `ExtractZip`:

```go
func SyncManifest(ctx context.Context, info *VersionInfo, m *Manifest,
    gameDir string, onProgress ProgressFunc) error
```

Steps:

1. **Diff.** For each manifest file, hash the local copy if present. A file
   needs downloading if it is missing, the wrong size, or the wrong BLAKE3.
2. **Disk check.** Sum the sizes of files that need downloading and call the
   existing platform `checkDiskSpace(gameDir, required)` (no `×2`, no zip).
3. **Mark incomplete.** Write a sync sentinel so an interrupted sync forces a
   re-sync next run (reuses/renames the existing "extraction incomplete" idea).
4. **Download (bounded worker pool, 6–8 concurrent).** Each needed file is
   fetched from `base_url + "/" + path` into a temp file in the destination
   directory, verified against its BLAKE3, then atomically renamed into place.
   Parent directories are created as needed. A **path-traversal guard** ensures
   the resolved destination stays under `gameDir` (the same protection the old
   zip-slip code provided). Transient errors retry 2–3×; the first hard error
   cancels the pool via context.
5. **Clean sync.** Walk `gameDir` and delete any file not present in the
   manifest set.
6. **Clear the sentinel** on full success.

Progress: a single aggregated `ProgressFunc(downloaded, total)` callback over
cumulative downloaded bytes vs. total bytes to download, replacing the old
separate download and extract progress bars.

## Caller Changes

| Caller | Change |
|---|---|
| `cli/update.go` | `FetchVersionInfo → NeedsUpdate → FetchManifest → SyncManifest → re-check NeedsUpdate`. Remove the zip download/extract path. |
| `launch/pipeline.go` (prep download steps) | Same swap; the prep pipeline's download + extract steps become a single sync step. |
| `gui/screens/main.go` | Update **and** repair flows call `SyncManifest`; collapse the two progress bars into one. Repair runs an unconditional sync (sync already re-hashes every local file). |
| `cli/status.go` | Unchanged — still `FetchVersionInfo` + `NeedsUpdate`. |

`NeedsUpdate` keeps using the `GameVersion.dat` BLAKE3 hash as the cheap
"is an update available?" gate — unchanged and still correct, since the
manifest includes `GameVersion.dat`.

Dead code removed: zip-based `DownloadGameZip*`, `DownloadAndVerify*`, and the
zip `ExtractZip*` functions. The sync sentinel helper replaces
`IsExtractionIncomplete`.

## Error Handling

- All user-facing failures return `*ui.UserError` with Message/Detail/Suggestion.
- A failed or cancelled sync leaves the sentinel in place; the next run re-syncs.
- A per-file BLAKE3 mismatch after retries aborts the sync with a clear error.
- Context cancellation (Ctrl-C) stops the pool promptly and is reported as
  cancellation, not corruption.

## Testing (TDD)

`httptest.Server` serves a fixture `manifest.json` plus the referenced files.
Cases:

- Manifest parse (valid; wrong schema rejected).
- Downloads missing files.
- Skips files whose local hash already matches.
- Re-downloads files whose hash mismatches.
- Deletes stale local files not in the manifest (clean sync).
- Rejects a file whose downloaded bytes fail BLAKE3 verification.
- Rejects path traversal in a manifest `path`.
- Respects context cancellation.
- Worker-pool aggregation downloads all needed files exactly once.

Existing `NeedsUpdate` tests remain green. The GUI must still compile under
`-tags gui`. All tests use `t.TempDir()` + `t.Setenv("CLUCKERS_HOME", tmp)`.

## Non-Goals

- Delta/patch downloads, repair index, minisign verification.
- Byte-range resume within a single file (files are re-fetched whole on retry;
  whole-install resume is handled by the per-file diff on the next run).
