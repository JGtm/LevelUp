# PLAN — Transcoding HLS multipiste à l'ingestion média

> Branche : `feat/media-hls-transcoding` (créée depuis `fix/metadata-art-catalog-upsert-invalidation`)
> Statut : Phases 1-6 livrées (POC navigateur Opus GO). Backend + serving + front + CLI complets.

## 1. Contexte & objectif

Le lecteur média sert aujourd'hui les MKV/AVI via un **remux WebM live** ([internal/media/remux.go](../apps/go-api/internal/media/remux.go)) :
ffmpeg `-c copy` vers WebM, exigeant des codecs AV1/VP8/VP9 + Opus/Vorbis, sans Range (pas de seek),
et **sans sélection de piste audio**. Conséquence : les MKV multipistes (game + micro) sont mal lus
selon le navigateur, et tout codec non-WebM fait échouer le remux.

**Objectif** : générer un arbre **HLS-fMP4 à l'upload** (et en backfill), avec **pistes audio
sélectionnables**, seek fonctionnel, sur Chrome/Firefox/Edge.

**Critère de succès** : upload d'un MKV Opus multipiste → clip lisible dans le coverflow avec menu de
sélection de piste audio, sur Chrome/Firefox/Edge ; MKV source supprimé après validation ; historique
existant convertissable via une commande CLI.

## 2. Décisions verrouillées (validées avec l'utilisateur)

| Sujet | Choix | Raison |
|---|---|---|
| Exécution | **Async** via `platform/jobs` + `JobTypeTranscodeMedia` | Infra existante, persistante, reprise après crash |
| Périmètre | **Nouveaux uploads + backfill CLI** | Valeur immédiate sur l'historique |
| Source MKV | **Supprimé** après HLS validé | Thumbnail + ffprobe AVANT suppression |
| Codecs | **Copy par défaut**, cible Chrome/Firefox/Edge | Opus copié tel quel, zéro réencode, zéro perte |
| Segments | **fMP4** (`.m4s` + `init.mp4`) | Permet `-c copy` H.264/HEVC/AV1 + Opus |
| Format état | Colonnes **dédiées** `hls_path` + `transcode_status` | `status='active'` déjà utilisé par le rail home — interdit de le réutiliser |

**Cible navigateur** : Chrome/Firefox/Edge uniquement. Safari/iOS ne lit pas l'Opus en HLS (limitation
Apple) — hors périmètre. Si besoin Safari plus tard : ajouter une rendition AAC (double piste), sans
refonte.

## 3. POC Phase 0 — résultats

ffmpeg disponible : **8.0.1 full build gyan.dev** (muxer `hls`, fMP4, `libopus`, `libx264`, `aac`,
`libwebp`). Aucune installation nécessaire.

**Commande de transcodage éprouvée** (copy pur, multipiste) :

```bash
ffmpeg -i source.mkv \
  -map 0:v:0 -map 0:a:0 -map 0:a:1 \
  -c:v copy -c:a copy \
  -var_stream_map "v:0,agroup:aud a:0,agroup:aud,name:game,default:yes a:1,agroup:aud,name:mic" \
  -master_pl_name master.m3u8 \
  -f hls -hls_segment_type fmp4 -hls_playlist_type vod -hls_time 2 \
  -hls_flags independent_segments \
  -hls_fmp4_init_filename "init_%v.mp4" \
  -hls_segment_filename "out/seg_%v_%03d.m4s" \
  "out/stream_%v.m3u8"
```

**Master playlist généré** (extrait) :

```
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="group_aud",NAME="audio_1",DEFAULT=YES,LANGUAGE="fr",CHANNELS="1",URI="stream_game.m3u8"
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="group_aud",NAME="audio_2",DEFAULT=NO,LANGUAGE="fr",CHANNELS="1",URI="stream_mic.m3u8"
#EXT-X-STREAM-INF:BANDWIDTH=201245,RESOLUTION=640x360,AUDIO="group_aud"
stream_0.m3u8
```

**Preuves** :
- Copy confirmé : vidéo reste `h264`, audio reste `opus` sur les deux pistes (ffprobe sortie).
- Re-décodage du HLS Opus → WAV **6.0135s, zéro erreur**.
- Timestamps **continus** (frames 20ms, aucun trou) → timing fMP4 correct.
- Warning `Error parsing Opus packet header` / `codec frame size is not set` = **faux-positif bénin**
  (ffmpeg cherche un `OpusHead` dans les paquets ; il vit dans la box `dOps`). Apparaît aussi en remux
  MP4 simple, round-trip intact.

**Point de contrôle restant** : lecture réelle hls.js dans le navigateur (MSE `audio/mp4; codecs="opus"`).
Non testable en CLI. Risque faible (fix hls.js documenté + flux prouvé valide). **Go/no-go de Phase 5
avant de retirer le remux WebM legacy.**

## 4. Architecture cible (pipeline)

