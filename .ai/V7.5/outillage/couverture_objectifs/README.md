# Instrument de couverture des calques d'objectif (audit 2026-09-10)

Confronte les **artefacts de rejeu deja cuits** a l'**oracle officiel de l'API Halo**
exporte en TSV. N'ouvre **aucune base**, ne recuit **aucun film**, n'ecrit rien dans le parc.

## Rejouer les chiffres de l'audit

```bash
cd .ai/V7.5/outillage/couverture_objectifs
CGO_ENABLED=0 go run . \
  -parc   <racine>/data/cache/replays/halo_infinite \
  -oracle <racine>/.ai/V7.5/replay2d/registre_film \
  -out    <racine>/.ai/V7.5/replay2d/registre_film
```

Les huit sorties committees dans `replay2d/registre_film/` (`vague6_*.tsv|log`) sont la
sortie brute de cette commande, sur le parc du 2026-09-10 (64 artefacts, schema 51).

## Ce que chaque sortie porte

| fichier | contenu |
|---|---|
| `vague6_calibrage.log` | les deux temoins de calibrage (cf. plus bas) |
| `vague6_couverture_parc.tsv` | recensement : quel calque est publie sur quel film |
| `vague6_stats_publiees.tsv` | compte de chaque `stat` du calque `objectives`, par film |
| `vague6_couverture_portage.tsv` | axe 1 — temps de portage du drapeau publie vs oracle, par joueur |
| `vague6_couverture_zones.tsv` | axe 1 bis — zones : possession publiee vs `time_in_zones_seconds` |
| `vague6_couverture_actions.tsv` | axe 2 — actions publiees vs oracle, par joueur |
| `vague6_identite.tsv` | axe 3 — manches, `noSlot`, `noBridge`, porteurs non nommes |
| `vague6_bornage.tsv` | axe 4 — ecart par periode et simulation d'une demi-fenetre de tic |
| `vague6_objet_sans_position.tsv` | axe 5 — images de portage ou le porteur n'a pas de position |

## Calibrage — les deux moities, et celle qui n'est pas reproductible ici

1. **Lecture de l'artefact** : le compte de spans `carried` recalcule depuis `flagCarries`
   doit egaler `coverage.flagCarries.carries`. Resultat : **11 films sur 11**.
2. **Lecture de l'oracle** : les valeurs Oddball du rapport
   `RAPPORT_ODDBALL_FANTOMES_2026-09-10.md` §3.3 doivent ressortir du TSV. Resultat :
   **1 249,0 s** au total sur les quatre films, et les trois porteurs cites a la decimale
   (51,1 / 62,3 / 40,8 s).
3. **Ce qui n'est PAS reproductible ici** : les 82,2 % de l'oracle du meme rapport. Ils
   viennent d'une CUISSON hors ligne des quatre films Oddball ; **aucun de ces films n'a
   d'artefact au parc** (colonne `skullCarries` a zero sur les 64 du recensement). Les
   reproduire exigerait `replay-build`, hors perimetre d'un audit.

## Conventions de mesure

- Un **span** de `flagCarries` couvre les images `[t0, t1]` bornes incluses : sa duree vaut
  `(t1 - t0 + 1) x frameIntervalMs`.
- Une **periode de portage** est une suite maximale de spans `carried`/`carried_open`
  contigus du meme xuid.
- La **position d'un porteur a une image** reproduit `posOfPlayerAt`
  (`apps/web/src/features/match-replay/model/livesPosition.ts`) : une vie qui couvre
  l'image, sinon la derniere position d'une vie close depuis moins de 15 images
  (`KILLPOS_WINDOW_MS` = 1 500 ms au pas de 100 ms), sinon rien. Le repli vehicule de
  `carrierPosition.ts` est compte a part.
