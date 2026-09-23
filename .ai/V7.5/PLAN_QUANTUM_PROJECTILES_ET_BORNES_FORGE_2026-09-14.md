# PLAN — Repli du quantum des projectiles, et bornes des cartes Forge

> Cree le 2026-09-14, apres revue sur pieces du fork `ChaseWoodhams/LevelUp` (commits du
> 11 au 13/09) contre `feat/v75` au commit `3d83ca75e`.
>
> **Contrat d'execution : skill `plan-execution`.** Ordre strict, une etape a la fois, aucun
> report d'une etape executable maintenant, verification sur pieces avant de cocher, zero fix
> hors perimetre. Skill `delivery-checklist` avant tout commit.
>
> **Statuts** : `[x]` fait · `[~]` couvert ailleurs (avec reference) · `[!]` non traite (avec
> justification ecrite). Aucune case vide a la cloture.
>
> **Deux lots INDEPENDANTS** (aucun fichier commun) : ils peuvent etre menes en parallele,
> dans deux worktrees temporaires distincts. Chacun se clot entierement (CI verte au niveau
> job) avant sa fusion dans `feat/v75`.
>
> **Reprise de session** : le premier item non coche du lot en cours est le point de reprise.
> Ne jamais reprendre au milieu d'un lot dont le gate n'a pas ete rejoue.

---

## Decisions tranchees (ne pas rouvrir en cours d'execution)

- **D1** — Le depliage se fait dans `filmdec` (a la source), la garde de pas de
  `replay/projectiles.go` est CONSERVEE en filet contre un record mal decode. Les deux ne
  font pas double emploi : l'une repare une lecture juste, l'autre refuse une lecture fausse.
- **D2** — Seuil de detection du repli : la MOITIE de l'etendue de l'axe (regle de
  l'arithmetique modulaire, pas un reglage). Elle n'est acceptable que si le gate 1B.4 le
  confirme : apres depliage, aucun pas reconstitue ne depasse `projectileMaxStepM`.
- **D3** — **La re-cuisson du parc n'est PAS dans ce plan.** Le correctif ne change que les
  films decodes apres lui ; la re-cuisson est une decision utilisateur separee (bombe RAM,
  verrou `filmproc.AcquireSolo`). Elle entre au `REGISTRE_REPORTS` a la cloture du lot 1.
- **D4** — Le sous-lot 1C est une **MESURE**, pas une fonctionnalite. Il ne pose aucune
  capability, n'ecrit aucune table, ne touche pas au web. S'il reussit, il justifie un plan
  separe ; s'il echoue, il ferme la question par ecrit.
- **D5** — 1C ne re-tente PAS l'appariement tir vers degat (interdit par la remise du
  2026-09-01). Il travaille sur un autre canal : la fin certifiee d'un vol contre la position
  d'un bipede.
- **D6** — 1C ne concerne QUE les armes a projectile (entite repliquee ti=41). Le MA40 et les
  autres armes hitscan sont hors de sa portee par construction : le verrou V1 de la precision
  n'est pas adresse ici, et ne doit pas etre annonce comme tel.
- **D7** — Lot 2 : une carte Forge herite des bornes de son CANEVAS (doctrine deja etablie et
  rejouee par `TestPreuveLevelIDCartes`). Le lien nom vers canevas est celui de
  `himap.CartesForge`, jamais une seconde table.

## Hors perimetre, definitivement

Sa pile `chore/english-only` (suppression de `docs/FR/`), `cmd/study-archiver`, `apps/study`,
ses images de sol, son `mainBSP` heuristique (notre critere moteur `himap.BSPQuantification`
le remplace), son `team_designators.go` (notre `NOTE_EQUIPE_FILM_2026-09-12.md` va plus loin),
son `lives_start.go` (chantier decodeur en cours, autre session). Aucune branche du fork n'est
mergee : cherry-pick d'un commit isole, ou re-implementation.

---

# LOT 1 — Le repli du quantum sur les trajectoires de projectile

**Branche** : `feat/quantum-projectiles`, depuis `feat/v75`. Worktree temporaire dedie.
**Effort** : 1A+1B rapide (une demi-journee, gate compris) · 1C moyen (une journee de mesure).
**Source** : commit `a900c5ba3` du fork (`unwrapLife`, 40 lignes) — a RE-ECRIRE en francais,
aux chemins v75, jamais cherry-picke tel quel (il est pose sur `english-only`).

## 1A — Confirmer la cause sur NOTRE parc, avant d'ecrire une ligne

L'etat de l'art est deja etabli, il ne se re-instruit pas :

