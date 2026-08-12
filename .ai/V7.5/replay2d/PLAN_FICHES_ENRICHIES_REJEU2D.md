# Plan — Fiches joueur enrichies du rejeu 2D (arme en main, vitals, capacite, grenades, drops)

> Branche cible : `feat/v75` (mode branche unique v7.5 ; JAMAIS de merge vers `main` avant le
> tag). Le rejeu reste OFF EN PROD : le garde `handlers/replay_local_gate.go` n'est PAS touche.
> Contrat d'execution : skill `plan-execution` (ordre strict, une etape a la fois, aucun item
> sans statut a la cloture, zero fix hors perimetre). Reprise : lire la section « Suivi » en bas.

## Objectif et critere de succes

Enrichir les fiches joueur de la page replay (`features/match-replay/ReplayTeams.tsx`) avec, par
joueur et a l'instant lu : la **sante**, l'**arme en main** (a gauche) / **secondaire** (a droite),
la **capacite d'armure equipee**, les **grenades portees** — et un calque **armes au sol (drops)**.

> **PERIMETRE FINAL (2026-08-12)** : le calque **armes au sol est HORS v7.5** — sa position s'est
> revelee indecodable offline-pur (mesure 2.1, temoin fantome). Seule la Phase 1 est livree cote
> produit ; la Phase 2 laisse un decodeur d'IDENTITE et son instrument de mesure, et un report au
> registre. Le reste de ce document conserve l'enonce d'origine, corrige la ou il etait faux.

**Critere de succes** : sur le match `000d5950` (Cliffhanger, seul artefact sur disque), gate
visuel utilisateur PASSE ; chaque donnee affichee est soit une mesure du film soit une LACUNE
explicite (jamais un zero ni une valeur par defaut) ; parite FR+EN ; zero couleur en dur ; CI de
branche verte au niveau job.

**Contrainte non negociable** : OFFLINE-PUR. Aucune donnee ne doit dependre d'une capture Cheat
Engine ni d'un acces online. Tout ce qui n'est decodable que par le walk delta calibre CE est
HORS PERIMETRE (voir Phase 3, renvoyee post-v7.5).

**Multi-titre** : le decodage film est **Halo Infinite uniquement**. La feature se degrade par
ABSENCE DE DONNEE (les champs `hp`/`d`/`a`/`g` sont simplement absents pour un autre titre ou un
match sans film) : le front n'affiche alors rien pour ces lignes, exactement comme le bouclier
absent aujourd'hui. Aucune comparaison de slug ; aucune panic ; aucune donnee d'un autre titre.

---

## Etat de l'art VERIFIE sur pieces (investigation 7 agents, 2026-08-12)

> Le doc `.ai/ETAT_DU_POC.md` (2026-07-27) est PERIME (i43/i42 y sont dits « non decodes/non
> cables » — FAUX au regard du code actuel). Ce tableau fait foi ; il est mesure sur l'artefact
> `data/cache/replays/halo_infinite/000d5950.json` et cite le code de `feat/v75`.

