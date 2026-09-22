# HANDOFF — Decodeur de film, serie 5 (post-chantier) — etat au 2026-09-22 soir

> Pour la prochaine session (Claude ou humain). Le user est a 96 % de son quota hebdomadaire :
> ce document dit ce qui est FAIT, ce qui RESTE, dans quel ordre, avec les commandes et les pieges.
> Sources de verite : `.ai/PLAN_DECODEUR_FILM_2026-09-13.md` (sections « Post-chantier — lot 5.x »,
> §4 decouvertes, §5 journal), `.ai/thought_log.md` (entree du 2026-09-22), les notes
> `.ai/V7.5/film_re/NOTE_5_*.md`, la page Notion « Grammaire du film — serie 5 » (section 11).

## 1. Ou en sont les branches

| branche | tete | etat |
|---|---|---|
| `feat/recherche-decodeur-film` (integration de la serie) | `d39c3d3e2` | poussee, CI VERTE (coverage relancee : flake d annulation connu) |
| `feat/v75-serie5` | `fbce7b87c` = `2d6e67a20` (serie 5 + 5.24, CI verte) + merge de `origin/feat/v75` `0dc29713a` (sections transverses du user) | commit local dans le worktree `LevelUp-wt-v75-merge` ; push + CI en cours au moment de l ecriture (pre-push knip exige `node_modules` : `npm ci` dans `apps/web` du worktree) |
| `origin/feat/v75` | `0dc29713a` | EN RETARD de la serie 5 tant que le fast-forward ci-dessous n est pas fait |
| checkout principal `LevelUp-go-migration` (feat/v75 local) | `ee22aec27` | en retard ; le user fait `git pull` lui-meme — JAMAIS ecrire dans ce checkout depuis une session |

**Geste qui reste pour livrer la serie 5 dans feat/v75** (des que la CI de `feat/v75-serie5` est verte
au niveau job ; si le job Coverage est « cancelled », `gh run rerun <id> --failed`) :

```bash
cd /c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-v75-merge
git fetch origin feat/v75 && git merge-base --is-ancestor origin/feat/v75 HEAD && \
  git push origin feat/v75-serie5:feat/v75            # fast-forward ; refuse si feat/v75 a encore bouge -> re-merger origin/feat/v75, gates, push, CI, recommencer
git push origin --delete feat/v75-serie5              # hygiene des branches distantes
cd /c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-recherche-film && git worktree remove /c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-v75-merge
```

## 2. Ce qui est FAIT le 22/09 (tout fusionne dans l integration, CI verte)

- Lots 5.13 a 5.25 (grammaire de trame : trois classes de vue, branche vive, table de datums, image-cle
  et bloc de type 1 lus en entier, controle de corruption lu dans le film, type 8 = liste des joueurs,
  table anticipee des archetypes ; escalade `clamber` prouvee 9/9 dans Theater ; saut = action d unite
  0x16, PAS un champ replique, recherche ABANDONNEE par le user ; mesure 5.25 : le calque des etats perd
  35,4 % des ticks de joueur sur `bfecd02b`, 53,7 % sur `4f77afc1`). **Schema 68**,
  `grammar-2026-09-22.12`, `killsource-2026-09-22.2`.
- Lot 5.24 : `levelup backfill-killsource` a 3 ouvriers (x2,7), fichier d etat
  `data/global/admin_state/backfill_killsource_<slug>.json`, `--status`, reprise prouvee, arret propre.
- Fin de serie : journal, references d equivalence re-figees (20/20), gate corpus 19 sur base
  `6e86db356` ACCEPTE par le user comme nouvelle reference (12 « pertes » = le modele des sieges lus du
  lot 5.10 ; `abilityImpulses.unread -> 0` est un gain lu a l envers) ; prochaine base du gate =
  `d39c3d3e2` (ou la tete de feat/v75 apres fast-forward).
- Backfills locaux (serveur arrete, binaire de la fusion) : republication des 99 artefacts au schema 68
  (`backfill-replay --only-existing`, 0 erreur), `backfill-usage-summary`, `backfill-pad-tiers --force`
  (99), `backfill-flag-grabs-net` (rien a ecrire), **`backfill-killsource` du parc entier : 1 611 films
  en 57 min 57 s + credit 3 min 9 s, 0 erreur**. Serveur dev relance (air detache, checkout principal).
- Notion : page « Grammaire du film — serie 5 » completee (section 11).
- Worktrees des lots retires (jonctions d abord). Reste `LevelUp-wt-v75-merge` (cf. §1) et
  `LevelUp-wt-recherche-film` (integration, a garder tant que la serie n est pas dans feat/v75).

