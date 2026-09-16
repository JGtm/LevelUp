# BROUILLON — entrée de chronique v61 (lot 2.6.3)

> Brouillon PRÊT À COLLER, produit par le volet documentaire du lot 2.6 (M2, pas 6), préparé en
> parallèle du code (décision V15). Rien ici n'est encore dans `document_chronicle.go` : ce fichier
> n'est pas la chronique, il en porte le texte en attente du commit 2.6.3.
>
> Contrat que ce brouillon sert (ADR 0034, « Corrections », point 2) : **une entrée de chronique
> s'écrit dans le commit qui monte la version, jamais après** — le trou de v51 était resté
> invisible quatre jours. Le brouillon existe pour que l'exécuteur de 2.6.3 n'ait pas à rédiger
> sous la pression du commit, pas pour différer l'écriture.
>
> Base : `1e246b209`. `SchemaVersion` y vaut **60** (le plan écrit 57, chiffre périmé du
> 2026-09-13 : à corriger dans le commit 2.6.3). Date : 2026-09-17.

---

## 1. L'entrée, à coller en fin de `document_chronicle.go`

Style : celui des entrées v58 à v60 — titre en capitales, colonne de gauche en étiquettes,
français **sans accents** (convention du fichier), apostrophes conservées, aucun emoji.

```go
// v61 (2026-09-17, lot 2.6 du PLAN_DECODEUR_FILM) : L'ARTEFACT DIT SOUS QUELLES REVISIONS IL A
// ETE CUIT.
//
//	ce qui etait   la revision du decodeur ne vivait que dans le depot : le golden de
//	hors du        `grammar.Rev` et celui de `facts.Rev` disent ce que la TETE decode, jamais ce
//	document       qu'un artefact DEJA CUIT porte. Pour savoir sous quelle grammaire un document
//	               avait ete produit, il fallait dater sa cuisson et remonter au commit — et la
//	               version de schema ne repond pas a cette question : plusieurs revisions de
//	               grammaire tiennent sous une meme version de schema.
//
//	le bloc        `coverage.decoder` est NEUF : `grammarRev` (la grammaire de lecture),
//	ajoute         `factsRev` (la couche des faits, celle qui commande le backlog killsource) et
//	               `build` (la cle du profil, lue en clair dans `chunk_00` section 2, D-3 de
//	               l'ADR 0034). Un pointeur en `omitempty`, pour la meme raison que
//	               `FilmMajorVersion` : l'ABSENCE du bloc dit « artefact anterieur a ce lot »,
//	               et c'est une reponse, pas un trou.
//
//	build inconnu  `build` vaut la CHAINE VIDE et le bloc reste PRESENT quand le film porte un
//	               build absent du profil (`ErrUnknownBuild`). Sans cette regle, l'absence du
//	               bloc porterait deux sens — « cuit avant le lot » et « build inconnu » — et
//	               c'est exactement l'ambiguite « entre deux versions de schema » que D-7
//	               interdit.
//
//	AUCUNE AUTRE   TELEMETRIE PURE : aucun rendu n'en depend, aucune decision de decodage ne
//	DIFFERENCE     change, et `facts.Rev` naissante REPREND la valeur de `KillSourceDecoderRev`
//	               — donc aucun match ne devient candidat au backlog (D6). L'equivalence est a
//	               zero difference hors le seul champ `coverage.decoder`.
//
//	POURQUOI LA    un bloc apparait dans le document, donc la FORME change (garde-rail
//	VERSION MONTE  `document_shape_test.go`, qui refuse la regeneration sans montee). Un artefact
//	               60 ne peut pas dire sous quelle grammaire il a ete cuit : il peut seulement ne
//	               rien en dire. Les artefacts deja cuits restent servis tels quels, leur bloc
//	               `decoder` absent jusqu'a leur prochaine cuisson — aucune recuisson requise.
```

## 2. Les huit fichiers que la montée de schéma touche

Liste reprise de la note `.ai/PREPARATION_M2_PAS_4_A_6_2026-09-17.md` §3.3, dans l'ordre
d'exécution qu'elle prescrit. Les chemins sont relatifs à `apps/go-api/`.

