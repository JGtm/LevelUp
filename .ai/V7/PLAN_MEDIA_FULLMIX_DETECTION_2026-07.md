# PLAN — Détection full-mix robuste à la voix + outillage collapse/analyze (2026-07)

Branche : `fix/media-fullmix-detection` (worktree `media-fullmix-detection`, base = main local).
Aucun commit dans ce chantier (livraison = revue).

## Contexte

`analyzeAudioLayout` (`internal/media/hls_audio_analyze.go`) décide si la piste 0 d'une
capture OBS est le mix complet de sortie via la corrélation de Pearson des enveloppes dB
(piste 0 vs amix des composantes), seuil `fullMixEnvelopeCorrThreshold=0.80`. Ce critère
ÉCHOUE dès qu'il y a de la voix significative : les gains du mix OBS diffèrent des gains 1:1
de l'amix et la voix fait diverger les enveloppes dB. Mesuré : 0.519 (`2026-07-03 21-38-44`)
et 0.767 (`2026-07-07 23-03-38`) → sous le seuil → repli mapping historique → renditions
game/voices redondantes → toggles Jeu/Voix inaudibles. Les clips sans voix (07-16) passent
(corr ~0.98). `classifyGameComponent` (silence ratio) fonctionne — NE PAS TOUCHER.

## Item 1 — Détection full-mix robuste à la voix

Remplacer le critère de corrélation dB par une régression en domaine PUISSANCE.

- [x] 1.1 Maths pures (nouveau fichier `internal/media/audio_fullmix_fit.go`, séparé de l'IO) :
  - `dbToPower` (dB → puissance linéaire `10^(dB/10)`).
  - `nnls` : moindres carrés à gains ≥ 0 (Lawson-Hanson sur équations normales AtA/Atb).
  - `decideFullMix(env0DB, compsDB)` : ajuste `P0 ≈ Σ gᵢ·Pᵢ` (composantes non corrélées →
    puissance additive par trame), calcule R² (variance expliquée), la part de puissance de
    chaque composante, et la garde de stationnarité `stdDev(env0DB)`. Retourne une décision
    riche (R², gains, parts, composantes actives).
  - Discriminant anti-faux-positif DISJOINT : full-mix seulement si CHAQUE composante NON
    silencieuse a une contribution matériellement > 0 (couverture). Une composante muette
    (percentile 90 proche du plancher −91 dB, ex. micro coupé) n'impose rien.
- [x] 1.2 IO ffmpeg : `analyzeAudioLayout(ctx, src, audioStreams)` décode env0 + chaque
  composante (dB) via `audioEnvelope` (inchangé), appelle `decideFullMix` (pur). Composante
  jeu via `gameComponentIndex` (titre de piste OBS « game »/« jeu » prioritaire, repli
  `classifyGameComponent` INCHANGÉ). Fonction exportée `AnalyzeAudioLayoutReport` pour
  `--analyze`. Log `BuildHLS` : `fullmix_r2` au lieu de `envelope_corr`. `restMixFilter`
  (amix d'analyse) supprimé + tests associés.
- [x] 1.3 Tests purs (`decideFullMix`, `nnls`, `dbToPower`, `TestNNLS_*`) : mix gains
  inégaux → true ; disjoint (track0 = duplicata composante) → false via couverture ;
  composante muette ignorée ; stationnaire → false.
- [x] 1.4 Tests ffmpeg-gated : synthétique full-mix pondéré (1.0/0.35) → true ; DISJOINT
  2 pistes → false ; réels via `LEVELUP_MEDIA_TEST_DIR` : 07-03 + 07-07 (voix) → true,
  07-16 (sans voix) → true, jeu = 0:a:1 sur les trois.
- [x] 1.5 Seuils calibrés + justifiés (audio_fullmix_fit.go) : R² ≥ 0,80 (réels 0,946/
  0,927/0,972) ; multiSourceMinShare 0,05 (2ᵉ part réelle 0,17/0,13) ; coverageMinPowerCorr
  0,30 ; silentComponentCeilingDB −70 dB (actif −7..−30, muet −91).

Gate 1 : `go vet ./internal/media/...` + `go test ./internal/media/...` (ffmpeg + vrais
clips réellement exécutés) → VERTS (5,1 s).

## Item 2 — `cmd/migrate-hls-audio` : modes `--collapse-dir` et `--analyze`

- [x] 2.1 `--collapse-dir <chemin>` : collapse FORCÉ d'UN arbre HLS (dossier avec
  `master.m3u8`), SANS le critère de corrélation → `ForceCollapseHLSAudioTree`
  (hls_audio_collapse.go) réutilisant `collapseMasterToSingleAudio` + `writeMasterAtomic`.
  Aide explicite sur le caractère forcé + cas d'usage. Dry-run respecté (vérifié CLI :
  master intact en dry-run, réduit à `game`/DEFAULT=YES en réel). Test unitaire
  `TestForceCollapseHLSAudioTree` (dry / réel / sans game).