## 3. Ce qui RESTE, dans l ordre (GO du user deja donnes le 22/09, sous reserve de quota)

1. **Fast-forward de feat/v75** (§1). Puis mettre a jour la memoire de session et le plan (§5 du plan :
   « serie 5 dans feat/v75 »).
2. **Lot 5.26 — le quart de trames encore abandonne sur film dense** (agent Opus, un seul, arret ecrit).
   Cause nommee : les entites nees ET mortes entre deux images-cles (projectiles/effets probables :
   72,5 % vivent < 100 ms) ne sont declarees par rien de ce que le depot lit ; le decodeur ne lit aucun
   record NEW la ou elles naissent. DEUX pistes jamais ouvertes, une adresse chacune :
   (a) les types de paquet **6, 0xb, 0xc** ont un handler dans le jeu (`FUN_142988084` -> `session+0x114`,
   `FUN_1429882c8`, `FUN_1429875e4` ; D3 5.17) que le depot ne lit pas — lecon du bloc de type 1 :
   « saute par la pompe » ≠ « jamais lu » ; (b) **le tir d arme comme acte de naissance** : si l evenement
   de tete `action_weapon_fire` (type 36, deja lu par `fire_events`) porte l identifiant du projectile
   qu il cree, la naissance est dans un canal deja lu (5,5 % / 16,4 % des premiers rejets tombent dans un
   paquet a tir ; mesurer sur les identifiants rejetes de `TestTicks525`). Gate : `TestGate516` carte
   `snowbound` (paquets fermes a reste NUL, avant 3 940/30 387 ; rejets hors datum 16 129), oracle
   conserve, `replay-equiv -films bcb6d393` sans `-update`. Si aucune piste ne tient : s arreter, rendre
   les mesures, PAS de troisieme suspect (sept lots 5.15-5.21 ont deja echoue ainsi).
   Brief modele : `brief_lot_5_23_table_anticipee_des_naissances.md` (perimetre ferme + interdits).
3. **Lot docs de release** (agent Opus) : synchroniser le README FR sur la version EN mise a jour par le
   user (`8ee9cc760` sur feat/v75) ; completer `CHANGELOG.md`, `RELEASE_NOTES.md`, `docs/RELEASE_NOTES.md`
   et les notes de version in-app (`apps/web/src/features/help/`) avec ce qui a ete ajoute depuis
   `2ddef392c` — dont la serie 5 cote utilisateur : escalade lue, sprint lu, sieges et retraits de
   vehicules, camp de capture lu, rejeu plus complet sur les matchs denses, backfill killsource rapide.
   Publics differents : What s new = end-user, changelog = technique.
4. **Lot archivage `.ai`** : 65 documents a la racine ; trier vivant / clos, deplacer les clos vers
   `.ai/V7.5/` (index `README.md` du dossier a tenir). Le plan du decodeur reste VIVANT tant que 5.26
   n est pas statue.
5. **Sequence de release Notion** (page « Backlog LevelUp », section « Pour la v7.5 ») : cocher les
   backfills locaux faits ; le reste est prod/VPS (jeton stage, worker, `LEVELUP_REPLAY_PUBLIC`,
   migrations medailles, semis des amis, copie des BDD). **Au deploiement de feat/v75 en prod :
   `levelup backfill-pad-tiers --force` (regle de la session ajustements-supp) puis
   `backfill-killsource`, serveur arrete, PREVENIR le user avant toute operation VPS.**

## 4. Registre des decouvertes non traitees (a reprendre plus tard, pas des lots par defaut)

- Petites regressions vues au gate corpus : `deathsPaths.directScan.matched` -1/-2 sur 3 temoins,
  `shots.h/presents` -8 et -1 sur 2 temoins (pas une correction de modele : a instruire).
- `TestKillSourceFaitsDIsolementFilmReel` ROUGE sur films reels : liste `named_by` du 2026-09-08 perimee
  (D3 5.24, remede de 3 lignes) ; les tests `KILLSOURCE_FIXTURES` se SAUTENT en CI (D4 5.24).
- Trois anticipations du meme fait dans trois marches (`killsource preload`, `newMarchTimeline`,
  `TableAnticipee`) — regle 6, a centraliser (5.23).
- `equipment-deployed-component` est livre par le film (5.23) : le signal deploye/lache que l heuristique
  200 ms / 1,5 m devait remplacer (memoire user du 13/09).
