# HANDOFF superviseur — reprise v7.5, journee 24-25/08

> Point d'entree pour la prochaine session de PILOTAGE. Meme regime : un agent par worktree
> FRERE cree DEPUIS LA RACINE en chemin absolu (`git -C &lt;racine&gt; worktree add
> /c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-&lt;nom&gt; -b wt/&lt;nom&gt; origin/feat/v75`),
> CR verifie SUR PIECES, fusion `--no-ff` par le superviseur, push superviseur, CI verte au
> niveau JOB. Lire aussi : `PLAN_SUPERVISION_2026-08-20.md`, `REGISTRE_REPORTS.md` (fin),
> `.ai/thought_log.md` (tete), memoire.

## 0. Etat exact

- `feat/v75` HEAD `a6ef4d139`, POUSSE, CI verte. **Schema artefact 18** (prochain libre 19 ;
  aucun bump aujourd'hui). Cache : 34/35 artefacts au schema 18 ; `51101d1d` non constructible.
- **L'ARBRE EST PARTAGE** avec plusieurs sessions de l'utilisateur (notion-cinq, lettres-bases,
  amis/presence, deps...). REGLE : ne committer QUE ses propres fichiers, PRESERVER tout etat
  non commite d'une session voisine (ex. au 25/08 : une ligne dependabot TypeScript dans
  `REGISTRE_REPORTS.md`, laissee non commitee expres). Conflits de docs : garder les deux cotes.
- Worktrees de CETTE session au handoff : `wt-jobmon` (agent actif) et `wt-cache` (agent actif).
  Tous les autres nettoyes.

## 1. Livre et fusionne les 24-25/08 (tout pousse, CI verte)

