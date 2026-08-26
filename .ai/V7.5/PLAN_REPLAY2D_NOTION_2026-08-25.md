# PLAN — Encadré Notion « REPLAY 2D » (11 points) — 2026-08-25

> Contrat : skill `plan-execution`. Pilotage : superviseur (session courante), exécuteurs
> Opus, UN agent par worktree FRÈRE créé depuis la racine en chemin absolu, base
> `origin/feat/v75` (`e2f56e2c7`). Exécuteurs : ne touchent NI `thought_log` NI
> `REGISTRE_REPORTS` NI Notion, jamais `git add -A`, commits sur leur branche `wt/*`
> uniquement, JAMAIS de push. Fusion `--no-ff` + journal + registre + push + CI au niveau
> JOB = superviseur. Verdicts de gates : commandes NUES (jamais de pipe qui masque l'exit).
> Reprise de session : lire ce plan (statuts) + tête de `.ai/thought_log.md`.

Source : encadré « REPLAY 2D » (callout jaune) de la page Notion « Backlog LevelUp »
(`39a7ae87-e7a3-809e-8e03-e4ffedcf5086`), 11 points numérotés.

## Triage des 11 points

| # | Point Notion | Statut / lot |
|---|---|---|
| 1 | Découper les zones de callout sur le décor réel | `[~]` barré dans Notion ; piste « zone jouable » déjà au catalogue callouts — ne pas traiter |
| 2 | Dispositifs de carte (canon humain / ascenseur / souffleur) | `[~]` TROUVÉ, reporté au registre le 24/08 (industrialisation sur demande produit) — ne pas traiter |
| 3 | Traits au bout du cône de visée : c'est quoi ? | Lot A (diagnostic obligatoire + fix si artefact de rendu) |
| 4 | Fiches : retirer la data temps morts + le symbole ami | Lot A |
| 5 | Notif Discord quand un rejeu est généré/récupéré (local ou ouvrier), groupée anti-spam | Lot B |
| 6 | Kill feed : retirer le compteur total d'éliminations (haut droite) | Lot A |
| 7 | Bases non prises : contour grisé | Lot A (si l'état est dérivable du document ; sinon `[!]` → condition de reprise lot D) |
| 8 | Barre de volume : mute = curseur à 0, ne pas disparaître | Lot A |
| 9 | Fin de rejeu : rester sur l'état final, pas de remise à zéro | Lot A |
| 10 | Cartes non validées : rogner toits/plafonds au-dessus des hauteurs fréquentées | Lot C (mesure d'abord, implémentation si concluant) |
| 11 | Objectifs de mode : état vivant (placement statique acquis) ; VIP absent du film → substitut `managed-objective-object-reference` | Lot D (plan dédié en phase 0) |

## Lot A — `wt/ui-rejeu` (web, `apps/web/src/features/match-replay/`)

Items (chaque item statué `[x]`/`[~]`/`[!]` au CR) :

- [x] A1 (pt 4) : retirer la donnée « temps morts » des fiches joueur (ancre :
  `ReplayTeamsDeadTime*`) ET le symbole ami. Supprimer le code débranché + tests + strings
  i18n orphelines (règle 0 code mort).
  FAIT : `DeadTimeRow` + `deadTimeLogic.ts` + ses 2 fichiers de tests SUPPRIMÉS, clés
  `deadTimeLabel` / `deadTimeUnmeasurable` retirées (FR, EN, contrat) ; `<PlayerMark>` retiré
  de la fiche avec la prop `marks` de `ReplayTeams` et son passage depuis la route. Le
  composant `PlayerMark` RESTE : le fil des éliminations en est l'autre lecteur.
- [x] A2 (pt 6) : retirer le compteur total d'éliminations en haut à droite du kill feed
  (ancre candidate : `ReplayCountersBadge`). Même règle 0 code mort.
  FAIT — **l'ancre candidate du plan était FAUSSE** : `ReplayCountersBadge` est le badge
  frags/morts/assistances d'UNE FICHE joueur (`ReplayTeams.tsx:308`), il n'a rien à voir avec
  le fil. Le vrai compteur était le rapport `« affichées / total »` écrit en dur dans l'en-tête
  du fil (`ReplayKillFeed.tsx`, en-tête `flex justify-between`). Retiré ; `ReplayCountersBadge`
  est INTACT.
- [x] A3 (pt 8) : couper le son ne fait plus disparaître la barre de volume : le curseur
  passe à 0 ; ré-activer restaure le volume précédent (`ReplaySoundControls`).
  FAIT : le `{sound.on && …}` qui escamotait le curseur est retiré ; il reste, à 0, `disabled`
  et estompé, avec une infobulle dédiée (nouvelle clé `soundVolumeMutedHint`, FR + EN). La
  RESTAURATION était déjà acquise côté hook (`toggle` coupe le maître du lecteur, il n'écrit
  jamais `volume`) — le zéro affiché est un affichage, pas une écriture ; 3 tests le tiennent.
- [x] A4 (pt 9) : à la fin du rejeu, rester sur l'état FINAL (curseur à T_final, scène
  finale affichée) — pas de remise à zéro (`ReplayTransport`). Relancer reste possible.
  FAIT — l'ancre réelle n'était PAS `ReplayTransport` (qui n'a aucun état) mais la boucle rAF
  de `ReplayCanvas.tsx` (`if (next >= doc.frameCount - 1) next = 0`). Le canvas était à 2 lignes
  de son cliquet de taille : la LECTURE part dans `useReplayPlayback.ts` (8e extraction,
  777 -> 742). Fin = borne à `endFrame`, dernier `draw()` + `soundTick`, puis pause ; « Lecture »
  sur un rejeu terminé rembobine. 7 tests, mordant prouvé par double mutation.
- [~] A5 (pt 3) : DIAGNOSTIC des « traits au bout du cône de visée » (rendu canvas —
  `ReplayCanvas` et calques). Le CR DOIT répondre « c'est quoi » avec fichier:ligne.
  Décision par défaut : artefact de rendu → corriger ; donnée réelle légitime (ex. trace
  de tir) → NE PAS supprimer, expliquer au CR et statuer `[~]`.
  RÉPONSE : c'est `drawPitchTick` (`replayAimCone.ts:139-157`, appelé depuis `drawAimCone`
  `:87`) — le SIGNE de l'ÉLÉVATION de visée (champ `p`, schéma 13). Le cosinus est PAIR :
  viser 30° au-dessus et 30° en dessous raccourcissent le cône exactement pareil, le trait est
  ce qui les départage (dehors = lève la tête, dedans = pique). DONNÉE RÉELLE, mesurée,
  testée (`replayMarkers.test.ts:154-221`) : NON SUPPRIMÉE. Le « parfois » est la zone morte
  de 2° (`AIM_TICK_DEAD_DEG`, `:37`) plus les artefacts antérieurs au schéma 13, qui ne
  portent pas `p`. SEUL CHANGEMENT : l'infobulle du calque « Visée » le décrit désormais
  (FR + EN) — elle ne le mentionnait pas, d'où la question.
- [x] A6 (pt 7) : zones/bases non prises (= neutres, aucune équipe propriétaire à
  l'instant t) : contour grisé (token sémantique, pas d'hex). Si l'état « non prise »
  n'est PAS dérivable du document schéma 18 → `[!]` + condition de reprise = lot D.
  DÉRIVABLE — pas de `[!]`. Preuve : `ZoneSpan.Owner *int` (`document_zones.go:199-205`),
  « l'equipe qui TIENT la zone, ou `null` quand personne ne la tient (valeur neutre
  `0xFFFFFFFF` du canal) » — pointeur SANS `omitempty` justement pour que « personne » se
  distingue d'un artefact plus ancien. Côté client, `spanStateAt` le sert déjà.
  FAIT : (a) le contour d'une zone non prise passe EN RETRAIT (α 0,95 -> 0,5 ; 2,5 -> 1,6 px) —
  il s'affirmait auparavant exactement autant que celui d'une base gagnée ; (b) l'encre neutre
  des objectifs passe de `--muted-foreground` (variable de LAYOUT lue par `readInk`) au TOKEN
  `divergent-neutral`, celui du fil pour une mort neutre ; (c) le seuil est `owner === null`,
  PAS `held` — une zone tenue par un camp non situable (aucune ligne « moi ») garde le trait
  plein. Mordant prouvé par double mutation (dont la confusion `held`/`owner`).

- [ ] A7 (demande utilisateur 25/08, reçue en cours de pilotage) : fiche joueur à l'état
  MORT : (a) retirer l'accentuation de la bordure gauche de la fiche ; (b) centrer le
  compteur de réapparition ; (c) retirer la jauge (barre de progression de réapparition).
  Ancres candidates : `ReplayTeams.tsx` / `ReplayVitality.tsx` — vérifier sur pièces.
  Mêmes gates que A1-A6 ; clés i18n orphelines supprimées le cas échéant ; 0 code mort.
- [ ] A8 (demande utilisateur 25/08, reçue en cours de pilotage) : (a) retirer le bandeau
  « Données de rejeu d'une version antérieure — certains éléments peuvent manquer. »
  (clé i18n FR+EN + contrat + logique d'affichage ; NE PAS supprimer la constante
  `EXPECTED_REPLAY_SCHEMA_VERSION` si elle a d'autres lecteurs — vérifier sur pièces) ;
  (b) retirer l'accentuation sur les lignes du kill feed (même esprit que la bordure de
  fiche A7 — identifier l'accent exact sur pièces avant de retirer). Mêmes gates ;
  0 code mort.

Gates (dans le worktree, exit codes réels) : `npm ci` (autorisé), typecheck
(`npx tsc -b` via `make check-types` ou équivalent local), `npx vitest run` ciblé
`match-replay`, lint web. i18n : toute string FR **et** EN. Pas de vérification
navigateur (gate visuel = utilisateur, à la fusion).

STATUT LOT A (25/08) : A1-A6 FUSIONNÉS dans feat/v75 (merge --no-ff, gates rejoués au
principal après fusion : tsc -b exit 0 cache purgé, vitest 75 fichiers / 1083 tests
exit 0). Arbitrages superviseur : phrase `layerAimHint` GARDÉE (elle répond à la question
utilisateur dans le produit) ; cliquet canvas 742 GARDÉ (croissance par extraction
uniquement — consigne transmise au lot D). A7 en vol (2e passe du même exécuteur,
même branche `wt/ui-rejeu`, 2e fusion à son CR).

## Lot B — `wt/notif-rejeu` (Go, pt 5)

- [x] B0 (validé superviseur 25/08 — vérifié sur pièces : ancrage `writeArtifactBytes`
  avec publication APRÈS écriture réelle seulement, chaînes locale+ouvrier prouvées ;
  arbitrages : A1 liens retenus via méthode de repo `OpenReadForQuery` sans SQL inline
  dans wire, A2 `discord_notify_replay` défaut TRUE retenu, A3 admin notifie aussi ;
  go B1 TENU jusqu'à la fin des builds Go du lot C — pas de builds concurrents) :
  mini-plan de l'exécuteur AVANT tout code, livré comme CR intermédiaire —
  STOP jusqu'à validation superviseur. Contenu exigé : point d'ancrage serveur UNIQUE de
  « artefact de rejeu enregistré » couvrant les DEUX chemins (génération locale ET livraison
  par l'ouvrier distant) avec fichier:ligne ; réutilisation du canal webhook Discord
  existant (pattern `discord_webhook_present`, env ou store) ; mécanisme de groupement.
- [ ] B1 : implémentation après go. Cadre imposé : groupement par fenêtre (défaut 10 min,
  1er événement arme le timer, un seul message « N rejeux prêts » + liste des matchs) ;
  perte du groupe en cours au redémarrage = ACCEPTÉE et documentée ; logique dans un
  service/notifier (JAMAIS dans un handler HTTP) ; `slog.InfoContext/ErrorContext`
  systématiques ; aucune erreur avalée ; aucun flag qui laisse la feature OFF.
- [ ] B2 : tests — groupeur avec horloge injectée (unitaires purs), test du point
  d'ancrage. Gates : `go test` ciblé NU, `go vet`/lint, arch-rules respectées.

## Lot C — `wt/plafonds` (pt 10)

- [x] C0 (MESURE, stop au verdict) : par carte NON validée, distribution des hauteurs
  des positions joueurs du corpus d'artefacts cuits (itération film par film — JAMAIS de
  balayage corpus en un processus, leçon bombe RAM) vs hauteur des éléments de géométrie
  rendus. Livrable : tableau par carte (h_max fréquentée, volumes de géométrie au-dessus,
  % de faces rognées), 2 cartes témoins avant/après. STOP — validation superviseur.
  **FAIT — `.ai/V7.5/MESURE_PLAFONDS_C0.md`** (instrument `cmd/mapplafond-mesure`, planches
  `.ai/V7.5/dumps/plafonds_c0/`). **VERDICT : non concluant pour le besoin, réfuté comme règle
  générale.** (a) sur une arène couverte le toit vit dans la même bande que le sol qu'il cache
  — Aquarius a TOUTE sa matière sous sol+9,9 m, un seuil « h max frequentée + 5 » tombe entre
  sol+8,1 et sol+14,5 et ne coupe rien ; (b) là où le seuil mord, il mord des cartes VALIDÉES
  (catalyst 37,5 % de l'image, ridgeline 23,2 %, chasm 14,9 % de ses volumes) ; (c) les cartes
  réellement non validées sont les 37 fonds Forge, ni mesurables ni re-cuisinables (`.mvar`
  gitignorés et absents). 3 préalables avant tout C1 : périmètre confirmé par l'utilisateur,
  `.ai/re_dump/mapvar` restauré, définition de « carte validée » écrite en donnée.
- [!] C1 NON OUVERTE (validation superviseur 26/08, CR C0 vérifié sur pièces : commits,
  mesure, claim `map_objectives.json` re-comptée 58-63/73 selon le motif). Le verdict C0
  réfute la coupe par plafond global ; l'ouvrir quand même produirait des régressions sur
  des cartes validées. Conditions de reprise consignées au registre (confirmation du
  périmètre par l'utilisateur, restauration `.ai/re_dump/mapvar` depuis la clé PNY,
  « carte validée » écrite en donnée). Lot fusionné pour l'instrument + la mesure
  (réfutation datée et rejouable).

## Lot D — `wt/obj-etat` (pt 11)

- [x] D0 (VALIDÉ superviseur 25/08, CR vérifié sur pièces — plan `662b6c953` sur
  `wt/obj-etat`, 559 L. DEUX CORRECTIONS DE FAIT confirmées : (1) schéma 19 DÉJÀ PRIS
  (`document.go:158`) par le lot « lecture vide » d'une session voisine → cible du lot =
  20, bump UNIQUE en D5 ; (2) la jauge tag 3 est publiée sur les modes à zones
  SIMULTANÉES (Bastion) et volontairement ABSENTE en KOTH (`zone_states_gauge.go:33-36`)
  — l'inverse de ce que ce plan affirmait. ARBITRAGES : ordre D6(D-R)→D2→D3→D4 ; TC
  statique gardé en l'état jusqu'à D3 (pas de churn intermédiaire, remonter au user si
  D3 échoue) ; mandat d'amender `[[objective_objects]]` accordé SI gate D4 passe, avec
  justification datée ; VIP `[!]` DÉFINITIF ; Extraction/Stockpile/Land Grab/Firefight
  `[!]` corpus avec conditions de reprise) : écrire `PLAN_OBJECTIFS_ETAT_VIVANT_2026-08.md` :
  inventaire PAR MODE de l'état vivant manquant (acquis : KOTH colline désignée + jauge
  schéma 18, CTF `flagCarries` + rendu drapeau ; à inventorier : Strongholds/Total
  Control possession vivante, Extraction, Oddball (crâne `[!]` corpus), VIP → mesurer le
  substitut `managed-objective-object-reference`) ; pour chaque mode : canal mesurable,
  oracle + témoin négatif, seuil de gate ; UN SEUL bump schéma (19) pour tout le lot ;
  re-cuisson des témoins seulement. STOP — validation superviseur.
- [ ] D1+ : implémentation par mode, gate mesuré par phase. **GO UTILISATEUR 25/08** :
  « KOTH et Total Control à finaliser correctement, pour Oddball aussi. VIP on verra
  plus tard. Ton ordre est correct. » => périmètre confirmé D2 (KOTH) + D3 (TC) + D4
  (Oddball) à FINALISER, VIP reste `[!]` définitif (reprise = décision utilisateur),
  ordre D6(D-R) -> D2 -> D3 -> D4 confirmé. CONTRAINTE inchangée : le rendu (calques
  objectifs) ne démarre qu'APRÈS fusion du lot A (mêmes fichiers) ; les phases Go
  attendent un créneau de build libre (jamais deux builds Go concurrents).
- [ ] D-R (demande utilisateur VALIDÉE le 25/08, en cours de pilotage) : rendu — la
  progression de capture ET l'appartenance directement SUR LA FORME de la base :
  remplissage progressif par découpe (`ctx.clip` sur `traceZonePath`) proportionnel à la
  jauge lue en escalier (schéma 18), en remplacement du seul arc externe (`drawGaugeArc`,
  `zoneStatesLayer.ts`) ; hiérarchie des encres à trancher à l'implémentation (progression
  de l'attaquant franche par-dessus la teinte du propriétaire affaiblie) ; évaluer le
  renforcement des alphas d'appartenance (`ZONE_HELD_FILL_ALPHA`). DONNÉES (corrigé
  25/08 après D0, vérifié sur pièces) : la série `gauge` est publiée sur les modes à
  zones SIMULTANÉES (Bastion — exactement le « mode base » de la demande) et absente en
  KOTH par construction (`zone_states_gauge.go:33-36`, tag 3 = compteur de transfert sur
  une colline) : D-R cible Bastion d'abord, Total Control après D3, jamais KOTH.
  L'appartenance sur forme existe déjà et reste. Séquencement : après fusion du lot A ;
  D-R (= D6 du plan fils) passe AVANT D2 (gain visible tôt, web seul — peut chevaucher
  un lot Go).

## Ordre et parallélisme

A, B0, C0, D0 en parallèle (worktrees et périmètres disjoints). B1 après validation B0 ;
C1 après validation C0 ; D1+ après validation D0 ET fusion de A. Fusions séquentielles
par le superviseur, CI JOB verte entre chaque (`gh run list --branch feat/v75`).

## Clôture

Par lot : CR vérifié SUR PIÈCES → fusion `--no-ff` → journal + registre (superviseur) →
push → CI JOB verte → suppression worktree + branche. Clôture générale : encadré Notion
mis à jour (barrer + « TRAITÉ jj/mm — fusion <sha> » par point, style existant),
thought_log, mémoire de session.

## Découvertes (hors périmètre — consigner ici, ne PAS traiter)

- **Lot A — sentinelle `last === 0` de la boucle de lecture** (`useReplayPlayback.ts`, reprise
  telle quelle de `ReplayCanvas`) : l'amorce de l'horloge est `let last = 0` puis
  `if (last === 0) last = ts`. Un horodatage `rAF` valant EXACTEMENT 0 ré-amorcerait l'horloge
  à chaque pas, et le rejeu n'avancerait plus. Inatteignable en navigateur (le premier `ts`
  est > 0), rencontré seulement en test — les tests servent donc des horodatages non nuls et
  le disent. NON TRAITÉ (règle 7). Correctif si un jour on y revient : `let last = -1`.
- **Lot A — `held` conflate deux états dans `zoneStatesLayer.paintZoneState`** : il est faux
  autant pour « personne ne tient » que pour « camp non situable » (aucune ligne « moi »).
  A6 a corrigé la conséquence VISIBLE (le grisé se décide sur `owner`, pas sur `held`), mais
  le drapeau décide encore du REMPLISSAGE : une zone tenue par un camp non situable n'est pas
  remplie. C'est le comportement voulu et documenté (« jamais une couleur devinée ») — noté
  parce que le nom `held` ment sur ce qu'il porte. NON TRAITÉ.