- Historique de baseline du corps de delta non porte (14 valeurs fausses / 174 606, 5.16).
- Anomalie de la grille de temps du document (la barre Theater = horloge relative du film ; la formule
  `originMs + frame x 100 ms` est en retard de 1 a 2,5 s par endroits, D2 5.22).
- `ecs_table.tsv` : `i60` mesure complet mais `partiel` ; glose « son etat » de `ti=23 i0` fausse.
- Saut : deux pistes non lancees si le user rouvre — detecteur sur l IMPULSION de decollage (vitesse
  verticale constante pour tous les Spartans) a la place de la fenetre de hauteur (rate un saut sur 7) ;
  lecture des bits d action de la vue C aux decollages une fois les trames fermees.
- Mantling : `R(7)` et `R(2)` de la queue d `i54` sans etiquette ; l ecrivain de `etat+0x1294` non trouve.
- `TestTableJoueursCorpus` rouge sur ce poste (build `HI_1_5_1` absent du profil, D4 5.17).

## 5. Methode et pieges (a relire avant d agir)

- **Pilotage** : un lot = un agent Opus (`model: opus` explicite) = un worktree `LevelUp-wt-decfilm-<n>`
  (branche `feat/decfilm-<n>` depuis la tete d integration) ; brief ecrit dans le scratchpad, perimetre
  FERME, interdits explicites, arret ecrit ; surveillance des commits et des fichiers touches toutes les
  20 min ; CR challenge SUR PIECES ; fusion par script (modele `merge_5xx.sh` : merge --no-ff --no-commit,
  keep_both plan/thought_log, openapi check, gofmt/build/vet/tests PK/lint, commit) ; `-p 1
  -tags=integration` si persist/sync/migration touches.
- **Jonctions du cache de films** : poser par PowerShell `New-Item -ItemType Junction` sur
  `data/cache/film_chunks` et `film_manifests` -> `LevelUp-go-migration/data/cache/...` ; verifier
  ~1 612 entrees par `Get-ChildItem` (`ls` n en montre qu une) ; RETIRER par `(Get-Item).Delete()` dans
  un tour SEUL, verifier 0 reparse point ET le compte du cache principal, PUIS `git worktree remove`
  sans `--force` (incident du 16/09 : cache vide).
- **Deux lots paralleles qui montent `grammar.Rev`** : fusionner a UN rang (rangs strictement croissants
  et sans trou dans le godoc ET le golden), re-figer par leur porte dans l ordre : grammar rev
  (`LEVELUP_UPDATE_GRAMMAR_REV=1 ... -update-grammar-rev`), facts rev, types shapes, `GoldenAssembly
  -update`, `REPLAY_CONTRACT_UPDATE=1 ... DocumentShape -update`, `... ContractFixtures -update`.
- **Gate corpus** : `go run ./cmd/replay-corpus-gate --base=<REVISION GIT> --parc-root <principal>
  --source-root <integration> --json <out> --work-root <dir> --keep-work` (la base est une revision, pas
  un JSON) ; lit les faits en base (un temoin sans faits est ignore) ; « perte » = toute baisse par cle.
- **Re-figeage d equivalence** : `go run ./cmd/replay-equiv -repo-root <integration> -update` puis sans
  `-update` (BILAN attendu : 20 identiques).
- **CI** : le job Coverage est souvent « cancelled » (45 min) -> `gh run rerun <id> --failed` ; un push
  suivant annule le run precedent ; le ratchet himap du chemin du jeu ne se voit QU en CI (les gates
  locaux excluent `internal/himap`) ; test supprime = retrait de la baseline JSONL dans le meme commit.
- **Push depuis un worktree** : le pre-push knip exige `apps/web/node_modules` (`npm ci`) — sinon
  « failed to push some refs » muet.
- **Serveur dev** : arret par `Get-Process air,server | Stop-Process -Force` (air relance server.exe :
  tuer les deux, verifier `:8000`) ; relance `Start-Process air.exe -WorkingDirectory
  <principal>\apps\go-api -WindowStyle Hidden` puis verifier le port. Aucune commande de backfill ne
  tourne serveur allume (handle RW).
- **Doctrine du user** : la grammaire chez l ecrivain (Ghidra 127.0.0.1:8089 lecture seule) avant toute
  mesure ; jamais de conclusion negative (un maillon manquant = une adresse) ; jamais de largeur
  inventee ; etats du joueur A L INSTANT ; « remarque ≠ validation » ; au 2e lot sans baisse du gate,
  S ARRETER et demander ; le lien avec les objectifs du user doit etre VISIBLE dans chaque compte rendu.
