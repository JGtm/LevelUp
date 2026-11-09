# LevelUp for Windows - third-party notices

LevelUp is free software (GNU Affero General Public License, version 3 only). The full text is in
the `LICENSE` file installed next to this one; the licence terms and what they do not cover are in
`NOTICE`. Source code: https://github.com/JGtm/LevelUp

(C) JGtm - LevelUp - https://github.com/JGtm/LevelUp

## Halo content (Microsoft)

Halo Infinite © Microsoft Corporation. LevelUp was created under Microsoft's "Game Content Usage Rules" using assets from Halo Infinite, and it is not endorsed by or affiliated with Microsoft.

Game Content Usage Rules: https://www.xbox.com/en-US/developers/rules

Halo, Halo Infinite, Halo 5: Guardians and Xbox, and every image, sound and piece of data taken from
the games or their services, belong to Microsoft Corporation and its licensors. LevelUp is a free
application without advertising.

Game sound effects (the replay sounds) are the property of Microsoft and its licensors, and are
removed on simple request: open an issue at https://github.com/JGtm/LevelUp/issues.

## ffmpeg and ffprobe (LGPL)

LevelUp installs the programs `ffmpeg.exe` and `ffprobe.exe` and their libraries (`avcodec`,
`avdevice`, `avfilter`, `avformat`, `avutil`, `swresample`, `swscale`, `.dll`) next to the
application. They are used for thumbnails, video encoding and stream conversion.

- Project: FFmpeg, https://ffmpeg.org - Copyright (c) the FFmpeg developers.
- Licence: GNU Lesser General Public License, version 3 (file `LICENSE.LGPL` installed next to the
  application). This build was configured without any GPL component (`--enable-version3`,
  without `--enable-gpl`; no libx264, no libx265).
- Build: BtbN/FFmpeg-Builds, release `autobuild-2026-10-31-12-59`, archive
  `ffmpeg-n8.1.3-27-g8dca1586f4-win64-lgpl-shared-8.1.zip`
  (SHA-256 `5cef7bc48e513f3cee7c881e8b2999d835541edb1846c3e89443a67f8f8556ab`).
  https://github.com/BtbN/FFmpeg-Builds/releases/tag/autobuild-2026-10-31-12-59
- Corresponding source: FFmpeg release 8.1.3, https://ffmpeg.org/releases/ffmpeg-8.1.3.tar.xz
  (the build is made from tag `n8.1.3` plus commits up to `g8dca1586f4`); the build scripts and
  the patches applied are in https://github.com/BtbN/FFmpeg-Builds
- The libraries are dynamic: you may replace the `.dll` files and the executables by another
  build of the same FFmpeg version; LevelUp looks for them first in its own folder.

## Font

Roboto, Copyright 2011 The Roboto Project Authors (https://github.com/googlefonts/roboto-classic),
SIL Open Font License, Version 1.1 (https://openfontlicense.org). Served by the application.

## Interface icons (Lucide, ISC)

The navigation and footer icons of the application (sidebar, About, settings and others) are copied
from Lucide, https://lucide.dev, as SVG (no library is installed): `layout-dashboard`, `crosshair`,
`swords`, `award`, `flame`, `globe`, `film`, `search`, `book-open`, `bug`, `book-open-text`,
`settings`, `sun-moon`, `info`, `panel-left-close`, `panel-left-open`, `ellipsis`, `pin`, `pin-off` (lucide-static 0.552.0).

ISC License

Copyright (c) for portions of Lucide are held by Cole Bemis 2013-2022 as part of Feather (MIT).
All other copyright (c) for Lucide are held by Lucide Contributors 2022.

Permission to use, copy, modify, and/or distribute this software for any purpose with or without
fee is hereby granted, provided that the above copyright notice and this permission notice appear
in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH REGARD TO THIS
SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE
AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT,
NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE
OF THIS SOFTWARE.


## Libraries

Third-party libraries keep their own licences:

- Rust dependencies of the application shell: the list with versions and licences is
  `THIRD_PARTY_CARGO.md`, installed next to this file.
- Go dependencies of the server (DuckDB, chi, Huma and others): `apps/go-api/go.mod` and
  `libs/*/go.mod` in the source repository.
- JavaScript dependencies of the interface (React, TanStack, ECharts and others):
  `apps/web/package-lock.json` in the source repository.
- Microsoft Edge WebView2 runtime, installed if missing, under Microsoft's licence terms
  (https://developer.microsoft.com/microsoft-edge/webview2).
