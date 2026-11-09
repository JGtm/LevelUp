# ADR 0046 — Media identity and reconciliation with the disk

Status: accepted (2026-11-08); implemented in the media work of the same date (steps M1 to M7, reviewed in M8).
Source: decisions D1 to D17 of the media plan, of which D1 to D13, D16 and D17 are recorded here; D14 and D15 belong to
the KPI revision and are not part of this ADR.
Relates to: ADR 0026 (append-only tables and `_latest` views), ADR 0039 (no unfiltered reads of the raw journal),
ADR 0044 (deployment capabilities), ADR 0045 (desktop mode: the shell holds no business logic).

## Vocabulary

- **Original**: the file the user captured, as found in the capture folder.
- **Converted copy**: the HLS copy the app produces for a video whose container the browser cannot play.
- **Missing row**: a `media_files` row whose `status` is `missing` (`domain.MediaStatusMissing`). Its file is not on disk
  at the last reconciliation pass.
- **Deleted row**: a row whose `status` is `deleted`. The app hid it on purpose; it is never shown again unless the
  file comes back under a new scan.

## Problem

Before this ADR the media index keyed each item by its path, and a row disappeared from the gallery only when the
app itself deleted it. Renaming or moving a file in the Windows explorer therefore created a duplicate item and lost its
likes and match association. Deleting a file in the explorer left a row pointing at nothing, shown as a broken item.
Likes are keyed by `file_path`, so any path change had to move them.

## Decision

- **D1 — Identity.** A media item's identity is its content hash (`file_hash`). The path is an attribute that changes.
- **D2 — Columns.** `media_files` gains three non-indexed columns: `source_path VARCHAR`, `source_size BIGINT`,
  `source_mtime_ns BIGINT`. `source_path` holds the stored path of the original; NULL means no original is expected.
- **D3 — State.** A file that is not on disk is `status = 'missing'`. No new column is added, because 24 files create
  `media_files` by hand and would break. When the file returns, `status` goes back to NULL.
- **D4 — Presence.** A converted video (`hls_path` set) is present when either the converted copy or the original exists.
  Any other item is present when its file (`source_path`, else `file_path`) exists. A converted video whose original
  disappears stays visible and plays from its converted copy, because keeping only one of the two formats is a legitimate
  cleanup; its `source_path` goes back to NULL. If the original returns, it is recognised by hash.
- **D5 — Automatic relocation.** A row is moved to a new path only on an identical hash, or on a change of container
  (same folder, same name without extension, a single candidate). There is no approximate matching.
- **D6 — Likes follow the path.** Likes are indexed by `file_path`. Any change of `file_path` transfers the likes by
  INSERT into the history table, never by UPDATE.
- **D7 — No automatic purge.** Forgetting a missing item is always the user's decision.
- **D8 — Forget.** Forgetting never touches the original. It removes only the files the app created (thumbnail, HLS copy)
  and hides the row (`status = 'deleted'`).
- **D9 — Delete from the app.** Deleting also removes the original, so the item does not come back at the next scan. On
  the desktop the original goes to the Windows Recycle Bin, with an explicit message before confirmation. Where there is no
  Recycle Bin (USB key, network share), the deletion is permanent and the message and the toast say so. A failure to send
  to the Recycle Bin never turns into a permanent deletion. Thumbnail and converted copy are always deleted permanently.
  An original restored from the Recycle Bin brings the item back at the next scan. Online, deletion stays permanent, as before.
- **D10 — Safety guards.** Marking is cancelled when the root folder is unavailable. Marking is suspended when at least 20
  active rows exist and half or more would become missing in one pass: a banner asks for confirmation, and the
  confirmation applies the marking on the next pass, which runs at once. Relocations are always applied.
- **D11 — No watcher.** There is no continuous watch on the folder. A pass runs at startup (background task) and after
  each synchronisation.
- **D12 — Missing-files screen.** It reuses the shell of the match picker, extracted without visual change. No new
  interface pattern.
- **D13 — Desktop originals.** On the desktop, deleting the original after conversion is forbidden
  (capability `KeepMediaSources`, true on desktop). The setting is hidden and the environment variable is ignored.
- **D16 — Manual reattachment is a merge.** A scan already indexes any unknown file, so a re-encoded clip has its own
  item. Candidates are the other present items of the same player and kind. Confirming transfers the likes and the current
  match association of the missing item (manual or automatic) to the chosen item, where it becomes manual, unless the chosen
  item already has a manual association. Then the missing item is forgotten (D8).
- **D17 — Original back, copy gone.** A converted video whose original is present but whose app copy is absent returns to
  direct playback: `file_path = source_path`, `hls_path` and `transcode_status` set to NULL, and the background scan converts
  it again.

## Invariants enforced by code and tests

- `file_stem` is written once, at insertion, and never updated (DuckDB ART bug #23645 applies to indexed columns).
  `player_slug` and `created_at` are not updated either.
- No `DELETE` on `media_files`; `media_likes_history` is INSERT only.
- Every read of `media_files` carries a visibility predicate (`MediaVisiblePredicate` or `MediaNotDeletedPredicate`);
  a guard test enumerates the reads.
- Child processes go through `internal/platform/childproc`.

## Consequences

- A rename or move of a file in the explorer keeps the item, its likes, its thumbnail and its match association.
- A file removed in the explorer leaves the gallery and appears in the missing-files screen; putting it back restores it.
- Two files with the same name in two sub-folders are two distinct items, each with its own thumbnail and conversion.
- A pass that finds no change on disk does not re-read indexed files in full.
- Known limits, not decided here: two rows sharing one `file_path` can both be marked by a forget; a thumbnail of a deleted
  row is not counted as taken by the thumbnail naming check; a file renamed and then deleted from the app before any pass
  keeps its stale stored path, so the original is not sent to the Recycle Bin, and a different file that now sits at that
  path would be sent instead.