```
Upload (handler inchangé : PostUploadMedia)
   └─ MediaService.UploadMedia
        ├─ écrit le fichier source (captures/)
        ├─ ffprobe : codecs + pistes audio (langue/titre/canaux)
        ├─ images & vidéos OK mono-piste → flux actuel (file_path direct), FIN
        └─ vidéo MKV/AVI OU multipiste :
             ├─ génère le thumbnail WebP (AVANT suppression source)
             ├─ INSERT media_files : transcode_status='processing', hls_path=NULL
             ├─ jobStore.Create(JobTypeTranscodeMedia)
             └─ go transcodeWorker(...)            ← rend la main immédiatement

transcodeWorker (background)
   ├─ media.BuildHLS(src → captures/{slug}/hls/{stem}/)   [copy ou réencode selon planHLS]
   ├─ valide (master.m3u8 + init + ≥1 segment, ffprobe master OK)
   ├─ UPDATE media_files : hls_path, file_path→master.m3u8, transcode_status='ready'
   ├─ supprime le MKV source
   ├─ job → succeeded ; BumpMediaFeedVersion()   ← galerie rafraîchie (polling existant)
   └─ échec → transcode_status='failed', source conservé, job → failed
```

## 5. Matrice codec (policy `planHLS`, cible Chrome/Firefox/Edge)

| Flux source | Action | Note |
|---|---|---|
| Vidéo H.264 / AV1 | **copy** | Remux pur |
| Vidéo HEVC | copy | Lecture limitée hors Safari (warn log) |
| Vidéo VP8 / VP9 | **réencode H.264** | HLS ne supporte pas VP8/VP9 |
| Audio **Opus** | **copy** | Lu Chrome/FF/Edge via hls.js |
| Audio AAC / MP3 | copy | Compatibles fMP4 |
| Audio Vorbis / autre | réencode AAC | Rare |

Déclencheur HLS : `RequiresRemux(ext)` (mkv/avi) **OU** `len(audioTracks) > 1`. Les MP4/WebM/MOV
mono-piste web-natifs restent servis en direct (inchangé).

## 6. Phases d'implémentation

### Phase 1 — Cœur transcoding (`internal/media/hls.go`)
- `probeStreamsDetailed(ctx, path)` : ffprobe par piste (codec, language, title, channels).
- `needsHLS(ext, streams) bool` : déclencheur — **pur**.
- `planHLS(streams) HLSPlan` : décide copy/réencode par flux + construit le `-var_stream_map` — **pur**.
- `BuildHLS(ctx, src, outDir) (HLSResult, error)` : lance ffmpeg, valide la sortie.
- Tests purs table-driven (`planHLS`, `needsHLS`, `buildVarStreamMap`) + golden d'intégration (skip si ffmpeg absent).
- **Done** : `go test ./internal/media/` vert.

### Phase 2 — Schéma DB & indexation (`internal/ops/`)
- `ensureMediaTables` : `ADD COLUMN IF NOT EXISTS hls_path VARCHAR` + `transcode_status VARCHAR`.
- `walkMediaDir` : **SkipDir sur `hls/`** (comme `thumbs/`) — sinon `init.mp4`/`.m4s` indexés en faux médias.
- Adapter le stem-conflict de `insertMediaFile` + `ReconcileOrphanedMediaFiles` : ne plus présumer le source vidéo présent sur disque.
- **Done** : test arbo prouvant que les fichiers HLS ne sont pas indexés.

### Phase 3 — Orchestration async (`internal/service/` + `internal/domain/job.go`)
- `JobTypeTranscodeMedia`.
- `UploadMedia` : détection + `Create` job + `go transcodeWorker`. Injecter `*jobs.Store` dans `MediaService`.
- `transcodeWorker` : BuildHLS → update DB → delete source → bump feed-version. Reprise des `processing` orphelins au boot.
- **Done** : tests worker (succès/échec) avec `BuildHLS` mocké.

### Phase 4 — Serving (`internal/api/handlers/media.go`)
- `setMediaContentType` : `.m3u8` → `application/vnd.apple.mpegurl`, `.m4s`/`.mp4` segment → `video/mp4`. Bypass remux pour `.m3u8`.
- Le catch-all `/media/files/*` sert déjà l'arbre. Remux WebM legacy conservé pour les médias sans `hls_path`.
- **Done** : httptest Content-Types + non-régression remux legacy.

### Phase 5 — Frontend (`apps/web`)
- Dépendance `hls.js`.
- `ClipPlayer` ([CoverFlowModal.tsx](../apps/web/src/features/media/CoverFlowModal.tsx)) : si `.m3u8` → feature-detect Safari natif sinon hls.js ; menu sélection piste via `hls.audioTracks`. Sinon `<video src>` direct.
- État `processing` : overlay « préparation » sur la vignette ; `useFeedVersion` rafraîchit.
- Étendre `MediaItemRow` + `normalizeMediaItem` : `transcode_status`.
- Tests vitest (hls.js mocké comme echarts, hors sandbox).
- **VALIDATION NAVIGATEUR** : point de contrôle Opus go/no-go.
- **Done** : typecheck + lint + vitest verts + lecture navigateur confirmée.