| Fait | Piece |
|---|---|
| 338 trajectoires coupees sur 20 750 pistes, 64 films sur 76 | `data/cache/replays/halo_infinite`, champ `coverage.projectiles.truncated` |
| Ces artefacts sont cuits le 12/09 a 22h | mtime — donc APRES notre correctif de la porte |
| Le correctif de la porte d'i0 est `fb71e9b3c` (12/09 03h04) | il avait fait tomber 8 091 pas impossibles a 7 sur `0797ce72` |
| Un record d'une AUTRE region est rejete, pas mal decode | `decodeWorldObjectPos` compare l'index de region a `WorldObjectPrecision.Region` |

Le residu n'est donc ni la porte, ni une region etrangere. Reste a verifier la signature du
repli modulaire.

- [ ] 1A.1 Ecrire `filmdec/projectiles_repli_measure_test.go` (garde d'environnement, saute
      en CI, jamais de cuisson d'artefact) : sur **6 films** du cache, pour chaque pas
      superieur a `projectileMaxStepM`, journaliser l'axe, l'amplitude du saut, l'etendue de
      cet axe, le rapport saut/etendue, et l'amplitude des DEUX autres axes.
- [ ] 1A.2 **Verdict ecrit** dans le test : le repli est confirme si, sur les pas impossibles,
      le rapport saut/etendue vaut 1,00 a moins de 1 % pres sur UN axe et que les deux autres
      restent sous 1 m. Si la signature ne sort pas, **arreter le lot ici** et consigner
      (le correctif deviendrait une heuristique sans cause).
- [ ] 1A.3 Mesurer la part des pas impossibles portes par un axe dont l'etendue est proche du
      seuil physique (moitie d'etendue sous 12 m) : c'est la population ou D2 est fragile.

**Gate 1A** : `go test ./internal/games/halo_infinite/film/filmdec/ -run Repli -v` avec la
garde d'environnement armee, verdict 1A.2 imprime et recopie dans le plan.

## 1B — Le correctif

- [ ] 1B.1 Ajouter `deplieVie(seg, wr)` dans `filmdec/projectiles.go`, appele a l'UNIQUE
      site de construction des pistes (`ScanWorldObjectsForBand`, la boucle `splitLives` de
      la ligne 213). Verifier sur pieces qu'il n'existe pas de second site avant d'ecrire.
- [ ] 1B.2 Documenter la cause en francais, avec les chiffres de 1A : pourquoi une position
      se replie (quantification sur la boite de la carte, seul `q mod 2^w` voyage), pourquoi
      le premier echantillon est pris tel quel, pourquoi plusieurs franchissements
      s'accumulent et un retour ramene le decalage a zero.
- [ ] 1B.3 **Corriger le diagnostic faux** de `replay/projectiles.go:66` (« apres un
      basculement de bit ») : ce n'est pas un bit qui bascule, c'est le quantum qui reboucle.
      Le commentaire de `projectileMaxStepM` dit desormais que la garde est un filet contre un
      record mal decode, le repli etant traite en amont.
- [ ] 1B.4 Test unitaire pur `projectiles_repli_test.go` : une vie synthetique qui franchit
      un bord, une qui le franchit deux fois, une qui revient (decalage ramene a zero), une
      vie d'un seul point, une vie sans bornes utiles. **Aucun pas reconstitue ne depasse
      `projectileMaxStepM`** — c'est la verification de D2.
- [ ] 1B.5 Re-jouer 1A.1 sur les memes 6 films apres correctif : le compte de pas impossibles
      doit tomber a zero ou s'expliquer un par un. Chiffre avant/apres consigne.
- [ ] 1B.6 Verifier que le golden du rejeu ne bouge PAS sur un film sans pas impossible, et
      qu'il bouge de maniere expliquee sur un film qui en a (`golden_assembly_test.go`).

**Gate 1B** : `go test ./internal/games/halo_infinite/film/...` vert · `go vet ./...` (PATH
msys64/ucrt64 en tete, CGO obligatoire pour himap) · `make go-api-lint` · commit + push +
**CI verte au niveau job**.

## 1C — Sous-lot : est-ce que ca debloque la precision par arme ?

**La question exacte** : le film permet-il de dire qu'un projectile a TOUCHE un adversaire,
sans passer par le flux de degats — qui est aveugle aux armes a projectile (verrou V2 de
`RECALAGE_WEAPON_ACCURACY_FILM_2026-09-01.md` : Ravager 16 tirs / 0 touche, SPNKr 16/0,
Mangler 15/0) ?

**Le principe** : `projectile-at-rest-state` (i18) certifie la fin d'un vol dans 78 cas sur 79.
Si cette fin tombe sur un bipede ennemi a cet instant, le projectile a touche. La verite
terrain existe pour les touches FATALES : killsource couvre les kills a 97,6 % avec l'arme, la
victime et l'instant.

- [ ] 1C.1 Constituer l'echantillon : **8 films au plus**, choisis pour contenir des kills
      d'armes a projectile (`weapon_kills` / `match_kill_events`, **vue `_latest` uniquement**,
      regle ART n°2). Lecture seule, sequentielle. Aucune cuisson d'artefact.
      **Acces** : la DB partagee peut etre tenue RW par le serveur local — `OpenReadForQuery`,
      jamais `OpenReadOnly` force, jamais d'ouverture nue hors provider. Chemins par
      `PathResolver` ; chunks de film par `filmcache.ChunkDir`, jamais un `filepath.Join`.
- [ ] 1C.2 Instrument `replay/touche_projectile_measure_test.go` : pour chaque kill d'arme a
      projectile, chercher la vie de projectile dont la fin certifiee est la plus proche de la
      victime dans une fenetre de plus ou moins 300 ms autour du kill, et journaliser la
      distance.
- [ ] 1C.3 **Les deux temoins, ecrits d'avance** (meme protocole que la preuve lancer vers
      naissance a 70/70) : instant permute (meme kill, position de la victime tiree a un autre
      instant) et bipede tire au hasard parmi les vivants. Sans ces deux temoins, la mesure ne
      vaut rien.
