# HANDOFF superviseur — pipeline v7.5, rejeu 2D — 2026-08-20 (nuit)

> Ecrit par la session de pilotage a l'approche de sa fin de contexte. Point d'entree de la
> reprise du PILOTAGE (plan ecrit avant chaque lot, un agent Opus/Sonnet par worktree FRERE
> `../LevelUp-wt-<nom>`, CR verifie SUR PIECES, fusion --no-ff par le superviseur, journal +
> registre a la fusion, worktree supprime, push par le superviseur). Lire aussi :
> `HANDOFF_SUPERVISEUR_v75_2026-08-18.md` (regles payees, inchangees), `REGISTRE_REPORTS.md`
> (fin de fichier), `.ai/thought_log.md` (tete), memoire `project_v75_etat_courant`.

## 0. Etat exact

- `feat/v75` HEAD `e4757c723`, POUSSE, arbre propre hors 2 fichiers non suivis d'autres sessions
  (`.ai/AUDIT_V7.2.0_MAIN_2026-08-06.md`, `apps/go-api/internal/himap/sonde_bouillie_gamefiles_test.go`).
  AUCUN agent en vol, aucun worktree de cette session. Schema d'artefact courant : **17** ;
  contrat MatchView etendu (`mode_category`) ; contrat rejeu 37 champs.
- Temoins re-cuits au schema 17 : `000d5950` (295 poses, 0 pad — Fiesta), `01e1f945` (11 pads
  dont power-up + surbouclier lache), `64e8adfa`, `bcb6d393`, `530820e5`, `00162144`.
  Le PARC (~950 artefacts) reste aux vieux schemas : re-cuisson de masse REPORTEE (decision
  utilisateur 18/08) — la relancer = `backfill-replay --only-existing` + `backfill-killsource`,
  serveur local arrete.

## 1. Livre les 19-20/08 (tout fusionne, pousse, journalise — SHA au journal)

Cycle socles COMPLET : detection film (schema 11) -> power-ups (17) -> rendu web -> catalogue
statique `map_weapon_pads.json` (72 cartes / 1 454 emplacements, `cmd/mapopads-build`) ->
croisement « allumes seulement » cote reponse (`mapWeaponPads`, decision utilisateur, Fiesta = 0
en assertion). Drapeau CTF : portages publies (14/15) + rendu (glyphe porteur / lache / base) ;
lancer REFUTE ; 26 naissances sans porteur = condition de reprise de `flagObjects`. Laches hors
Fiesta (equipements + power-ups, anneau pointille, garde de mode par `mode_category` — trou
0,7 % FERME). Sons : 41 egalises, explosions a chaque fin de vol, champ de reparation extrait du
jeu ; balise du translocateur : la banque ecoutee etait celle de l'APPAREIL — page v2 (45 sons
des BONNES banques + 5 candidats mixes) publiee, EN ATTENTE d'ecoute. Sonde variants : voie API
FERMEE (l'activation des socles vit dans le jeu installe). Fix bloquant `deathProgressions`
(19-22 Go). Retours planche R2 + R3 integralement rendus.

## 2. EN ATTENTE DE L'UTILISATEUR (ne rien relancer sans lui)

1. **Ecoute balise v2** : `https://claude.ai/code/artifact/2274faa6-f5b7-4c75-a84d-6c93c1e305bb`
   — sa designation -> une ligne dans `EQUIPMENT_PLACEMENT_SOUND_STEMS` ; si les 44 refuses :
   balayage des 1 305 `sbnk` de `pc/globals` (20-30 min, registre).
2. **Planche (NOUVELLE URL, changement de compte)** :
   `https://claude.ai/code/artifact/8d0e34bf-66fe-4467-b2ac-629bff74f8df` — verdicts R3-1 (croix
   2,5 s ou 4 s) et R3-2 (ecran de dissimulation : semi-transparent ou pions au-dessus).
   ATTENTION : la planche du scratchpad date du 18/08 21h57 — 2 lignes fausses depuis le schema
   17 (notees au registre) ; la reconstruire a la prochaine fournee (recette `assemble.cjs`).
3. **Gate visuel en app** sur les temoins re-cuits : equipements sur `000d5950`, power-up +
   surbouclier lache sur `01e1f945`, socles croises (position spawner), fil des morts qui
   defile, drapeau CTF sur `64e8adfa`.
4. **Heatmap « morts subies »** : question posee (A8), sans reponse.
5. **Score au-dessus du replay, tous modes** : demande produit au registre — le chantier score
   appartient a la session registre-film : COORDONNER avant tout lot.
6. Re-cuisson de masse : reportee (voir §0).

## 3. FILE DONNEES (plans a ecrire courts, un agent par lot, dans cet ordre propose)

1. **ARMES lachees au sol** : aucun canal ne dit « telle arme au sol en (x,y) depuis telle
   image » (`equipmentPlacements` = eqip, `weaponPads` = socles) — la decision produit du 18/08
   couvre aussi les armes de puissance lachees. Donnees puis web (`placementDropped.ts` accueille
   les familles telles quelles).
2. **KOTH colline par periode** (plan objectifs vivants phase 2) : formes du `.mvar` non
   attribuees + grappe des positions aux instants de score ; PREREQUIS : role `hill` absent du
   catalogue (`objective_roles.toml`, `mapvar.Role`) ; `instance_id` = 0 partout (registre).
3. **26 naissances de drapeau sans porteur** -> `flagObjects` (instrument `DrapeauObjetControle`
   en place ; piste « fenetre 2 s en production » au registre).
4. **Nommer via le jeu installe (himap)** : les 3 `type_id` de socle (`0x5F379533`, `0x6253CFC0`,
   `0x5E86D110`), `0x4396db42` (rang 10, negatif triple — dictionnaire epuise), et
   `parcel_mp_weapon_manager.lua` (activation des socles par mode, voie deterministe).
5. Crane d'Oddball (correlation temps-de-portage / score) ; translocateur etat actif (registre).

## 4. Pieges recents (en plus des regles du handoff du 18/08)

- Un agent a cree son worktree DANS l'arbre (`apps/LevelUp-wt-...`) : imposer le chemin
  `C:\Users\Guillaume\Projects\LevelUp-wt-<nom>` dans le brief et verifier au retour.
- FAUX VERT de pipe : `npm run typecheck | tail` masque l'exit — toujours `tsc -b --force` nu.
- Un `git add -A` d'agent (repris) ; messages de commit avec ligne parasite (filter-branch AVANT push).
- Les artefacts du principal peuvent etre perimes pour les gates visuels : re-cuire les temoins
  (script du 19/08 au scratchpad : `recuire.sh`, chemins node en `C:/`).
- Agents coupes par 500/529/limite : relancer par SendMessage avec l'etat constate ; commit par
  etape ; si le contexte de l'agent est gros, agent FRAIS avec brief complet.

## 5. Reprise en 4 lignes

1. Lire ce handoff + memoire + tete de journal ; verifier `git log --oneline -3` et `git status`.
2. Traiter les reponses utilisateur arrivees (ecoute balise -> branchement ; verdicts R3 -> lots).
3. Derouler la file donnees §3 (plan court -> agent -> CR sur pieces -> fusion -> journal -> push).
4. CI au niveau job apres chaque push (`gh run list --branch feat/v75`).