### Phase 6 — Backfill CLI + formalisation ffmpeg
- `cmd/backfill-media-hls/main.go` (pattern `cmd/reindex-media-thumbs`) : itère les media_files vidéo mkv/avi/multipiste sans `hls_path`, appelle la **même** `BuildHLS`, update DB, supprime source. Idempotent.
- **Formalisation ffmpeg** : prérequis documenté (README/docs setup) + check au démarrage (`healthcheck`) — log clair si absent au lieu d'échecs silencieux.
- **Done** : CLI testée sur un échantillon + doc à jour.

## 7. Stratégie de tests (3 objectifs)

### A. Valider que la pipeline marche
- `planHLS` table-driven (sans ffmpeg) — encode la policy codec.
- `BuildHLS` golden (intégration) : MKV synthétique 2 pistes Opus → asserts master (2 EXT-X-MEDIA), codec audio reste `opus` (preuve copy). `t.Skip` justifié si ffmpeg absent.
- `transcodeWorker` (mock BuildHLS) : succès → ready+source supprimé+feed bump ; échec → failed+source conservé.

### B. Empêcher la régression sur l'existant
- `walkMediaDir` ignore `hls/` (garde-rail permanent — le piège n°1).
- Serving : `.mp4`/`.webm` direct (Range OK), **remux WebM legacy toujours fonctionnel**, image PNG directe.
- `insertMediaFile` dédup stem/hash préservée avec source supprimé.
- Suites existantes (media_test, media_serve_test, media_service_test) vertes sans modifier les assertions.

### C. Figer le comportement
- Golden manifest **structurel** (nb pistes, DEFAULT=YES, attribut CODECS) — invariants, pas byte-exact (ffmpeg varie selon version).
- Contrat API : `MediaItemRow` JSON gèle `transcode_status`/`hls_path` ; garde nil-slice ; `normalizeMediaItem` front.
- Front (hls.js mocké) : `.m3u8` → hls + sélecteur ; `.mp4` → direct ; `processing` → overlay.

## 8. Pièges identifiés

1. **`walkMediaDir`** doit exclure `hls/` — sinon `init.mp4` réindexé en faux média.
2. **`status='active'`** déjà utilisé par le rail home ([queries_home_citations.go:506](../apps/go-api/internal/platform/duckdb/queries_home_citations.go)) → colonnes dédiées.
3. **Thumbnail + ffprobe AVANT suppression** du source.
4. **Dédup hash/stem** (`insertMediaFile`, `ReconcileOrphanedMediaFiles`) suppose le source sur disque → adapter.
5. **ffmpeg chemin critique** (plus best-effort) → check démarrage.
6. **Pic disque temporaire** (source + HLS avant suppression) ; `maxUploadSize` reste 500 Mo.
7. **Reprise** des `processing` interrompus par un crash → relançables via CLI / rescan boot.

## 9. Conformité architecture (plan-review)

- Pur → `media/hls.go` · orchestration → `service` + worker · handler sans logique métier · pas de DuckDB dans le handler.
- Multi-titres : arbre HLS sous `capturesDir` résolu par `PathResolver` ; capability `CapMedia` déjà gardée.
- Logging `slog.*Context`. Pas de `fmt.Println`.
- `thought_log.md` : entrée obligatoire avant chaque commit.

## 10. Prérequis & exploitation

### ffmpeg / ffprobe
Requis dans le PATH (transcoding HLS + miniatures). Vérifiés par `RunHealthcheck`
(checks `ffmpeg`/`ffprobe`). Leur absence n'empêche pas le serveur de démarrer
mais désactive le transcoding (les MKV restent servis en remux WebM live). Build
minimal suffisant (muxer `hls` + fMP4) ; le réencodage fallback (VP9 → H.264,
audio exotique → AAC) exige libx264/aac.

### Backfill de l'historique
`cmd/backfill-media-hls` convertit les vidéos existantes (MKV/AVI ou multipistes)
sans `hls_path` et reprend les `processing` orphelins. À lancer **serveur arrêté** :

    go run ./cmd/backfill-media-hls --db <shared_social.duckdb> --captures-base <dir> [--slug X] [--limit N] [--dry-run]

`--dry-run` liste les clips qui seraient transcodés sans rien écrire.

### Validation navigateur (POC, go/no-go Opus = GO)
Chrome lit l'Opus-in-fMP4 via hls.js **sans réencodage** : `currentTime` complet,
2 pistes Game/Mic listées (via `AUDIO_TRACKS_UPDATED`), bascule fonctionnelle, et
`MediaSource.isTypeSupported('audio/mp4; codecs="opus"') === true`. Safari/iOS hors
périmètre (ne lisent pas l'Opus en HLS).
