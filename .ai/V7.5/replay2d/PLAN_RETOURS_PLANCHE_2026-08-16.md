# Plan — Retours de la planche de validation du 2026-08-16 (bilan utilisateur)

> Le bilan complet est reproduit en annexe. Ce plan le convertit en lots, avec les decisions
> produit TELLES QUE FORMULEES par l'utilisateur. Contrat `plan-execution` pour chaque lot.
> Branche `feat/v75` ; lots web sur worktrees freres `wt/*` (fusion par l'orchestrateur,
> procedure `FUSION_WT_2026-08-16.md`).

## Verdicts, en une ligne chacun

| item | verdict | ce qui en decoule |
|---|---|---|
| A1 marqueur/trainee/cone | VALIDE — « je veux EXACTEMENT ce style pour les points, la croix de mort et la trainee ; le cone un peu plus prononce » ; « je prefere ce rendu a l'actuel » | **routé vers le plan d'habillage de l'utilisateur** (`PLAN_HABILLAGE_REJEU_2D.md`, autre session) : les parametres exacts du schema sont dans `NOTE_STYLE_MARQUEURS_PLANCHE_2026-08-16.md` |
| A2 eclair de bouche | VALIDE + reglage : bascules « effets de tirs » (tous) et « effets de mort », (i) tooltip disant que la couverture des tirs peut ne pas etre totale | lot R1 |
| A3 effet de mort | VALIDE, **optionnel, desactive par defaut** | lot R1 |
| A4 grenades | A REVOIR : la nappe Dynamo n'etait pas previsualisee ; les 3 explosions sont **trop breves** | lot R1 (duree) + planche (apercu nappe) |
| A5 grappin | VALIDE | — |
| A6 callouts | VALIDE : police **trop grande** ; limiter le debordement au maximum | debordement = phase A callouts (en cours) ; police = lot R1 |
| A7 objectifs / fond | sans avis | — |
| B1 eclat de mort | VALIDE | — |
| B2 reapparition | VALIDE : eclat **plus lent** ; texte **« Reapparition dans X s »** | lot R1 |
| B3 vitalite | VALIDE | — |
| B4 armes | VALIDE | — |
| B5 grenades/capacite/munitions | VALIDE : **images** pour les grenades (pas de texte sauf le compteur) ; capacite absente/inconnue = **symbole special**, pas un caractere | lot R1 |
| B6 rendu actuel equipement | A REVOIR (remplace par B7) | deja fait (fusion) |
| B7 verre + or | VALIDE | fait |
| C1 fil des morts | VALIDE : pas de « assiste par » — **icone d'assistance + nom + « - N % »** | lot R1 |
| D0 regles sonores | VALIDE | — |
| D1 tirs par arme | A REVOIR : les sons d'armes extraits du jeu sont sur `feat/v75-sons-fusion` (session utilisateur) ; tir charge = Ravageur seul ?, continu = Rayon de Sentinelle | reponse ci-dessous + lot R2 (fusion) |
| D2 lancers | VALIDE | — |
| D3 explosions + melee | A REVOIR : frag OK ; **Dynamo = « Full » sans le debut (lancer)** ; spike/plasma **ecourtes** | lot R2 |
| D4 equipements | A REVOIR : **ecourtes** | lot R2 |
| E1 tiroir | VALIDE + reglage des effets a ajouter ; **panneau PAR-DESSUS** plutot que qui pousse | lot R1 |
| F1 refus | A REVOIR : **murs et capteurs = a creuser en priorite** ; **objectifs vivants** (crane, drapeau, noyau) ; tir charge facultatif ; grenades ambigues et eclair dirige = **abandonnes** ; **son au kill au repulseur** ; translocateur apres mur/capteur | lots R3 (ti=37) et R4 (ti=11) ; repulseur = lot R2 |

## Reponse a la question D1 (tirs charges / continus) — d'apres le tag `weap` et le corpus

- **Charge** (`behavior` = charge / spew-charge / charge-with-magazine, sous-bloc `charging`) :
  **Pistolet a plasma** (tir surcharge — le fichier existe dans ta bibliotheque : « Plasma
  Pistol - Charged Shot ») et **Ravageur** (mode 2 mesure par ta chaine sons). Il en manque
  donc UN a ta liste : le pistolet a plasma. `sword-charge` (epee) est une charge de MELEE,
  pas un tir.