- [ ] 1C.4 **Critere de succes, fixe d'avance** : rappel d'au moins 70 % des kills d'arme a
      projectile apparies a moins de 2 m, ET rapport d'au moins 5x entre le rappel et le
      meilleur des deux temoins. En dessous : l'oracle n'existe pas.
- [ ] 1C.5 Mesurer le denominateur potentiel : combien de vies de projectile sont rattachables
      a un tireur par leur NAISSANCE (la preuve existante donne 0,77 unite pour les grenades),
      et combien portent une fin certifiee. Sans ces deux nombres, un taux de touche n'a pas
      de denominateur.
- [ ] 1C.6 **Verdict ecrit** dans `.ai/V7.5/film_re/`, et une seule des deux suites :
      succes vers un plan separe « precision des armes a projectile » (table, capability, UI) ;
      echec vers une entree au `REGISTRE_REPORTS` avec sa condition de reprise, la remise du
      2026-09-01 restant en l'etat.
- [ ] 1C.7 Rappeler dans le verdict, quel qu'il soit, que le verrou V1 (armes automatiques,
      un degat ne s'apparie qu'a un tir) n'est PAS adresse par ce sous-lot.

- [ ] 1C.8 Entree `.ai/thought_log.md` pour le lot 1 entier : date, titre, statut, decision
      technique (le repli est suivi a la source, la garde reste en filet), resultats mesures
      (1A.2, 1B.5, 1C.4), conclusion et prochaine etape. L'absence d'entree = lot non termine.

**Gate 1C** : l'instrument tourne sous garde d'environnement, saute en CI, et le verdict 1C.4
est imprime avec ses deux temoins. Aucun code de production modifie par ce sous-lot.

## Ce que le lot 1 NE touche pas

Aucun handler, aucun service, aucun repo, aucune migration, aucune capability, aucune string
i18n, aucun fichier de `apps/web`. Si l'execution amene a en ouvrir un, c'est que le perimetre
a derive : consigner en « Decouvertes » et s'arreter.

---

# LOT 2 — Les 24 cartes Forge sans bornes de quantification

**Branche** : `feat/mapquant-forge-registre`, depuis `feat/v75`. Worktree temporaire dedie.
**Effort** : moyen. **Pre-requis dur** : Halo Infinite installe (tag `gamefiles`).

## 2A — Le constat, verifie sur pieces

`himap.CartesForge` declare 87 cartes avec leur canevas. Apres normalisation des suffixes de
variante (`- ranked`, `heavies`), **24 d'entre elles n'ont aucune entree dans
`data/titles/halo_infinite/reference/map_quant_bounds.json`** :

Alpha Site, Ardent Prayer, Argyle, Boulevard, Canopy, Cold Storage, Courtyard, Detachment,
Diminished, Foundry, Ghost Town, Guardian, Immolate, Interference, Ivory Tower, Lone Wolf,
Megapolis, Powerhouse, Ruujaya, Security Zone, Serenity, Showdown Arena, Vacancy, Yuletide.

Consequence : `MapQuantCatalog.Lookup` n'a AUCUN repli (`filmdec/map_bounds.go:117`) — sur ces
cartes, `ErrUnknownMapBounds` interdit toute coordonnee monde : pas de rejeu 2D, pas de
distance de kill, pas de portee.

