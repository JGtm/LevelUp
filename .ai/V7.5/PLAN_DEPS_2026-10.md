# PLAN - Dépendances d'octobre 2026 (branche chore/deps-2026-10)

Base : origin/feat/v75 a9a9ac78a. Pas de fusion, pas de touche à main/feat/v75, PR Dependabot non fermées.

## Items
- [x] 1. Go toolchain 1.26.5 -> 1.26.9 (go.mod + commentaire) ; Dockerfile (golang:1.26-bookworm) et workflows (go-version-file) vérifiés cohérents
- [x] 2. Modules Go (rejeu PR #88) : x/crypto 0.57.0, x/sync 0.23.0, x/sys 0.48.0, x/time 0.16.0, modernc.org/sqlite 1.59.0
- [x] 2b. Ajout décidé par le superviseur : golang.org/x/image 0.24.0 -> 0.45.0 (GO-2026-6222 atteinte)
- [x] 3. npm lockfile seul : browserslist 4.29.3, baseline-browser-mapping 2.11.28, brace-expansion 2.1.7 (imbriqué), undici 8.12.1
- [x] 4. Gate complet (voir journal)
- [x] 5. Clôture : thought_log, commits deps-2610:, push, CI, disk-hygiene

## Journal
- govulncheck AVANT (go 1.26.5) : 19 vulnérabilités atteintes : 16 stdlib (net/http, net/textproto, crypto/tls, net/url, encoding/xml ; corrigées en 1.26.6/1.26.9) + 3 golang.org/x/image@v0.24.0 (GO-2026-6222, 5061, 4961, via mapdecoupe.ChargeMasque).
- govulncheck APRÈS toolchain + modules : 3 (x/image seul).
- govulncheck APRÈS x/image v0.45.0 : 0 atteinte ("No vulnerabilities found") ; 1 vulnérabilité dans un module requis, non appelée.
- go.mod comparé à la branche Dependabot : identique sauf nativewebp et x/image (absents de sa base antérieure) et la directive go. Transitifs montés : x/mod, x/telemetry, x/text, x/tools, modernc libc/memory.
- npm : package.json inchangé ; sorties de npm update acceptées (undici 8.12.1 >= 8.11.2).
- Gate Go rejoué après le dernier commit Go : go vet 0 ; go test ./... 201 paquets ok ; integration -p 1 (openspartan, mapper, handlers, service) ok ; paquets x/image (mapfond-webp, mapdecoupe) ok ; golangci-lint 0 issues.
- Gate web (après le commit npm, inchangé depuis) : tsc -b 0 (cache .tmp purgé) ; vitest 919 fichiers / 9598 tests ok ; knip-ratchet 0/0/0.
- Aucune montée retirée.

## Découvertes (non traitées)
- scripts/git-hooks/lefthook/govulncheck.sh : commentaires « Go 1.26.1 » périmés.
