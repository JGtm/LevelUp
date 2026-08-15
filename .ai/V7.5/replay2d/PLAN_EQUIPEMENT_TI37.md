# Plan — L'equipement au rejeu 2D : le nommer, le dater, le poser

> Ecrit le 2026-08-15. Il REMPLACE `PLAN_EQUIPEMENT_ACTIF.md`, dont les etapes 3 et 4 sont
> bloquees sur un canal (`i57`) que la mesure a refuse. Le present plan part d'un canal que
> personne n'avait ouvert : l'archetype `ti=37` « equipment ».
> Branche `feat/v75`. Execution sous le contrat du skill `plan-execution` (ordre strict, un
> item = un statut, zero fix hors perimetre, zero report d'une etape executable).

## Objectif et critere de succes

Montrer dans le rejeu 2D, pour chaque joueur : **QUEL** equipement il porte, **QUAND** il
l'active, **OU** l'objet pose se trouve — et le faire sonner.

Critere de succes, et il est negatif autant que positif : aucune de ces trois quantites
n'est affichee sans avoir ete MESUREE. Une quantite non etablie s'affiche comme ABSENTE,
jamais devinee. C'est la regle qui a fait echouer le lot du 14/08 et elle avait raison.

## Vision globale — les cinq canaux, et ce que chacun peut porter

| canal | archetype | ce qu'il porte | etat au 2026-08-15 |
|---|---|---|---|
| `i48` biped-desired-ability-set | 35 biped | IDENTITE : rang de palette COMPLET | **PUBLIE** (`filmdec/ability_rank.go`) — 748 lectures, 0 illisible, verite terrain 8/8 |
| champ d'ancrage image-cle | 35 keyframe | identite, **fenetre 16..23 seulement** | publie (`replay/inventory_decode.go`), borgne par construction |
| `i56` biped-spartan-ability-energy | 35 biped | ENERGIE : masque R(3) + 7 bits par charge, `0x7F` = plein | porte, valeur JETEE ; mesure existante sur UN film (176 lectures), jugee trop maigre |
| `i57` biped-spartan-ability | 35 biped | 4 etats, dont UN SEUL paie 24 bits de charge utile | porte ; mesure : 1 414 lectures, association a `i54` de 2,1x les temoins — **ne tranche pas** |
| **`ti=37` `equipment-*`** | **37 equipment** | **active / deploye / createur / energie / POSITION** | **porte, valeur JETEE, JAMAIS MESURE** |

Le dernier est le seul qui nomme la chose en clair, et le seul qui porte une POSITION.

## Ce qui est ACQUIS — ne pas re-chercher

| acquis | ou | consequence |
|---|---|---|
| La palette des capacites (rangs 0..23, 12 noms) | RECETTE_LOADOUT §13 | les 7 equipements sont identifies |
| La palette varie PAR FILM et se deduit de la signature des rangs | RECETTE_LOADOUT §14-15 | ne jamais presumer la famille A |
| L'identite est PUBLIEE et nommee (8/8) | `ability_rank.go`, lot du 14/08 | le « QUEL » est FAIT |
| Les 7 desers `ti=37` sont portes | `traverse.go:277-345`, `components_batch3.go:46` | il n'y a AUCUN decodage nouveau a ecrire pour la phase 0 |
| `EquipmentTypeIndex = 37` + `ScanFilmWorldObjects` | `filmdec/projectiles.go:85-120` | les positions sont un APPEL DE FONCTION |
| Le patron de publication d'une valeur jetee | `filmdec/ability_rank.go` | hook + marche + statistiques de denominateur |
| Les sons des 7 equipements (Activate / Deactivate / Recharge / Idle) | bibliotheque utilisateur, `Downloads/Audio Library.../EQUIPMENT` | la phase 4 n'attend aucune extraction |

## Ce qui est REFUTE — ne pas rejouer

- **Le bit 0 d'`i57` comme interrupteur** : 1 sur 386 lectures sur 386. Un interrupteur qui
  ne bascule jamais n'informe de rien.
- **La traversee d'`i56`/`i57` comme cause de rarete** : ZERO composant casse la marche sur
  1 535 records. La rarete est une FREQUENCE DE TRANSMISSION, pas un defaut de decodage.
- **Elargir le champ d'image-cle a 6 bits** (`large6`/`suite6`/`aval6`) : 0/8 contre la
  verite terrain la ou `prod3` rend 6/8.
- **Presumer la famille A de palette** : trois films n'exposent que les rangs 19..22.

## Phases

Chaque phase est livrable independamment et se clot sur un gate VERIFIABLE. Une phase
n'ouvre pas tant que la precedente n'est pas close.

### Phase 0 — MESURER — CLOSE le 2026-08-15

Corpus : **12 films**, un film par processus. Les 8 dont la palette `i48` etait deja mesuree
le 14/08 — `000d5950` (verite terrain Theater), `00162144` (palette hors famille A),
`00ba2e1c`, `06dfe6d9`, `084a804d`, `00502e52`, `07aa428d`, `0014603f` (temoin negatif :
`i48` jamais au masque) — plus **4 tires a pas regulier** des 951 du cache (`331ff98d`,
`64e8adfa`, `9edfcaa9`, `cfb85a58`), pour ne pas mesurer seulement les films qui marchaient
deja. Duree 11 a 256 s par film.

- [x] 0.1 Volume `ti=37`. **1 097 619 records delta, 19 260 slots, 6 904 vies d'objet.**
      L'archetype est verifie SUR PIECES (31 composants) : les index du plan sont exacts, et
      il en porte SIX de plus, jamais cites — i25 being-hacked, **i26 energy-delay-ticks-left**,
      **i27 charges-remaining**, i28 tracked-object-handles-stack, i29 command-tick,
      i30 has-infinite-uses.
- [x] 0.2 Les quatre champs publies par hook sur le deser de PRODUCTION
      (`filmdec/equipment_state.go`, `SetEquipmentStateHook`) : **13 187 records annoncent au
      moins un des quatre (1,20 %), marche aboutie 13 099, cassee 88.** Par champ
      (masque / lu / porte fermee) : deployed 3839/3820/0 · activated 2126/2121/894 ·
      creator 2209/2194/866 · energy 5268/5217/0. **Transitions : 81 sur 208 paires pour
      `activated`, 1 601 sur 3 960 tous champs confondus.**
- [x] 0.3 `ScanFilmWorldObjects(dir, wr, 37)` : **9 043 trajectoires, 628 368 echantillons.**
      Les bornes MONDE exigeraient une base ; le controle est donc joue en coordonnees
      NORMALISEES de l'AABB, contre le nuage des BIPEDES du meme film — plus severe et sans
      dependance : **610 693 sur 628 368 (97,2 %) dedans, 12 films sur 12 (92,3 a 99,7 %).**
- [x] 0.4 Controle non circulaire du `R(5)` de `creator`. **ECHOUE, et publie tel quel :
      0 valeur sur 1 328 ne tombe dans `bipedSlotBand`.** Le repli C2 (« au moins un index
      compact ») passe mais est VACUEUX — 5 bits plafonnent a 31 et tous les films ont plus
      de 31 slots ; dit tel quel. Le seul controle qui tranche vient de la verite terrain :
      sur `000d5950`, 8 joueurs, le champ prend 15 valeurs distinctes et monte a 28 ; sur le
      corpus il couvre ses 32 valeurs avec une queue plate. **Ni slot, ni index de joueur.**
- [x] 0.5 `i56` elargi (`filmdec/i56_drops_test.go`, garde `I56_DROPS_FILM`, lecture par le
      deser de production). **2 519 042 records delta biped, 2 088 lectures (0,083 %), 0
      illisible, 1 275 slots ; 519 paires -> 282 CHUTES, 26 REMONTEES.** Les deux encodages
      sont separes : **282 chutes sur 282 franchissent un multiple de 16, aucune ne reste
      dans un quartet.** Temoins +/-5 s : **la coincidence avec `i54` est REFUTEE par
      l'elargissement — reel 7/601 (1,2 %) contre 9 (1,5 %) et 5 (0,8 %)** ; le 4,5 % du
      14/08 etait 3 episodes sur 67 d'un seul film, et l'instrument le reproduit au chiffre
      pres sur ce film-la avant de s'effondrer sur les onze autres.
- [!] 0.6 Sept equipements ou deployables seuls ? **NON TRANCHE, et la mesure ne pouvait pas
      le trancher — c'est le resultat, pas un abandon.** Compte : 9 043 trajectoires sur
      12 matchs, bien plus que des equipements de joueur ; l'archetype s'appelle `item-*`
      autant qu'`equipment-*` (i18 `item-at-rest`, i19 `item-ignore-player`), donc bonus au
      sol et socles y sont vraisemblablement melanges. **Aucun des quatre champs ne porte
      d'identite** : rien ne distingue une entite d'une autre. Porte au
      `REGISTRE_REPORTS.md` avec sa condition de reprise (chercher la reference de definition
      dans le record de CREATION, comme la famille high-32 des armes au sol).

**Gate 0 : PASSE.** Les trois verdicts, ecrits et chiffres :

1. **DATER l'activation — NON.** `activated` est transmis 2 121 fois sur 1 097 619 records
   (0,19 %) ; 949 vies d'objet sur 6 904 en recoivent une valeur, dont **227 seulement apres
   le premier record de la vie** (une valeur au premier record date la REPLICATION, pas le
   geste). Il reste 208 paires consecutives et 81 transitions sur 12 matchs, et **aucune des
   huit valeurs de R(3) n'est nommee** — elles sont toutes peuplees, aucune ne domine.
   Verdict identique pour `i56` : 282 chutes pour 1 275 vies lues, une chute pour cinq vies.
2. **QUI active — NON.** Controle prescrit ECHOUE (0/1 328), et le champ n'est pas davantage
   un index de joueur (28 sur un film a 8 joueurs, 32 valeurs a queue plate).
3. **OU l'objet est pose — OUI.** 97,2 % de 628 368 echantillons dans l'emprise des joueurs
   du meme film, 12 films sur 12. C'est le seul acquis exploitable du lot.

Instruments versionnes et gardes (`EQUIP_FILM`, `I56_DROPS_FILM`), sautes en CI.
`go build`, `go vet`, `go test ./internal/analysis/filmdec/ ./internal/analysis/replay/`
verts, `golangci-lint --new-from-merge-base=origin/main` 0 issue. Entree
`[2026-08-15]` au `thought_log.md`. **Journal detaille : `PLAN_EQUIPEMENT_ACTIF.md`
etape 5** (ce lot prolonge ses etapes 3 et 4, d'ou le journal y reste).

**CE QUE LA PHASE 0 DESIGNE POUR LA SUITE, et qui n'etait dans aucun plan.** Le recensement
au masque des 31 composants de `ti=37`, cumule sur les 12 films, place les quatre champs
mesures LOIN derriere deux autres :

    equipment-charges-remaining-component        16 125   <- 3,1x l energie, 7,6x l active
    equipment-energy-delay-ticks-left-component  10 608   <- un compte a rebours en ticks
    equipment-energy-component                    5 268
    equipment-deployed-component                  3 839
    equipment-creator-component                   2 209
    equipment-activated-component                 2 126

Les deux sont **DEJA PORTES** par `consumeByName` : il ne manque qu'un hook, comme pour les
quatre d'aujourd'hui. Un compteur de charges qui decroit date une utilisation par
construction. Hors du perimetre FERME de la phase 0 — c'est le premier geste de la suite.

### Phase 1 — PUBLIER le canal gagnant (branche sur le verdict 0)

- [ ] 1.1 `filmdec/equipment_state.go` sur le patron d'`ability_rank.go` : hook sur le
      deser, marche des composants du masque avec les desers de PRODUCTION, enregistrement
      localise (slot, chunk, packet, `TimestampUS`), et un type `Stats` avec ses
      denominateurs (`Records` / `WithComponent` / `Read` / `Unread`).
- [ ] 1.2 Entree de DONNEES dans `replay.BuildOptions` (comme `AbilityRanks`), peuplee par
      `BuildFromFilm`, absence NON FATALE et loggee (`slog.WarnContext`).
- [ ] 1.3 Champ du `ReplayDocument` + contrat (`replayContract.test.ts`) + normalisation web.
- [ ] 1.4 Golden d'assemblage mis a jour (`golden_inputs_test.go`).

**Gate 1** : le document porte la donnee sur le corpus local, avec sa COUVERTURE publiee
(combien de vies ont un instant d'activation, combien n'en ont aucun). Une couverture
partielle est un resultat, pas un echec — elle s'ecrit.

Interdits de cette phase : aucun rendu, aucun son, aucune valeur par defaut inventee.

### Phase 2 — NOMMER (manifeste, jamais de litteral en dur)

- [ ] 2.1 Les noms d'equipement vivent dans `config/titles/halo_infinite/mappings/replay_labels.toml`,
      section `[[ability_palettes]]` / `[ability_palettes.ranks]` qui existe deja. Aucun
      libelle FR/EN en dur cote Go (regle transverse multi-titre).
- [ ] 2.2 Un rang non resolu s'affiche COMME RANG, jamais comme capacite.
- [ ] 2.3 Si un nom s'ajoute, il porte sa PROVENANCE en commentaire (releve terrain sur au
      moins deux porteurs, ou double chaine murmur3 + banque sonore — regle RECETTE §14).

**Gate 2** : `go test ./internal/games/...` vert ; aucun nom ajoute sans provenance ecrite.

### Phase 3 — MONTRER

- [ ] 3.1 L'effet PLEINE FICHE a l'activation (demande utilisateur du 14/08 : toute la
      fiche, pas un lisere). Duree de remanence derivee de l'instant mesure, pas choisie.
- [ ] 3.2 L'objet pose sur la CARTE, si la phase 0 rend des positions : meme chaine de
      projection que les trajectoires (`MondeVersPixel`, calage du fond de carte).
- [ ] 3.3 Tokens semantiques uniquement (skill `color-tokens`), `prefers-reduced-motion`
      respecte, strings en FR **et** EN dans `i18n.ts` avec parite par typage.

**Gate 3** : `npm run typecheck` apres purge de `node_modules/.tmp`, `npm run lint`,
`npm run test` — les trois a exit 0. Aucun hex ni classe Tailwind couleur dans
`features/`. **Gate visuel utilisateur** : la session ne juge pas son propre rendu.

### Phase 4 — FAIRE SONNER

- [ ] 4.1 Les 7 `Activate` de la bibliotheque, joints a l'instant mesure en phase 1.
- [ ] 4.2 Les 5 `Recharge`, joints a la remontee d'energie si la phase 0 l'etablit.
- [ ] 4.3 Meme regle que le lot du 15/08 : une source sans donnee reste MUETTE, jamais le
      son d'un equipement voisin. Aucun asset verse qui ne soit joue (garde-rail
      `replaySoundAssets.guard.test.ts`).