Cause : `cmd/mapquant-build` porte sa PROPRE table `mapModule` en dur, independante de
`himap.CartesForge`, et le catalogue n'a pas ete regenere depuis le 2026-08-27 alors que le
registre a grossi jusqu'au 03/09. Deux tables pour un seul fait, exactement l'anti-pattern
« factorisation abandonnee » de CLAUDE.md.

- [ ] 2A.1 Rejouer le calcul de la liste (registre normalise moins catalogue) et la figer dans
      le plan : c'est le perimetre FERME du lot, aucune carte ajoutee en cours de route.
- [ ] 2A.2 Pour chacune, relever son canevas declare. Verifie : tous sont deja au catalogue
      sous d'autres noms SAUF `fo10_deadland` (Ivory Tower), qui exige une vraie lecture de
      module.

## 2B — Une seule table

- [ ] 2B.1 Dans `cmd/mapquant-build`, construire la table effective = `mapModule` (cartes
      natives) UNION la projection de `himap.CartesForge` (Nom vers ModuleCanevas). Sur nom
      commun avec modules differents : **erreur, jamais un arbitrage silencieux**.
- [ ] 2B.2 Retirer de `mapModule` les entrees Forge qui font desormais doublon avec le
      registre (Vagabond, Corpo, Origin, Solitude, Empyrean, Lattice...) — 0 code mort laisse.
- [ ] 2B.3 Regenerer le catalogue avec le jeu installe. `fo10_deadland` est le seul module
      jamais lu : verifier qu'il donne bien la boite de 463 m au decoupage [15 15 17], comme
      les six autres canevas, et refuser l'entree sinon.
- [ ] 2B.4 Verifier que les 79 entrees existantes sont **inchangees au bit pres** (diff du
      JSON) : une regeneration qui bouge une borne existante est un signal, pas un detail.
- [ ] 2B.5 `slog` a la regeneration : une ligne par carte ajoutee (nom, module, etendue,
      largeurs), une ligne d'erreur par carte refusee avec sa raison.

## 2C — Le controle sur film, la ou il est possible

- [ ] 2C.1 Etablir la liste des cartes de 2A qui ont au moins un film dans
      `data/cache/film_chunks` (jointure parc de matchs et cache).
- [ ] 2C.2 Sur celles-la, confronter les largeurs DEDUITES des bornes au decoupage LU dans le
      film (`DetectI0Layout`) — c'est le controle qui avait refuse 14 cartes le 2026-08-16 et
      revele la bouillie des canevas Forge. Toute carte en desaccord sort du catalogue avec
      sa justification `[!]`.
- [ ] 2C.3 Pour les cartes sans film local, ecrire l'argument de transitivite : meme canevas
      donc meme boite donc memes largeurs, et ces largeurs sont deja validees sur film pour ce
      canevas. Citer le film temoin par canevas.

## 2D — Le garde-rail, dans le meme commit

- [ ] 2D.1 Test-cliquet (hors `gamefiles`, il ne lit que deux fichiers du depot) : toute carte
      de `himap.CartesForge`, apres `NormalizeMapName`, doit avoir une entree dans
      `map_quant_bounds.json`, ou figurer dans une allowlist DATEE portant sa raison (le
      refus 2C.2 est la seule raison admise).
- [ ] 2D.2 Verifier que le cliquet est ROUGE avant 2B et VERT apres — un cliquet qu'on n'a pas
      vu echouer ne prouve rien.
- [ ] 2D.3 Entree `.ai/thought_log.md` : date, titre, statut, decision (une seule table,
      derivation du registre), resultats (24 cartes entrees, N refusees par 2C.2), suite.

**Gate 2** : `go test ./internal/himap/ ./cmd/mapquant-build/...` vert ·
`go test -tags=gamefiles -run 'TestPreuveLevelIDCartes|TestBSPQuantificationTousModules'` vert
sur le poste avec le jeu · `make go-api-lint` · diff du JSON relu a l'oeil (2B.4) ·
commit + push + **CI verte au niveau job**.

---

## Fusion

Les deux lots vont dans `feat/v75` une fois leur CI verte, dans n'importe quel ordre (aucun
fichier commun). En cas de conflit, `feat/v75` a raison et le tactique se re-branche. Jamais
de merge vers `main` depuis ce plan (push main = deploiement prod).

## Decouvertes (a remplir en cours d'execution, NE PAS TRAITER)

- (vide)

## Reports (a verser au `.ai/V7.5/REGISTRE_REPORTS.md` a la cloture)

- Re-cuisson du parc pour que les 338 trajectoires coupees soient reparees dans les artefacts
  deja publies (decision utilisateur, D3).