| Donnee | Decodee offline | Extraite/publiee | Cadence | Couverture 000d5950 | Statut |
|---|---|---|---|---|---|
| Bouclier (i5) | oui | oui (`Point.Sh`) | par-record, AU CHANGEMENT | 15,81 % pts / 73,7 % vies | **DEJA AFFICHE** (fiche+canvas) |
| Sante (i4) | oui | oui (`Point.Hp`) | par-record, AU CHANGEMENT | 0,56 % pts / 32,3 % vies | publie, **0 consommateur web** |
| Arme portee (paire) | oui | oui (`Loadout.W`) | keyframe ~20 s | 150 loadouts / 24 kf | affiche (liste) |
| Arme EN MAIN (drawn slot i42) | oui | oui (`Inventory.D`) | keyframe ~20 s | 150/184 ; {0:70,1:70,2:10} | publie, **pas mis en valeur** |
| Capacite equipee (index i48) | oui | oui (`Inventory.A` + `abilityLabels`) | keyframe ~20 s | 132/184 (71,7 %) ; table 4/11 | publie, **pas dans la fiche** |
| Grenades comptes (i22) | oui | oui (`Inventory.G` + `grenadeLabels`) | keyframe ~20 s | 120/184 (65,2 %) | publie (`InventoryRow`) |
| Munitions (i30..i42) | oui | oui (`Inventory.Am`) | keyframe ~20 s | 150/184 | **DEJA AFFICHE** |
| Lancers de grenade (events) | oui | oui (`doc.grenades`) | evenement | 70/70 (100 %) | publie, pas dans la fiche |
| Swap d'arme (diff keyframe) | oui | derivable | ~20 s (grossier) | 12-14/70 transitions | non calque |
| Armes au sol — IDENTITE (ti=42) | oui | oui depuis 2.1 (`KeyframeGroundWeapon`) | keyframe ~20 s | **196 lectures / 22 armes** (000d5950) | **LIVRE (decodeur)**, non publie |
| Armes au sol — POSITION (ti=42) | **NON (refutee, cf. 2.1)** | non | — | 3,3 % de slots stables ; temoin fantome | **INDECODABLE offline-pur** |
| Capacite ACTIVE (i57) | consumed_only | non | delta | — | not_offline dense |
| Grenade SELECTIONNEE / en main (i47) | consumed_only | non | delta | — | not_offline (voir Decisions) |
| Swap fin / arme equipee dense (i43) | not offline (calib CE) | non | delta | — | **Phase 3 (RE, hors v7.5)** |
| Event typé pickup/drop | NON (le film n'en porte pas) | non | — | — | non decodable |
| Compteur d'utilisations capacite | NON localise | non | — | — | non decodable |
| Surbouclier / regen bouclier | consumed_only, semantique non etablie | non | — | 0/4620 >1 | non exploitable |

Points d'ancrage code (verifies) :
- Walk vitals per-record (offline-pur, DEJA cable) : `filmdec/offline_biped.go:159,264` +
  `filmdec/offline_aim.go:125-206` -> peuple `Point.Hp/Sh/H` ; `replay/build.go:345-350`.
- Scan keyframe inventaire (offline-pur, DEJA cable) : `replay/inventory_decode.go:147` ->
  `Inventory{D,A,G,Am}` ; `replay/build.go:143-148` ; contrat `replay/document.go:99-134`.
- Scan keyframe loadout : `filmdec/keyframe_loadout.go:64` ; `replay/loadouts.go:59-97`.
- Front fiches : `web ReplayTeams.tsx` (PlayerCard, ShieldBar, InventoryRow, WeaponsRow) ;
  helper report+fondu `web replayLogic.ts:254-270` (`heldReading`, `freshness`).
- DROPS — IDENTITE : le filtre `keyframeBipedTI=35` de `filmdec/keyframe_loadout.go` ecartait
  `ti=42` ; depuis 2.1 le balayage est factorise (`familiesByRecord(pay, known, wantTI)`) et les
  armes au sol se lisent dans `filmdec/keyframe_ground_weapons.go`.
- DROPS — POSITION : **CORRIGE le 2026-08-12, l'affirmation initiale de ce plan etait FAUSSE.**
  Il etait ecrit « position via object-position-component (i0 world-object, 45 bits) deja porte
  `filmdec/traverse.go:229-258` ». Le COMPOSANT est bien porte, mais cela ne donne ni OU il commence
  dans un record de keyframe `ti=42` (le default-state de l'archetype n'est pas resolu — pas
  d'entree 42 dans `defaultStateDeserByTI`), ni QUEL slot d'un paquet delta est une arme (un record
  delta ne porte aucun typeIndex, et la bande de slots comblee est contaminee). La verite est celle
  de `.ai/SUIVI_REPLAY_2D.md` (2026-07-28) : « Objets au sol — bloque, et proprement ». La mesure
  2.1 la CONFIRME (temoin fantome + discriminant spatial). Ce plan aurait du reprendre ce verdict
  au lieu de le contredire.

---

## Decisions produit TRANCHEES (ne pas les rouvrir en cours d'execution)

1. **Sante** : afficher via le MEME patron que le bouclier (`heldReading` + maintien court +
   fondu de fraicheur + LACUNE si non lue). JAMAIS une jauge pleine par defaut. Raison mesuree :
   mediane 0 echantillon/vie ; une barre permanente serait vide/fausse la plupart du temps, et
   reporter EN ARRIERE (unique mesure = 30 % juste avant la mort) peindrait faux tout le debut de
   vie. La sante est une valeur ABSOLUE repliquee-au-changement, donc le report AVANT est honnete
   (inchangee), le report ARRIERE ne l'est pas. `healthHold` initial = 2000 ms (identique au
   bouclier) ; ajustable au gate visuel. Couleur = token semantique distinct du bouclier.
2. **Arme en main / secondaire** : l'arme EN MAIN (`W[D]` quand `D` ∈ {0,1}) s'affiche A GAUCHE,
   la secondaire (`W[1-D]`) A DROITE, comme dans le jeu. `D=2` (rien degaine, typiquement 1er
   keyframe) = pas de mise en valeur, ordre par emplacement. `D` absent/-1 = report de la derniere
   lecture avec son age, ou « non lu ». L'icone d'arme (deja disponible cote kill feed) peut etre
   reutilisee ; a defaut, le libelle. Reutiliser les tokens de couleur d'equipe existants.
3. **Capacite equipee** : afficher le nom (`abilityLabels[a]`) ; index hors table = garder le
   NUMERO marque non interpretable (patron existant `InventoryRow`/`abilityText`). Le COMPTEUR
   d'utilisations n'est PAS affiche (non localise — 36006 positions testees, aucune ne reproduit
   le releve). La capacite ACTIVE (i57) n'est PAS affichee (consumed_only, non publiee).
4. **Grenades** : afficher les COMPTES portes par rang (`Inventory.G`, deja dans `InventoryRow`) +
   optionnellement un marqueur des LANCERS depuis `doc.grenades`. La « grenade EN MAIN /
   selectionnee » n'est PAS affichee : elle n'est PAS decodable offline-pur (i47 consumed_only,
   pas d'archetype porte ; au mieux une inference du dernier lancer, ecartee pour ne pas afficher
   une inference comme une mesure).
5. **Swaps** : deriver du DIFF des loadouts d'un meme slot entre keyframes (granularite ~20 s),
   avec age affiche. PAS de swap fin intra-keyframe (Phase 3). Marquer clairement « etat de
   reference, ~20 s », jamais un suivi continu.
6. **Drops (armes au sol)** : calque OPTIONNEL Phase 2 (Go). **TRANCHE le 2026-08-12 : HORS v7.5.**
   La mesure 2.1 a rendu son chiffre — identite acquise (196 lectures / 22 armes), position REFUTEE
   — et l'utilisateur a choisi le report post-v7.5 (registre). Aucun calque n'est livre.
7. **Cadence honnete partout** : vitals = au changement ; loadout/arme-en-main/capacite/grenades
   = keyframe ~20 s avec AGE affiche et fondu. Toute valeur non lue = LACUNE, jamais un defaut.
8. **HORS PERIMETRE v7.5, renvoye Phase 3 / registre** : suivi dense per-record (arme equipee au
   swap fin, capacite active continue, grenade selectionnee) — bloque par le walk delta non
   offline-pur (calibration CE i0/i21 + faute de corps i22). C'est un chantier de RE, pas un
   cablage. Aucun executeur ne doit s'y engager dans ce lot.

---

## Phase 1 — Cablage web des fiches enrichies (offline-pur, DONNEE DEJA PUBLIEE)

> Effort : MOYEN, front uniquement (aucune ligne Go de decodage — tout est deja dans l'artefact
> et le contrat OpenAPI genere : `Point.hp/sh`, `Inventory.d/a/g/am`, `doc.grenades`). Livrable
> independamment. Perimetre FERME ci-dessous.

- [x] **1.1 Barre de sante** — ajouter `health: {value, age} | null` a `PlayerState`
  (`rosterLogic.ts`), peuple par `heldReading(live.points, frame, p => p.hp, HEALTH_HOLD)` ;
  composant `HealthBar` calque sur `ShieldBar` (fondu `freshness`, LACUNE si null, token couleur
  sante). JAMAIS 100 % par defaut. `HEALTH_HOLD = 2000` ms initial.
  - Gate : `grep -n "p.hp" apps/web/src/features/match-replay/*.ts*` retourne >=1 lecteur ;
    `make check-types` ; `make test-web` (nouveau test `rosterLogic` sur report+lacune sante).
  - FAIT (2026-08-12) : `rosterLogic.ts` (PlayerState.health + healthHold), `HealthBar` dans
    `ReplayTeams.tsx` (token `success`, bouclier au-dessus), 4 tests sante (report avant,
    jamais arriere, expiration, mort). Gates verts.
- [x] **1.2 Arme en main a gauche / secondaire a droite** — dans `WeaponsRow` (ou nouveau
  `EquippedWeaponsRow`), ordonner par `Inventory.D` : `W[D]` a gauche marquee « en main »,
  `W[1-D]` a droite ; `D=2`/absent gere selon Decision 2. Reutiliser l'icone d'arme si dispo.
  - Gate : test `killFeedLogic`/nouveau `equippedLogic` sur les 3 cas (`D=0`, `D=1`, `D=2`) ;
    `make test-web`.
  - FAIT (2026-08-12) : nouveau `equippedLogic.ts` (equippedWeapons, pur), `WeaponsRow`
    reordonne + tag « en main » typographique (pas d'icone : le document replay ne porte pas
    d'URL d'icone — `WeaponLabel = {en, fr, fx?}` — le libelle fait foi, comme prevu par le
    plan). 7 tests (D=0/1/2, absent, desapparie, fuite de slot, null). Gates verts.
- [x] **1.3 Capacite equipee dans la fiche** — afficher `abilityLabels[Inventory.A]` (nom
  bilingue), index hors table -> numero marque non interpretable. Reutiliser `abilityText`.
  - Gate : test rendu capacite connue + inconnue ; `make test-web`.
  - FAIT (2026-08-12) : le rendu existait DEJA (`InventoryRow`/`abilityText`, verifie sur
    pieces — l'etat de l'art « pas dans la fiche » etait perime sur ce point) ; ajout du test
    de rendu manquant (`ReplayTeams.test.tsx` : connue, inconnue, non lue) + title
    `abilityLabel` bilingue. Gates verts.
- [x] **1.4 Grenades portees** — verifier que `InventoryRow` rend deja `Inventory.G` par rang ;
  si non, l'ajouter (labels `grenadeLabels`). Optionnel : pastille de LANCER depuis `doc.grenades`.
  - Gate : test rendu comptes + lacune (`GrenadesRead=false`) ; `make test-web`.
  - FAIT (2026-08-12) : rendu deja cable (verifie sur pieces : `grenadesCarried` +
    `InventoryRow`), tests de rendu ajoutes (comptes par rang, lacune `g` absent). Pastille de
    lancer : NON RETENUE — les lancers sont deja un calque du canvas (`layerGrenades`) ; la
    dupliquer dans la fiche melangerait evenement et etat porte. Gates verts.
- [x] **1.5 Swaps (diff keyframe)** — deriver cote client le changement de `Loadout.W` d'un meme
  slot entre deux keyframes ; indicateur discret avec AGE. PAS de calque serveur.
  - Gate : test diff sur 2 loadouts same-slot ; `make test-web`.
  - FAIT (2026-08-12) : `loadoutSwapAt` (diff MULTIENSEMBLE des deux dernieres lectures du
    slot) + `SwapMark` discret (libelle « echange », detail +/- et age dans l'infobulle,
    mention « etat de reference, pas un suivi continu »). 6 tests. Gates verts.
- [x] **1.6 i18n FR+EN** — toutes les strings nouvelles dans `match-replay/i18n.ts` (parite par
  typage `Record<Locale, T>`). Libelles « en main », « secondaire », « Sante », « Capacite ».
  - Gate : `make check-types` ; lint i18n vert.
  - FAIT (2026-08-12) : 10 strings nouvelles FR+EN (healthUnread/healthLabel/shieldLabel/
    abilityLabel/weaponInHand/-Hint/weaponSecondaryHint/weaponsHolstered/weaponSwap/-Hint),
    parite forcee par le typage (`ReplayText`). « secondaire » = infobulle sur l'arme non
    degainee quand une main est designee. `tsc -b` vert ; eslint 0 erreur (19 warnings
    preexistants hors match-replay, downgrades documentes dans eslint.config.js).
- [x] **1.7 Couleurs** — tokens semantiques uniquement (skill `color-tokens`), zero hex/Tailwind
  couleur.
  - Gate : `grep -rnE "#[0-9a-fA-F]{3,6}|bg-(red|blue|green)" apps/web/src/features/match-replay/` = vide.
  - FAIT (2026-08-12) : sante = token `success` (distinct du bouclier `info`), marquage « en
    main » purement typographique. Le grep du gate remonte UNE ligne : un hex CITE dans un
    commentaire preexistant de `canvasInk.ts:10` qui documente precisement l'interdiction des
    litteraux — aucune couleur appliquee ; le meme grep restreint aux fichiers touches par ce
    lot est VIDE. Fichier hors perimetre non modifie (zero fix opportuniste).
- [x] **1.8 Degradation multi-titre** — verifier qu'un artefact sans `hp`/`d`/`a` (ou H5) rend la
  fiche SANS ces lignes, sans erreur (les champs sont deja `?:` dans le contrat genere).
  - Gate : test rendu fiche avec `Point` sans `hp` et `Inventory` sans `d/a`.
  - FAIT (2026-08-12) : 2 tests de rendu (`titleSlug` neutre + champs absents -> lacunes
    dites, aucun marquage, zero erreur ; document reduit aux traces -> toutes lacunes dites).
    Aucune comparaison de slug nulle part. Gates verts.

**Gate de Phase 1** : `make check-types` + `make test-web` verts ; gate visuel utilisateur sur
`000d5950` (temoins choisis par l'utilisateur) ; entree `thought_log.md`.

**Etat du gate de Phase 1 (2026-08-12)** : `make check-types` VERT + `make test-web` VERT
(409 fichiers, 3600 tests, 14 skips preexistants) ; entree `thought_log.md` FAITE ; **gate
visuel utilisateur EN ATTENTE** (report valide : temoins choisis par l'utilisateur, rendu
pret sur `000d5950`). Couverture mesuree sur l'artefact (voir journal ci-dessous).

---

## Phase 2 — Calque DROPS (armes au sol) — MESURE LIVREE, CALQUE ABANDONNE POUR v7.5

> **ETAT AU 2026-08-12 (decision utilisateur, OPTION 1 : REPORT POST-v7.5).** 2.1 est livree :
> l'IDENTITE des armes au sol est decodee (196 lectures / 22 armes sur `000d5950`). La POSITION est
> REFUTEE sur pieces — le calque cartographique n'a donc pas d'entree, et **2.2 / 2.3 sont
> ANNULEES** (pas « en attente »). Report inscrit au registre
> `.ai/V7.5/REGISTRE_REPORTS.md` avec sa condition de reprise. Une version NON SPATIALE (liste
> « quelles armes gisent au sol », sans position) reste une option produit si reprise.
>
> Effort constate : la mesure a coute l'essentiel du lot ; le cablage suppose n'a jamais eu lieu
> faute d'entree decodable.

- [x] **2.1 Recolter l'identite des armes au sol (ti=42) et MESURER, position comprise** au
  keyframe : lever le filtre `keyframeBipedTI=35` (`filmdec/keyframe_loadout.go`) pour COLLECTER
  aussi les familles `ti=42` — **196 lectures mesurees** avec le catalogue de production (le « 397 »
  de la premiere redaction de ce plan est une autre regle de compte, cf. ci-dessous) — puis
  eprouver leur position. Nouveau scan `ScanFilmKeyframeGroundWeapons` (analysis/filmdec), pur,
  sans capture. MESURER la couverture sur `000d5950` et la consigner.
  L'enonce initial « et leur position (object-position-component i0 world-object, 45 bits, deja
  porte `traverse.go:229-258`) » est **CORRIGE : cette position n'existe pas a la lecture** (voir
  POSITION ci-dessous et les points d'ancrage code, section Etat de l'art).
  - Gate : `cd apps/go-api && go test ./internal/analysis/filmdec/ -run GroundWeapon` vert ;
    couverture consignee dans le CR.
  - FAIT (2026-08-12) : `filmdec/keyframe_ground_weapons.go` (scan `ScanFilmKeyframeGroundWeapons`,
    pur, hors ligne) + factorisation `familiesByRecord(pay, known, wantTI)` dans
    `keyframe_loadout.go` (le balayage bit a bit n'existe QU'UNE FOIS : ti=35 et ti=42 ne sont plus
    que deux appelants) + 7 tests `-run GroundWeapon` VERTS + instrument de mesure
    `replay/ground_weapon_research_test.go` (sous garde `GW_FILM`, saute en CI).
  - **IDENTITE : ACQUISE.** `000d5950` : 26 images-cles, 269 records ti=42 (178 slots distincts),
    **196 lectures portant une famille d'arme connue**, 196 occurrences, 169 vies (slot,gen),
    **22 armes nommees distinctes** (Gravity Hammer 22, Sentinel Beam 16, SPNKr 11, S7 Sniper 11,
    Stalker Rifle 11, Disruptor 10, Energy Sword 10...). **Zero fuite d'alias** : 0 paire
    consecutive « meme nom, famille differente » sur 172 paires (4 paires legitimes du meme
    modele). Correction de l'etat de l'art : les « 397 occ. » de l'en-tete du decodeur relevent
    d'une autre regle de compte (occurrences brutes, alias et familles hors catalogue comprises) —
    temoin de comparabilite sur le MEME film : cote arme PORTEE le meme balayage rend 300
    occurrences la ou l'en-tete annonce 495. Avec le catalogue de PRODUCTION
    (`weaponv3.KnownWeaponHigh32`), le chiffre juste est **196**.
  - **POSITION : REFUTEE, mesuree.** [!] La position d'une arme au sol n'est PAS publiable.
    Bande ti=42 (899 slots apres comblement) -> 661 slots peuples / 44 498 echantillons ; TEMOIN
    FANTOME de meme cardinalite (899 slots jamais vus dans aucune image-cle) -> 493 slots peuples /
    10 950 echantillons : la seule PRESENCE d'une position n'informe donc presque rien (55 % des
    slots fantomes en ont). Discriminant spatial (une arme posee NE BOUGE PAS) : sur 458 slots
    reels a >=3 echantillons, **3,3 % seulement tiennent dans 0,5 u** et **62,4 % s'etalent sur
    plus de 20 u** (fantome : 0,5 % et 82,4 %). L'ecart de distribution existe mais la majorite des
    « positions » de slots reels est du bruit. Ceci CONFIRME le verdict du 2026-07-28
    (`.ai/SUIVI_REPLAY_2D.md`, « objets au sol — bloque, et proprement ») au lieu de le lever :
    l'hypothese du plan (« position deja portee, reste a recolter ») est FAUSSE. Cause structurelle :
    au keyframe l'offset d'i0 depend du default-state de l'archetype, non resolu pour ti=42
    (`defaultStateDeserByTI` n'a pas d'entree 42) ; sur la voie delta le record ne porte pas de
    typeIndex et la bande de slots comblee est contaminee.
  - **CONSEQUENCE DE PERIMETRE** : un calque « armes au sol » POSE SUR LA CARTE est hors de portee
    offline-pur ; 2.2/2.3 n'ont plus d'entree. **DECISION UTILISATEUR (2026-08-12) : OPTION 1,
    report post-v7.5** — la mesure est livree, la Phase 2 spatiale est abandonnee pour v7.5.
