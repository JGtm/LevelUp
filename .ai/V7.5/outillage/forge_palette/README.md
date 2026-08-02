# Outillage de recherche — palette Forge, formes de zone, temoin de zone

ARCHIVE. Ce code n'est ni compile ni importe par l'application : `apps/go-api/cmd/tmp_*/`
est gitignore par convention du depot (`.gitignore:294`), et cette copie existe pour que la
session du 2026-08-01/02 soit rejouable. Precedent : le plan maitre §5.2-6 demande de
versionner sous `.ai/V7.5/` le Python jetable de la belle carte, comme archive de recherche.

## Restaurer

    cp -r .ai/V7.5/outillage/forge_palette/tmp_* apps/go-api/cmd/
    cd apps/go-api && CGO_ENABLED=1 go build ./cmd/tmp_forgename ./cmd/tmp_forgeshape \
        ./cmd/tmp_forgedraw ./cmd/tmp_zonetest

CGO est requis (decompression Kraken via `internal/ooz`) : `PATH` doit contenir
`C:\msys64\ucrt64\bin`.

## Ce que chaque outil fait

| outil | commandes |
|---|---|
| `tmp_forgename` | `hdr` `list` `dump` `ascii` `entry` `survey` `where` (sondes module/tag) · `hash` (murmur3) · `crack` (recherche exhaustive de nom) · `control` (rejoue les 45 types nommes) · `classify` `groups` (type_id -> groupe de tag) · `slots` (le StringID de nom de chaque entree) · `raw` `blk` `scanu64` `refsof` |
| `tmp_forgeshape` | `fields` `inventory` (champs du record .mvar) · `shapes` (extraction des formes) · `coverage` (couverture par carte) · `props` (champs jetes) · `upz` (orientation sol/mur) · `types` `zones` `obj` |
| `tmp_forgedraw` | rendu PNG des zones + mesure orientee/alignee/temoin ; mode `map` pour marquer des type_id |
| `tmp_zonetest` | LE TEMOIN de Q2 : confronte l'artefact de rejeu aux zones du .mvar aux instants du releve terrain |

## Entrees externes

- palette Forge : `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/{any,ds}` ;
- les 199 `.mvar` : `.claude/worktrees/filmdec-continuation/.ai/re_dump/mapvar/` (non
  versionnes, copie sur la cle `E:/LevelUp_rejeu2D/captures_cheat_engine/mapvar/`).