- **Continu** (`prediction type` = continuous) : **Rayon de Sentinelle** seul dans le registre
  actuel — le corpus n'en porte AUCUN tir (mesure), le son est declare mais jamais entendu.
- **Ce qui manque n'est pas un fichier, c'est la DONNEE par tir** : la jauge de charge et la
  surchauffe sont lues aux paquets delta puis JETEES (`i30/i33`, `i32/i35`) ; tant qu'elles
  ne sont pas publiees, un son « charge » ne peut pas etre declenche. C'est le lot armes B
  (registre) — facultatif selon ton bilan, donc NON planifie ici.

## Lots

### R1 — Retours web (worktree `wt/retours-planche`, apres fusion de `wt/heatmap`) — CLOS le 2026-08-17 (da7baf122, fusionne 1eb25d5fd)

Perimetre FERME (chaque item = une case) :
- [x] R1.1 Tiroir en **overlay** (panneau par-dessus la carte, patron `AssetDrawer`/`FeedbackDrawer`
      du depot), plus « qui pousse ». Le canvas ne se retaille plus a l'ouverture.
- [x] R1.2 Bascules « Effets de tirs » (eclairs de bouche, tous les tirs — ON par defaut) et
      « Effets de mort » (tueur -> victime — **OFF par defaut**), persistees ; **(i) tooltip** :
      « La couverture des tirs peut ne pas etre totale : le film n'enregistre un tir que
      lorsqu'un degat est applique. » FR/EN.