- [x] 2.2 `--analyze <fichier source>` : inspection à sec → `AnalyzeAudioLayoutReport` +
  `printAnalyzeReport` (décision full-mix, composante jeu, R², gains, parts, corr puissance,
  p90, actif, silence par composante). Outil de validation ET diag durable.
- [x] 2.3 Modes exclusifs (`--analyze`/`--collapse-dir` court-circuitent, `--root` non
  requis ; garde d'exclusivité analyze↔collapse-dir). Aide du package + flags à jour.

Gate 2 : `go build ./cmd/migrate-hls-audio` + `go test ./internal/media/...` verts.

## Gates finaux (depuis `<worktree>\apps\go-api`, `CGO_ENABLED=1`, msys64 ucrt64 dans PATH)

1. [x] `go vet ./...` → propre (aucune sortie).
2. [x] `go test ./internal/media/...` (ffmpeg + vrais clips réellement exécutés) → ok, 5,2 s.
3. [x] `golangci-lint run --timeout 5m --new-from-merge-base=main` → 0 issue.
4. [x] Validation empirique `--analyze` sur les 3 vrais clips :
   - `2026-07-03 21-38-44` (4 pistes, voix) : full-mix=true, jeu=0:a:1, R²=0,9462,
     parts=[0,166 / 0,001 / 0,715], pcorr=[0,178 / −0,079 / 0,947]. Multi-source (2ᵉ part
     0,166) → accepté ; la composante 0:a:2 (micro brut décorrélé, part≈0) tolérée.
   - `2026-07-07 23-03-38` (3 pistes titrées, voix) : full-mix=true, jeu=0:a:1, R²=0,9268,
     parts=[0,812 / 0,132], pcorr=[0,932 / 0,430]. Multi-source (2ᵉ part 0,132).
   - `2026-07-16 16-07-19` (sans voix) : full-mix=true, jeu=0:a:1, R²=0,9725,
     parts=[0,995 / 0], pcorr=[0,986 / 0], 0:a:2 muette (p90=−91, inactive) → couverture OK.
   L'ANCIEN code (corr d'enveloppe dB, seuil 0,80) échouait sur 07-03 (0,519) et 07-07 (0,767).

## Découvertes (hors périmètre — noter, ne pas traiter)

- `classifyGameComponent` (silence ratio, NON touché) se trompe sur deux cas réels : une
  piste voix Discord plus continue que le jeu (07-07) et une piste muette au silence nul
  (07-16, cas solo) → classe la mauvaise piste. Contourné en amont par la préférence de
  TITRE (`gameComponentIndex`), non corrigé dans le classifieur (hors périmètre + consigne).

## Journal

- [2026-07-17] Plan créé. Sources réelles confirmées présentes
  (`C:\Users\Guillaume\Videos\Captures\JGtm\`) : `2026-07-03 21-38-44.mkv` (4 pistes opus,
  Track1..4), `2026-07-07 23-03-38.mkv` (3 pistes aac full/game/voices),
  `2026-07-16 16-07-19.mkv` (3 pistes aac). ffmpeg 8.0.1 présent.
- [2026-07-17] Items 1 et 2 COMPLÉTÉS, tous items statués `[x]`, 4 gates verts. Algorithme :
  régression puissance NNLS (Lawson-Hanson) `P0 ≈ Σ gᵢ·Pᵢ`, décision sur R² + garde de
  stationnarité + couverture à deux régimes (mix ≥2 sources OU mono-source + corr puissance
  de chaque composante active) — écarte le faux positif disjoint sans rejeter une piste
  auxiliaire non routée dans le mix. Composante jeu par titre de piste (repli classifieur).
  Fichiers : `audio_fullmix_fit.go` (neuf, maths pures), `hls_audio_analyze.go`,
  `hls_audio_collapse.go` (ForceCollapse), `hls.go`, `cmd/migrate-hls-audio/main.go`
  (modes `--analyze`/`--collapse-dir`). Aucun commit (livraison = revue).