- [!] **2.2 Calque document — ANNULEE** (decision utilisateur 2026-08-12, report post-v7.5). Le
  champ `GroundWeapons` dans `ReplayDocument` n'a pas de contenu publiable : sans position, un
  calque de document ne porterait rien que le client puisse poser. Ni `document.go`, ni
  `build.go`, ni le contrat OpenAPI, ni `generate-types` ne sont touches par ce lot.
  Justification : POSITION refutee en 2.1. Report : registre `.ai/V7.5/REGISTRE_REPORTS.md`.
- [!] **2.3 Rendu web — ANNULEE** (meme decision, meme cause). Zero ligne web dans ce lot.
  Une version NON SPATIALE (liste des armes au sol a l'instant lu, sans position) reste une option
  produit a la reprise — elle exigerait 2.2 sous une autre forme, donc un nouveau lot.

**Gate de Phase 2 (cloture 2026-08-12)** : la phase se clot sur la MESURE, pas sur un rendu.
Tests Go verts (`-run GroundWeapon` 7/7, `filmdec/` et `replay/` complets), `golangci-lint` 0 issue,
`make gate-push`, entree `thought_log.md`, report au registre. **Aucun gate visuel** : rien de
visible n'a ete produit (2.3 annulee). Garde `handlers/replay_local_gate.go` INCHANGE.

---

## Phase 3 — HORS v7.5 (registre) : suivi dense per-record

> NE PAS EXECUTER dans ce lot. Consigne ici pour fermer le perimetre.

Le suivi dense per-record (arme equipee au swap fin, capacite active continue, grenade
selectionnee i47/i48, pickup/swap par frame) exige de rendre le walk delta bit-exact `i0..i43`
OFFLINE : resoudre les largeurs de precision `i0/i21` (aujourd'hui issues d'une capture Cheat
Engine, `traverse.go:1199-1209` « NOT a general decode path. Empty by default ») ET corriger la
faute de corps `i22` (92,46 % de comptes impossibles, `frame_records.go:81`). C'est un chantier de
RETRO-INGENIERIE, pas un cablage. -> Registre `.ai/V7.5/REGISTRE_REPORTS.md`, condition de reprise :
« walk delta offline bit-exact jusqu'a i43 ».

**Non decodable (a ne pas chercher)** : evenement typé pickup/drop (le film n'en porte aucun) ;
compteur d'utilisations de capacite (non localise) ; surbouclier/regen bouclier (semantique non
etablie, temoin de recharge echoue).

---

## Decouvertes (a consigner ici pendant l'execution, NE PAS traiter hors perimetre)

- (2026-08-12, Phase 1) **Flake local d'un guard-test** : sur deux runs `vitest run` complets
  A CODE IDENTIQUE, un guard-test scannant `src/` (famille `srcRoot = resolve(process.cwd(),
  'src')`, ligne ~35 — fragDetailBreakdown/dominance/divergentZeroGradient) a echoue UNE fois
  puis passe DEUX fois. Ressemble a une contention I/O Windows (readFileSync recursif sous
  antivirus/indexeur), pas a un defaut de code. A surveiller si la CI (Linux) le reproduit —
  rien fait dans ce lot.
- (2026-08-12, Phase 1) **L'etat de l'art du plan etait perime sur 1.3/1.4** : la capacite et
  les grenades etaient DEJA rendues dans la fiche (`InventoryRow`) — seuls les tests de rendu
  manquaient. Le tableau « publie, pas dans la fiche » date d'avant le cablage d'InventoryRow.
- (2026-08-12, Phase 1) **`canvasInk.ts:10`** : un hex cite dans un COMMENTAIRE (qui documente
  l'interdiction des hex) fait remonter une ligne au grep du gate 1.7. Pas une violation
  (aucune couleur appliquee) ; a reformuler un jour si on veut un grep strictement vide —
  hors perimetre de ce lot.
- (2026-08-12, Phase 1) **Regle « <= 2 copies » declenchee sur les fixtures de test** : la 3e
  copie du document-de-test allait naitre -> kit `test/testDoc.ts` + garde-rail
  `testDoc.guard.test.ts` (interdit `normalizeReplayDocument(` dans les tests de la feature
  hors frontiere), 2 fixtures existantes migrees. Le kit vit sous `test/` : segment whiteliste
  par la regle eslint `@levelup/no-title-slug-literal` (fixtures), qui frappe a raison tout
  fichier non-test portant un slug litteral.

- (2026-08-12, Phase 2.1) **L'etat de l'art du plan etait FAUX sur la position des armes au sol** :
  le tableau la disait « portee » (« position via object-position-component i0 world-object, 45 bits,
  deja porte `traverse.go:229-258` »). Le composant est effectivement porte, mais rien ne dit OU il
  commence dans un record ti=42 (default-state d'archetype non resolu) ni quel slot delta est
  reellement une arme (record delta sans typeIndex). Le doc `.ai/SUIVI_REPLAY_2D.md` portait deja le
  verdict depuis le 2026-07-28 et le plan ne l'avait pas repris. Lecon : greper les verdicts
  anterieurs sur la MEME grandeur avant d'ecrire « il ne reste qu'a recolter ».
- (2026-08-12, Phase 2.1) **`filmdec.GroundWeaponPositions` / `WorldObjectPositionsForBand` n'ont
  qu'un consommateur : l'instrument de mesure** — arbitrage RENDU a la cloture : on les GARDE, parce
  que la refutation doit rester rejouable (`GW_FILM=<film> go test ./internal/analysis/replay/ -run
  GroundWeapon`) ; sans elles, la seule trace du NO-GO serait une phrase de doc. Leur unique
  consommateur est ecrit en tete de chaque fonction, comme l'exige la regle « zero code mort » : ce
  n'est pas du code « au cas ou », c'est l'instrument d'une mesure datee.

## Journal Phase 1 (2026-08-12)

- Commit unique du lot (voir git) ; executeur : plan-execution au contrat, ordre 1.1 -> 1.8.
- **Couverture mesuree sur `000d5950`** (script Node ponctuel sur l'artefact, 99 vies,
  29 221 points, 184 lectures d'inventaire, 150 loadouts, 80 slots) :
  - Sante : 163 points porteurs (0,56 %), 32/99 vies (32,3 %) — la LACUNE est l'etat
    ordinaire, conforme a la decision 1 (pas de jauge permanente).
  - Selecteur d'emplacement `D` : 150/184 lectures (81,5 %), distribution {0: 70, 1: 70,
    2: 10} — une main designee sur 140 lectures, « armes rangees » sur 10 (1re image-cle).
  - Capacite `A` : 132/184 (71,7 %) et 132/132 dans la table du document -> AUCUN
    « capacite inconnue (N) » attendu sur CE match (la table 4/11 couvre tout ce qui y est lu).
  - Grenades `G` : 120/184 (65,2 %). Lancers publies : 70.
  - Swaps same-slot entre lectures consecutives : 14 sur 70 paires — l'indicateur « echange »
    apparaitra ~14 fois sur le match, conforme a l'estimation du plan (12-14).
- Gate visuel : rendu PRET sur `000d5950` (page replay, garde local inchange). Les temoins
  restent a choisir par l'utilisateur.
- Seuil de taille (regle repo n° 5) : les ajouts ont fait franchir 500 L a `ReplayTeams.tsx`
  (579) -> extraction de la rangee d'armes dans `ReplayWeaponsRow.tsx` (WeaponsRow + SwapMark,
  122 L) ; `formatSeconds` et `READING_FADE` remontes dans `replayLogic.ts` (partages sans
  import circulaire). ReplayTeams.tsx retombe a 462 L. Gates rejoues apres extraction :
  typecheck (cache tsbuildinfo purge), eslint, vitest complet — VERTS.

## Contraintes transverses (rappel grille plan-review)

- Couches Go : decodage pur en `internal/analysis/filmdec` ; assemblage en
  `internal/analysis/replay` ; DTO d'artefact en `replay/document.go` ; aucun acces DB/HTTP.
- Tests par couche : filmdec (unitaire), replay (assemblage sur fixture), web (hooks +
  rendu jsdom). Logging : `slog.*Context` cote Go si nouvelle op significative ; jamais
  `fmt.Println`. Front : routes file-based, query keys dans `lib/query/keys.ts`, i18n FR+EN,
  tokens couleur.
- Offline-pur : aucun chemin ne doit appeler le walk delta calibre CE. Le decodage lourd reste
  hors ligne (I/O disque sur les chunks du film).

## Suivi / reprise de session

- Avancement = les cases de ce fichier + `thought_log.md`. Reprise : lire ce plan, puis
  `git -C <worktree> log --oneline -8 feat/v75`, puis rouvrir la premiere case non cochee.
- Contrat d'execution : skill `plan-execution` (fait foi). Cloture de lot = gate de phase +
  CI de branche verte au niveau job (mode branche unique v7.5, pas de merge).