| # | Fichier | Ce qu'on y fait |
|---|---|---|
| 1 | `internal/games/halo_infinite/film/replay/coverage.go` | `type DecoderCoverage struct { GrammarRev, FactsRev, Build string }` (tags JSON `grammarRev`, `factsRev`, `build`) + le champ `Decoder *DecoderCoverage` en tag `decoder,omitempty` dans `Coverage` |
| 2 | `internal/domain/replaydoc/coverage.go` | le jumeau servi, même forme |
| 3 | `internal/service/replayview/convert_coverage.go` | la conversion producteur -> servi (seule traduction) |
| 4 | `internal/games/halo_infinite/film/replay/document.go` (l. 48) | `SchemaVersion = 61` |
| 5 | `internal/games/halo_infinite/film/replay/document_chronicle.go` | l'entrée v61 du §1 ci-dessus, **dans ce commit** |
| 6 | `internal/games/halo_infinite/film/replay/testdata/document_shape.golden` | régénéré par sa porte unique |
| 7 | `api/openapi.yaml` | régénéré **en dernier** |
| 8 | `internal/service/replayview/parity_test.go` | la parité champ par champ du nouveau bloc (D-7) |

## 3. Les portes de régénération, dans l'ordre

1. **Empreinte de forme** (fichier 6) : `document_shape_test.go` a une porte unique et bruyante —
   `TestDocumentShapeRegenerate` exige `-update` **et** une variable d'environnement, et fait
   `t.Fatalf` même quand elle réussit. C'est voulu : « sans ce refus, la réponse naturelle à un
   golden rouge serait de le régénérer ». Deux gardes suivent la régénération :
   `...GoldenCarriesCurrentSchema` (le golden porte 61) et `...SchemaHasChronicleEntry` (61 a son
   entrée — d'où l'ordre fichier 5 avant fichier 6).
2. **Contrat HTTP** (fichier 7), en dernier : `make openapi-gen` puis `make generate-types`, gate
   `go test ./internal/api/ -run TestOpenAPIYAMLIsUpToDate -count=1` (CGO requis).
3. **Parité des jumeaux** (fichiers 2, 3, 8) : `service/replayview/parity_test.go` et
   `TestDocumentShape...TwinsAgree` — les deux formes doivent coïncider.

## 4. Ce que ce commit doit prouver, et où il se place

- **Dernier commit de M2.** D4 exige zéro différence d'équivalence à chaque pas structurel ; ce
  commit-ci en produit une par construction, et la preuve attendue est « différence limitée au
  champ `coverage.decoder` ». Le placer ailleurs mélangerait cette différence voulue avec celles,
  non voulues, des pas suivants.
- **Aucune recuisson.** `facts.Rev` naissante reprend la valeur de `KillSourceDecoderRev`
  (`killsource-2026-09-16.2` sur la base) : aucun match ne devient candidat au backlog, et le
  jalon se clôt sans backlog ouvert (arbitrage V15, question 16 de la note).
- **Le plan se corrige dans le même commit** : item 2.6.3, « montée de `SchemaVersion` 57 » ->
  **60 -> 61**.

## 5. Ligne prête pour le §5 du plan (journal des gates)

À coller telle quelle par le pilote si le volet documentaire doit figurer au journal — ce
brouillon ne touche pas `.ai/PLAN_DECODEUR_FILM_2026-09-13.md` (frontière du lot 2.6-docs).

```
### Lot 2.6 (M2, pas 6) — volet DOCUMENTAIRE, SANS AUCUN DECODAGE, 2026-09-17

Branche `feat/decfilm-26docs`, base `1e246b209`. Sous-section « Revisions du decodeur de film et
backlog killsource » ajoutee aux DEUX `docs/SYNC_GUIDE` (EN l. 141-160, FR l. 141-160, fichiers
alignes ligne pour ligne, 171 -> 192 lignes chacun) ; ADR 0034 corrige sur deux points factuels
(allowlist D9 a CINQ entrees et non une ; `weaponv3` vit sous `internal/analysis/`, pas sous
`games/halo_infinite/`) ; entree de chronique v61 en brouillon
(`.ai/BROUILLON_CHRONIQUE_V61_2026-09-17.md`), a coller au commit 2.6.3 avec la montee
`SchemaVersion` 60 -> 61. Aucune case de 2.6 cochee : 2.6.1 se coche au code.
```