- [x] R1.3 Explosions **plus longues** : `EXPLOSION_MS` 1 400 -> **2 400** (phases mises a
      l'echelle : flash 120, onde 650, braises/eclats/poussiere), et duree de retention alignee
      (`GRENADE_REST_HOLD_MS`). Rejouer le garde-rail e2e raster de l'explosion.
- [x] R1.4 Eclat de reapparition **plus lent** : 0,55 s -> **1,2 s** ; texte de la fiche morte
      « **Reapparition dans X s** » (FR) / « Respawn in X s » (EN), « Reapparition ? » si
      inconnu.
- [x] R1.5 Grenades en **images** sur la fiche (`static/grenades-assets/halo_infinite/{frag,
      plasma,dynamo,spike}_{dark,light}.png`, deja versionnes) : icone + compteur, type equipe
      souligne comme aujourd'hui, aucun texte de type. Capacite absente ou rang non resolu :
      un **glyphe SVG neutre** (pas un caractere), tooltip « capacite non identifiee (rang N) ».
- [x] R1.6 Fil des morts : supprimer « assiste par » ; afficher **l'icone d'assistance** (a
      trouver dans les assets du jeu — atlas kill feed / HUD ; a defaut un glyphe SVG neutre,
      jamais un caractere) + **nom de l'assistant** + « - N % » de participation quand la
      part est connue.
- [x] R1.7 Callouts : **taille de police reduite** (label des zones : au plus la taille du nom
      de joueur + 1 px ecran ; aujourd'hui « trop grande »).
- [x] R1.8 Legende de la carte de chaleur : inchangee (lot heatmap) — juste verifier qu'elle
      tient dans le panneau overlay.

Gates R1 : purge `.tmp`, typecheck, lint, vitest — exit 0 ; zero hex ; FR/EN ; e2e raster
explosion vert ; textes journal/registre au CR ; fusion par l'orchestrateur.

### R2 — Sons : durees par categorie, Dynamo, repulseur (worktree `wt/sons-duree`, base
### `feat/v75-sons-fusion` — la branche de l'utilisateur, a FUSIONNER d'abord dans `feat/v75`)

- [x] R2.0 (FAIT par avance-rapide de feat/v75 sur feat/v75-sons-fusion, 23:15) Verifier `feat/v75-sons-fusion` (gates annonces verts : 430 fichiers / 3 876
      tests) puis la fusionner dans `feat/v75` en `--no-ff` (elle est basee sur `10c1ff019`) —
      revue de fusion : les fichiers `replaySound.ts` / `replayAudio.ts` / `useReplaySound.ts`
      sont touches des deux cotes (filtres du 16/08 vs variations RANGED).
- [x] R2.1 (c7c18a102 · fcf1ae325 — plafond SOUND_CUT_MAX_S 4 s, duree = celle du fichier, garde-rail RIFF ; +2,82 Mo) **Durees par categorie** : armes 1,2 s (inchange) ; **explosions et equipements
      jusqu'a 4,0 s** (sources 1,8 a 4,8 s dans ta bibliotheque) ; lancers 1,2 s ; melee 1,2 s.
      Le lecteur coupe **a la duree du fichier** (plus de `SOUND_CUT_S` unique : par stem ou
      par categorie), enveloppe conservee. Poids : ~6 Mo de WAV en plus dans l'image —
      l'ecrire ; alternative OGG/Opus notee, non retenue (compatibilite decodeAudioData).
- [x] R2.2 (attaque mesuree +33,63 dB a 3,350 s, coupe [3,310 ; fin] = 1,527 s — le segment est le MEME son que « Explode », la nappe qui precede n est pas embarquee : le son part a l instant du kill) **Dynamo** : partir de « Dynamo Grenade - Full.wav » (4,84 s) et **couper le debut
      (le lancer)** : detecter l'attaque de l'explosion (energie par fenetre de 50 ms, saut
      > X dB, mesure ecrite), garder [attaque - 40 ms ; fin], borne 4 s.
- [x] R2.3 (frag 3,335 · plasma 4,000 · spike 3,951 · camo 3,416/1,903 · surbouclier 3,996/2,155 s ; la frag re-coupee aussi, a confirmer a l ecoute) Spike / Plasma / equipements : re-couper depuis les sources a la nouvelle borne
      (recette validee a l'octet : `-map_metadata -1 +bitexact`, PCM s16le 48 kHz stereo).
- [!] R2.4 REFUS MESURE (6dadf49ca) : aucune source de degat ne mene a killfeed-56 (0/473 lignes de labels.tsv) — fil d alerte + silence propre livres, son choisi (Repulser - Activate (On Object)) ; registre. **Repulseur** : un kill au repulseur est identifiable (vignette `killfeed-56`,
      `nom_jeu: repulsor`, source_tag `0302cad3` dans `jeu/index.json`) -> verifier que
      `killicon` le resout (regle par tag ou a ajouter dans `rules.tsv` avec sa
      justification), puis son `Repulser - Activate` (ta bibliotheque, plusieurs variantes :
      prendre « On Object » ou une des Var. — a trancher a l'ecoute) joue par la vignette,
      table `KILL_SPRITE_SOUND_STEMS`. Garde-rail assets etendu.
- [ ] R2.5 (a la prochaine mise a jour de l artefact) Planche : re-encoder les nouveaux sons pour l'artefact (a la fusion).

Gates R2 : garde-rail assets vert ; vitest ; taille du dossier `static/sounds` publiee ;
gate d'ECOUTE utilisateur.

### R3 — Identite des entites `ti=37` : mur de protection et capteur de menaces (principal, Opus)

Priorite utilisateur explicite (« a creuser »). Plan dedie a ecrire au lancement, sur les acquis
du 15/08 : positions ti=37 a 97,2 % dans l'emprise ; aucun des 4 champs mesures ne porte
d'identite ; piste = **record de CREATION** de l'entite (famille high-32 comme les armes au
sol, `keyframe_ground_weapons.go`), `equipment-creator` R(5) n'est ni slot ni index joueur ;
UI a reflechir plus tard (capteur = zone radar pulsee ; mur = segment pose).

### R4 — Objectifs vivants (`ti=11`) : crane, drapeau, noyau (principal, Opus, apres R3)

Symboliser le porteur et l'objet : `managed-objective-object-reference` en `i3` (avant le mur
de chainage), `interaction-filter` i4 polymorphe bloque la suite — livrable en deux temps
(Notion 15.2). Plan dedie au lancement.

### Abandonnes (decision utilisateur) : grenades ambigues (deja clos), eclair dirige sur les
### tirs (impossible : pas de victime dans les tirs). Facultatif : tir charge (lot armes B).

## Ordre d'execution

R2 (sons, base sons-fusion — peut partir maintenant sur son worktree) -> fusion `wt/heatmap`
-> R1 (web) -> fusion -> R3 (ti=37) -> R4 (ti=11). Le re-build de masse (fenetre ops) reste a
caler avec l'utilisateur ; il portera les schemas 7/8 et les bornes corrigees.

## Annexe — bilan utilisateur, verbatim

    BILAN — effets du rejeu 2D — 2026-08-16
    ## Sur la carte
    [VALIDÉ] A1 Marqueur de vie, traînée, cône de visée — Parfait. Je veux exactement ce style pour les points, la croix de mort et la trainee. L'icone de visee je veux celui là mais un peu plus prononcé, jsute un peu. Je prefere ce rendu à l'actuel
    [VALIDÉ] A2 Éclair de bouche — A ajouter dans les filtres, avec un (i) avec tooltip, pour chosiir d'afficher les effets de tirs, juste les fatals ou aucun ou les deux (je pense que deux boutons suffisent). Préciser dans le tooltip que la couverture peut ne pas être totale pour les tirs
    [VALIDÉ] A3 Effet de mort orienté tueur → victime — Optionnel, désactivé par défaut Parfait sinon
    [À REVOIR] A4 Lancers de grenade, vols et fin de vol — Manque l'effet dynamo, je ne le vois pas dans tes proposition des éclairs electrique, qui comme tu le dis est plus long. Sinon il faudra rallonger l'effet, là c'est trop bref pour les trois autres.
    [VALIDÉ] A5 Ligne du grappin — PArfait
    [VALIDÉ] A6 Zones de callout — Attention à la taille de police, trop grande actuellement. Limiter le débordement au maximum
    [sans avis] A7 Objectifs (placement) et fond de carte
    ## Les fiches joueur
    [VALIDÉ] B1 Éclat de mort sur la fiche — Parfait
    [VALIDÉ] B2 Éclat de réapparition et compteur — Plus lent l'éclat. Et pour le respawn on met un texte, "Réapparition dans Xs"
    [VALIDÉ] B3 Vitalité : bouclier et vie — PArfait
    [VALIDÉ] B4 Armes portées, arme en main, permutation — PArfait
    [VALIDÉ] B5 Grenades, capacité, munitions — Veiller à bien utiliser les images pour les grenades également, pas de texte sauf pour le compteur Si pas de capacité ou inconnu, juste mettre un symbole spécial qui n'est pas un caractere
    [À REVOIR] B6 Équipement actif — rendu ACTUEL (feat/v75)
    [VALIDÉ] B7 Équipement actif — VERRE et ENCADRÉ DORÉ (spec Notion)
    ## Le fil des morts
    [VALIDÉ] C1 Ligne du fil des morts — PAs besoin d'écrire "assisté par, on a une icone assist, on l'affiche avec le nom à côté du joueur et "- %" de partiticpation
    ## Les sons
    [VALIDÉ] D0 Règles de la piste sonore
    [À REVOIR] D1 Tirs par arme (et kills à l'arme) — Les sons extrait et packagés du jeu ont été récupérés et merge dans la branche donc attention. Pour les tirs chargés je crois n'avoir que le ravager actuellement, pour le tir continu j'ai le rayon de sentinelle. à toi de me dire s'il en manque
    [VALIDÉ] D2 Lancers de grenade
    [À REVOIR] D3 Explosions (kill à la grenade) et coup de mêlée fatal — Pour la grenade frag c'est ok. Pour la dynamo c'est bizarre, j'ai l'impression que le son "full" est mieux mais il faudrait enlever le début qui correspond au "throw". pour le reste le spike et plasma ça me parait écourté, tu utilises bien les éléments dans "...\Audio Library\GRENADE" ?
    [À REVOIR] D4 Équipements : activation et désactivation — Même remarque que plus haut, les sons me paraissent écourtés ici
    ## Réglages
    [VALIDÉ] E1 Tiroir de réglages et filtres de sons par catégorie — Comme dit plus haut, un réglage pour les effets à ajouter; Je vois plus un panneau par dessus, non ?
    ## Ce que le rejeu ne montre pas (encore), et pourquoi
    [À REVOIR] F1 Refus mesurés — J'aimerais les murs de protection et capteurs de menaces, on réflechira à l'UI plus tard, sujet à pousser davantage niveau investigation. Les objectifs le but c'est de les avoir aussi et de les symboliser aussi, surtout le crane d'oddball et le drapeau de CTF, ou le noyau de stockpile. Tir chargé facultatif. Grenades ambigues et eclairs dirigés, on peut drop. Répulseur utile de jouer le son quand on a un kill au repulseur. Translocateur ce serait pas mal, mais à voir une fois que le mur et le capteur sont mis en place.
