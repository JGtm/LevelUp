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
- [ ] A2 (pt 6) : retirer le compteur total d'éliminations en haut à droite du kill feed
  (ancre candidate : `ReplayCountersBadge`). Même règle 0 code mort.
- [ ] A3 (pt 8) : couper le son ne fait plus disparaître la barre de volume : le curseur
  passe à 0 ; ré-activer restaure le volume précédent (`ReplaySoundControls`).
- [ ] A4 (pt 9) : à la fin du rejeu, rester sur l'état FINAL (curseur à T_final, scène
  finale affichée) — pas de remise à zéro (`ReplayTransport`). Relancer reste possible.
- [ ] A5 (pt 3) : DIAGNOSTIC des « traits au bout du cône de visée » (rendu canvas —
  `ReplayCanvas` et calques). Le CR DOIT répondre « c'est quoi » avec fichier:ligne.
  Décision par défaut : artefact de rendu → corriger ; donnée réelle légitime (ex. trace
  de tir) → NE PAS supprimer, expliquer au CR et statuer `[~]`.
- [ ] A6 (pt 7) : zones/bases non prises (= neutres, aucune équipe propriétaire à
  l'instant t) : contour grisé (token sémantique, pas d'hex). Si l'état « non prise »
  n'est PAS dérivable du document schéma 18 → `[!]` + condition de reprise = lot D.

Gates (dans le worktree, exit codes réels) : `npm ci` (autorisé), typecheck
(`npx tsc -b` via `make check-types` ou équivalent local), `npx vitest run` ciblé
`match-replay`, lint web. i18n : toute string FR **et** EN. Pas de vérification
navigateur (gate visuel = utilisateur, à la fusion).

## Lot B — `wt/notif-rejeu` (Go, pt 5)

- [ ] B0 : mini-plan de l'exécuteur AVANT tout code, livré comme CR intermédiaire —
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

- [ ] C0 (MESURE, stop au verdict) : par carte NON validée, distribution des hauteurs
  des positions joueurs du corpus d'artefacts cuits (itération film par film — JAMAIS de
  balayage corpus en un processus, leçon bombe RAM) vs hauteur des éléments de géométrie
  rendus. Livrable : tableau par carte (h_max fréquentée, volumes de géométrie au-dessus,
  % de faces rognées), 2 cartes témoins avant/après. STOP — validation superviseur.
- [ ] C1 (si concluant) : coupe à h_max + marge 5 m, par étage/prisme (ne jamais rogner
  un niveau praticable sous un plafond), géométrie brute CONSERVÉE à côté de la filtrée,
  cartes validées EXCLUES du rognage. Tests Go du filtre. Gate visuel final = utilisateur
  (planche fournie par le superviseur à la fusion).

## Lot D — `wt/obj-etat` (pt 11)

- [ ] D0 (PLAN, stop au verdict) : écrire `PLAN_OBJECTIFS_ETAT_VIVANT_2026-08.md` :
  inventaire PAR MODE de l'état vivant manquant (acquis : KOTH colline désignée + jauge
  schéma 18, CTF `flagCarries` + rendu drapeau ; à inventorier : Strongholds/Total
  Control possession vivante, Extraction, Oddball (crâne `[!]` corpus), VIP → mesurer le
  substitut `managed-objective-object-reference`) ; pour chaque mode : canal mesurable,
  oracle + témoin négatif, seuil de gate ; UN SEUL bump schéma (19) pour tout le lot ;
  re-cuisson des témoins seulement. STOP — validation superviseur.
- [ ] D1+ : implémentation par mode après go, gate mesuré par phase. CONTRAINTE : le
  rendu (calques objectifs) ne démarre qu'APRÈS fusion du lot A (mêmes fichiers).

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

- (vide)