**Gate 4** : garde-rail vert, gates web verts, **gate d'ECOUTE utilisateur**.

## Regles dures (elles ont deja tranche ce chantier)

1. **Aucun effet, aucun son, aucun nom sans donnee mesuree.** Le lot du 14/08 a eu raison de
   ne rien afficher.
2. **Un rang non resolu s'affiche comme rang.**
3. **Offline-pur et universel** : pas de Cheat Engine, pas de capture runtime.
4. **Un seul decodage `filmdec` par process** (hooks globaux, `LockProcessDecode`).
5. **Aucune base DuckDB ouverte en ecriture** : le serveur de l'utilisateur tourne.
6. **Zero fix hors perimetre.** Toute decouverte va en section « Decouvertes », pas dans le
   diff.

## Statuts d'item et cloture

`[x]` fait · `[~]` couvert ailleurs (avec la reference) · `[!]` non traite (avec la
justification ecrite). **Aucune case vide a la cloture d'une phase.**

## Decouvertes — a consigner, NE PAS traiter dans ce chantier

- Type de grenade des 2 etiquettes AMBIGU (`31e8d17e`, `88f1034c`) : piste non prise —
  joindre un kill a la grenade au dernier LANCER du meme joueur (`doc.grenades` porte le
  type 0..3 et l'auteur). **Validable sans rien affirmer** : les 15 tags VALIDE forment un
  oracle ; mesurer le taux d'accord de la jointure sur eux AVANT de l'appliquer aux deux
  ambigus. A quantifier d'abord : quelle part des morts ces 2 tags representent (jamais
  compte).
- `i59 biped-spartan-ability-non-predicted-state`, branche `tag==3` : corps
  `FUN_142f25e90` = vecteur position + quaternions + dequantifications, NON PORTE. Candidat
  a la pose d'un mur ou a l'ancre d'un grappin si `ti=37` ne les couvre pas.
- `i51 biped-emp-timer` : un TIMER, jamais interroge.

## Protocole de reprise de session

1. Lire ce fichier de haut en bas : les cases cochees disent ou en est le chantier.
2. Lire l'entree la plus recente de `.ai/thought_log.md` portant « equipement ».
3. Lire `.ai/V7.5/REGISTRE_REPORTS.md` pour les reports en cours.
4. **Verifier sur pieces** avant de coder : rouvrir le fichier et la ligne cible, le code a
   pu bouger.