- **Bandeau de score** au-dessus du canvas (tous modes a equipes ; remplissage relatif au leader,
  aucun objectif de score n'existe dans les donnees).
- **Nettoyage UI** : killfeed retire de sous Dominance (fil du rejeu conserve) ; pions gris
  supprimes (vies sans identite non dessinees) ; murs disparaissent (~10 s, duree officielle car
  le film ne date aucun despawn) ; croix 2,5 s ; effets de mort orientes ON par defaut ; bascules
  persistees corrigees (l'ecriture localStorage se faisait EN PHASE DE RENDU React).
- **Grappin sonore** (`grapple_fire.wav` de l'archive Drive utilisateur, declenche par `grappleLines`).
- **KOTH repare + catalogue REGENERE** : 133 -&gt; 273 collines (29 cartes jamais re-parsees, 0 perdue),
  +302 zones Total Control (`totalcontrol_zone`, label SANS underscore), +73 objectifs Firefight ;
  garde-fou anti-rack durci (plancher d'emprise absolue seul, le ratio retire). Oasis x2 = absence
  LEGITIME (aucune paire officielle KOTH Arena, verifie sur les 850 paires 343 en base).
- **Ouvrier distant** durci (2 rondes adversariales, P0 de ronde 2 : garde anti-regression descendu
  au POINT D'ECRITURE `writeArtifactBytes`). Faits dans le job. ACTIVATION non faite (voir dettes).
- **Cuissons blindees** : un film = un processus, plafond memoire + sentinelle. Valide sur le crash reel.
- **Hygiene CI** : gitleaks re-vert + binaire local installe ; variant-probe supprime ; 2 tests
  repo-root migres ; registre repare.
- **A1 detection CLOSE sur mesures** : repulseur ET propulseur sans canal d'usage dans le film
  (5 voies chiffrees : i27, i54, i56, i51, i59 ; controle de validite tag 3 grappin a 20x).
- **clockOffsets** : etait DEJA resolu le 14/08 (le registre n'avait pas suivi) ; mesure du 25/08
  confirme (57,2 % rattachement vs temoin plat 13,4 %). Le bloquant containment = taux d'attribution
  40,9 % + absence d'oracle, jamais l'horloge.

## 2. EN VOL au handoff (fusionner a leur CR verifie)

- **`wt-jobmon`** (Sonnet) : erreur EXPLICITE dans le monitoring de job quand le film-bombe est
  isole (motif structure, pas un echec anonyme) + profiling `51101d1d` ajoute au backlog.
- **`wt-cache`** (Opus) : REMEDIATION du cache appauvri (mode de re-cuisson des artefacts a schema
  courant dont `scoreTimeline.players` est vide ALORS QUE les faits existent en base). Gate anti-ART
  `-tags=integration -p 1` obligatoire. A faire AVANT toute activation ouvrier.

## 3. EN ATTENTE UTILISATEUR (page Notion « Verdicts en attente & handoff », enfant du Backlog)

1. **Son de balise** : voie jeu EPUISEE. Solution = decoupes de la video Hectorlo (artefact
   « Decoupes video equipements », chapitre 43:40). Attend la designation d'une decoupe (T1-T6).
   Meme filon pour le Threat Seeker (44:15) et le Shroud (44:59, apres Theater).
2. **Theater `0x4396db42`** : seul equipement du corpus sans nom (104 occ / 9 matchs ; hypothese
   ecran de dissimulation, non prouvee). Artefact « Equipement 0x4396db42 » + fiche Notion (5 matchs
   avec carte + date/heure + timing + joueur). Une fois nomme -&gt; rendu + son.
3. **Repulseur-sur-kills** : oui/non (brancher son + effet sur les kills au repulseur, seul canal datable).
4. **Ghidra** : le connecteur MCP COTE CLAUDE CODE est fige (instance vue mais `connected:false` /
   nom `unknown` ; Ghidra tourne, ecoute 8089, socket frais). Reprise = redemarrer la session Claude
   Code (recharge le pont). Sert la voie deterministe (PostEvent) pour les sons introuvables.
5. Gate visuel : VALIDE le 25/08. Couverture des tirs (ouvrier) >= 88 % : VALIDEE par l'utilisateur.

## 4. Dettes avec condition de reprise (registre)

- **Profiling `51101d1d`** : pprof heap au pic ; suspect nomme par une session voisine
  (`NamedEventsFrom`/`incrementTimes`, ~26 Gio) ; correctif ou exclusion permanente.
- **Activation ouvrier prod** : critere couverture VALIDE ; reste la remediation cache (wt-cache) ;
  puis lever le garde `LocalOnlyReplay` (= deploiement, prevenir l'utilisateur).
- **Oracle de justesse containment** (Vagabond porte ses 3 zones reelles) : vraie porte, hors v7.5.
- **Choix de variante a egalite** (catalogue) : un lot qui ajoute un label DOIT re-tirer par le RESEAU
  (jamais re-parser le fichier deja nomme) ; departage a expliciter.
- Cuisson de masse (916 films) : apres stabilisation du contrat, outil blinde pret.

## 5. Lecons payees cette session (a NE PAS re-apprendre)

- **Jamais de verdict a travers un pipe** : `go test | grep | head` rend l'exit de `head` et tronque
  la suite par SIGPIPE (deux faux verts vecus, un agent + le superviseur). Rejouer NU, exit code reel.
- **Worktrees depuis la RACINE en absolu** : un `cwd` baladeur a cree `apps/LevelUp-wt-catalogue`
  (orphelin dans l'arbre). Toujours `git -C &lt;racine&gt; worktree add /c/.../LevelUp-wt-&lt;nom&gt;`.
- **Recette du handoff = piege** : `backfill-replay --only-existing` bouclait sur le corpus en UN
  processus (bombe RAM). Blinde depuis (un film = un process).
- La ronde 2 adversariale DOIT etre fraiche : elle a attrape le P0 que la ronde 1 avait manque.

## 6. Reprise en 4 lignes

1. Lire ce handoff + `REGISTRE_REPORTS.md` (fin) + tete de `thought_log` ; `git log --oneline -5`,
   `git status` (attention etats non commites voisins).
2. Traiter les CR de `wt-jobmon` et `wt-cache` (verifier sur pieces, fusionner, pousser).
3. Traiter les reponses utilisateur (decoupe balise -&gt; branchement ; Theater -&gt; nommage + rendu + son ;
   repulseur-sur-kills si oui).
4. CI au niveau job apres chaque push (`gh run list --branch feat/v75`).
