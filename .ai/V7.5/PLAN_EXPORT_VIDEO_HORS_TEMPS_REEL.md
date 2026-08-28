# PLAN — Export video du rejeu HORS TEMPS REEL (image + son + surimpressions)

> Cree le 2026-08-28. Branche : `feat/v75`. Chantier successeur de
> `PLAN_CAPTURE_EXPORT_REJEU.md` (capture temps reel, livree le 2026-08-26).
> Statut : REDIGE, en attente de validation utilisateur avant execution.

## Le probleme, tel que l'utilisateur l'a pose

L'enregistrement actuel filme l'ecran : il faut jouer le match en entier pour obtenir le
fichier. L'utilisateur veut un EXPORT — on demande le clip, la machine calcule aussi vite
qu'elle peut, le fichier tombe. Avec le SON, et avec les SURIMPRESSIONS qui coiffent la
toile.

**Critere de succes** : sur un match reel, l'utilisateur choisit une plage, clique
« Exporter », et obtient un fichier `.mp4` lisible dans n'importe quel lecteur, dont la
DUREE egale la plage demandee, dont l'image montre le terrain et les surimpressions, dont
le son tombe sur les bons instants — et dont le calcul a pris SENSIBLEMENT MOINS de temps
que la duree exportee. Les gates techniques (E0 a E5) sont verts, et la recette visuelle
et sonore est prononcee par l'utilisateur, jamais par un agent.

**Effort** : LOURD. Six etapes, une dependance npm nouvelle, deux API navigateur qui
n'ont aucun precedent dans le depot (WebCodecs, `OfflineAudioContext`). E1 porte un risque
bloquant qui doit etre leve avant d'engager le reste.

**Contrat d'execution** : ce plan s'execute sous le skill `plan-execution` — ordre strict,
aucune etape differee, statut sur chaque item a la cloture. Les regles d'execution
propres a ce plan sont en fin de document (« Regles d'execution »), et elles priment sur
toute envie d'avancer plus vite.

## Perimetre — tranche avec l'utilisateur le 2026-08-28, ne pas rouvrir

### Ce que l'export contient

1. **Le terrain**, c'est-a-dire tout ce que `draw()` peint deja dans la toile : fond de
   carte, calques statiques, poses, marques, effets, aux bascules de calques ACTIVES au
   moment de l'export.
2. **L'ecran de fin de match** (victoire / defaite / egalite), repeint en canvas.
3. **Le message inter-manche** (« Manche N terminee »), repeint en canvas.
4. **Le son** : la piste complete du rejeu, mixee hors ligne.

### Ce que l'export NE contient PAS (exclusions explicites de l'utilisateur)

- Le bandeau de score au-dessus de la toile (`ReplayScoreBanner`).
- Le fil des eliminations (`ReplayKillFeed`).
- Les fiches de joueurs (`ReplayTeams`).

Ces trois blocs sont du DOM hors toile et le restent. Le score FINAL apparait quand meme
dans le clip : il fait partie de l'ecran de fin de match, qui le porte deja.

## Decisions techniques — TRANCHEES

- **D1. WebCodecs, pas MediaRecorder.** `VideoEncoder` (H.264) + `AudioEncoder` (AAC),
  muxes en MP4 par la dependance `mp4-muxer`. C'est le seul chemin ou NOUS posons les
  horodatages : `MediaRecorder` horodate sur l'horloge murale, donc tout rendu plus rapide
  que le temps reel en sortirait accelere.
- **D2. Definition = celle de la toile a l'ecran** (backing store, deja a l'echelle DPR),
  arrondie au PAIR sur les deux axes — H.264 refuse les dimensions impaires. Pas de choix
  de definition en v1 (arbitrage utilisateur).
- **D3. Plage reglable** : deux bornes en temps de match, defaut = la fenetre de gameplay
  entiere (`replayWindow.ts`). C'est le seul reglage de plage ; pas de vitesse, pas de
  definition.
- **D4. Cadence de sortie FIXE a 30 im/s**, horodatages explicites. La duree du clip est
  donc la duree de MATCH exportee, quelle que soit la vitesse de lecture affichee a
  l'ecran pendant l'export. C'est la rupture nette avec la decision 4 du plan precedent
  (« on filme l'ecran ») : ici on RECALCULE le film.
- **D5. Le bouton d'enregistrement temps reel devient un REPLI.** Il ne se rend plus que
  sur un navigateur sans `VideoEncoder`. Deux boutons qui font presque la meme chose
  seraient un piege a clic ; et supprimer le repli couperait Firefox/Safari anciens.
- **D6. Son inclus par defaut**, case a decocher dans le dialogue. Le mixage suit les
  reglages courants de la page (categories actives, volume, chaine de distance) et ne
  depend PAS de l'etat coupe/actif des haut-parleurs : on exporte un clip sonore meme si
  on regarde en silence.
- **D7. Variantes sonores deterministes.** Le tirage des variantes (`replaySoundVariants`)
  passe par un generateur a GRAINE derivee de l'evenement (index + stem) pour l'export :
  deux exports du meme match sonnent pareil. Le chemin temps reel garde son tirage libre.
- **D8. Progression et annulation.** Le dialogue affiche « image N / total » et une barre ;
  « Annuler » ferme l'encodeur sans deposer de fichier. Un export ne bloque jamais l'onglet
  (la boucle rend la main a l'evenementiel entre les lots d'images).
- **D9. Nom du fichier** : `rejeu-<matchId>-<debut>-<fin>.mp4` en temps de match
  (`4m10s-6m30s`), par extension de `buildCaptureFilename`. Repli horodate inchange.

## Etat des lieux — mesure le 2026-08-28, RE-VERIFIER sur pieces avant de coder

Ce qui rend le chantier possible, et qu'il ne faut surtout pas casser :

- **Le dessin est une fonction PURE de l'image courante.** `draw()`
  (`ReplayCanvas.tsx:325`) ne lit que `frameRef.current` (l. 340) ; aucun `performance.now()`,
  aucun etat accumule. Tous les effets sont precalcules une fois par document
  (`useReplayFx.ts`). Poser `frameRef.current = f` puis appeler `draw()` peint donc
  EXACTEMENT l'image f, aussi vite qu'on le demande.
- **La piste sonore est deja une liste triee d'instants** : `buildSoundTimeline()`
  (`replaySound.ts:556`), consommee par un curseur (`replaySoundCursor.ts`). Le son de fin
  de manche y est ; SEULE la fanfare de fin de match en est absente (declenchee par
  `onEnded`, `useReplaySound.ts:429`) — elle se re-derive de la borne de fin.
- **L'enveloppe de lecture est une fonction pure** : `soundEnvelope()`
  (`replayAudio.ts:99`), plafond de voix `SOUND_MAX_VOICES`.
- **Les deux surimpressions ont leur logique deja separee du rendu** :
  `victoryLogic.ts` (`readVictory`), `roundsLogic.ts` (`roundTransitions`,
  `activeRoundTransition`), `scoreBannerLogic.ts` (`readScoreBanner`), et leur habillage
  partage dans `replayOverlayStyles.ts`. Les peintres canvas consommeront CES modules —
  pas une deuxieme lecture des donnees.
- **Le telechargement est partage** : `triggerDownload()` (`replayCapture.ts:71`).

## Architecture cible — modules

| Module | Nature | Role |
|---|---|---|
| `replayVideoEncoder.ts` | pur + WebCodecs | Choix du codec, config, ouverture/drainage/fermeture de l'encodeur, muxage MP4 |
| `replayExportPlan.ts` | pur | Des deux bornes et de la cadence : la liste des images a rendre et leurs horodatages |
| `overlayPaint.ts` | pur (contexte 2D) | Le voile, le bloc de statut, le filigrane teinte, les lignes de texte |
| `replayAudioMix.ts` | Web Audio | Piste -> `OfflineAudioContext` -> `AudioBuffer` -> `AudioEncoder` |
| `useReplayExport.ts` | couture React | Boucle, progression, annulation, assemblage, telechargement |
| `ReplayExportDialog.tsx` | UI | Bornes, case son, progression, annuler |

## Etapes — ordre strict, une a la fois (contrat `plan-execution`)

### E0 — Le filet du telechargement (prealable, corrige un bug existant)

`triggerDownload` revoque l'URL d'objet SYNCHRONEMENT apres `a.click()` et n'attache
jamais l'ancre au document (`replayCapture.ts:71-82`). Sur un blob video de plusieurs
dizaines de Mo, le telechargement peut etre annule avant d'avoir demarre — cause
plausible du « l'enregistrement video ne marche pas » signale par l'utilisateur.

Items :

- [x] `triggerDownload` attache l'ancre au document avant `click()`, la retire apres
- [x] La revocation de l'URL passe au tick suivant (pas dans le `finally` synchrone) —
      retenue 1 s (`URL_RELEASE_MS`), pas 0 : le tick suffirait a sortir du clic mais pas a
      garantir que la lecture du blob est engagee sur une machine chargee
- [x] Test : la revocation n'a PAS eu lieu au retour de `triggerDownload`
- [x] L'en-tete du module dit POURQUOI (le blob video annule, pas une preference de style)

**Gate** : `cd apps/web && npx vitest run src/features/match-replay/replayCapture` vert,
puis recette utilisateur : l'enregistrement temps reel actuel depose un fichier lisible.

**Etat 2026-08-28** : gate technique VERT — `npx vitest run
src/features/match-replay/replayCapture src/features/match-replay/useReplayCapture`,
2 fichiers / 27 tests passes. Le test qui exigeait la revocation SYNCHRONE a ete retourne
dans le meme commit (il verrouillait le bug). **Recette utilisateur EN ATTENTE** : elle
demande le navigateur, aucun agent ne peut la prononcer.

### E1 — L'encodeur video, et rien d'autre

Dependance `mp4-muxer`. `replayVideoEncoder.ts` : detection de capacite
(`VideoEncoder.isConfigSupported`), config (avc1, 30 im/s, debit derive de la surface),
image-cle toutes les 60 images, drainage sur `encodeQueueSize` (contre-pression), sortie
`Blob` MP4.

**Risque a lever DANS CETTE ETAPE** : la toile est-elle « teintee » (tainted) par le fond
de carte ou les emblemes ? Un canvas teinte fait lever `new VideoFrame(canvas)`. A
verifier sur un match reel AVANT d'ecrire la suite ; si teinte, servir les assets en CORS.

Items :

- [x] `mp4-muxer` ajoute aux dependances de `apps/web` (`^5.2.2`)
- [x] `canExportVideo()` : capacite du navigateur, pure, `false` en jsdom
- [x] `videoEncoderConfig(width, height)` : dimensions arrondies au pair, debit derive —
      le NIVEAU H.264 est calcule depuis la surface et la cadence (`avcLevelFor`), pas
      choisi au juge
- [x] Ouverture / encodage / drainage / fermeture, avec contre-pression sur `encodeQueueSize`
- [x] RISQUE TOILE TEINTEE leve et CONSIGNE dans « Decouvertes » (D-3) : verdict NEGATIF,
      aucune source d'image du canvas n'est distante

**Gate** : `cd apps/web && npx vitest run src/features/match-replay/replayVideoEncoder`
vert, `make check-types` vert, et un export manuel de 30 images d'une toile de test qui
produit un MP4 que le lecteur systeme ouvre. Si la toile est teintee : E1 ne se clot pas
avant que la cause soit corrigee (assets en CORS) — aucune etape suivante ne demarre.

**Etat 2026-08-28 : E1 CLOSE.** Gate vert — `vitest run replayVideoEncoder eventLoopYield
replayCapture` : 3 fichiers / 31 tests ; `make check-types` vert ; eslint sans erreur sur
les trois modules. Export manuel dans le navigateur : MP4 valide (`ftypisom`), dimensions
impaires 641x361 ramenees a 640x360, codec `avc1.64001e` calcule. **Mesure de vitesse** :
300 images en 1280x720 encodees en 0,54 s, soit **18,6x le temps reel**, onglet cache. Un
module non prevu par le plan a du etre cree en cours d'etape (`eventLoopYield.ts`, cf. D-1)
— sans lui, l'export etait DIX FOIS PLUS LENT que le temps reel.

### E2 — Les peintres de surimpression

`overlayPaint.ts` : voile (`bg-background/70` -> `fillRect` a l'alpha equivalent), bloc de
statut (rectangle arrondi, bord 2 px, typo du verdict), filigrane du logo TEINTE (image
consommee en masque : `source-in` sur une toile hors ecran), nom d'equipe, ligne de score
final ; version neutre pour l'egalite et pour le message inter-manche.

Les couleurs se resolvent depuis les tokens du theme (`getComputedStyle` sur la racine),
jamais en dur — regle CLAUDE.md n°12. Les polices : attendre `document.fonts.ready` avant
la premiere image.

Items :

- [x] `paintVictoryOverlay(ctx, reading, score, ...)` — panneau d'equipe ET panneau neutre.
      ECART ASSUME : un seul point d'entree `paintOverlayPanel` sert les DEUX panneaux et le
      message inter-manche, le style du bloc etant un parametre. C'est la transposition
      exacte du DOM, ou les trois partagent deja `replayOverlayStyles.ts` — deux peintres
      auraient recopie la meme carte.
- [~] `paintRoundBreak(ctx, endedIndex, ...)` — version neutre, meme bloc de statut : couvert
      par `paintOverlayPanel` + `neutralStatusStyle`, avec `veil: false` (teste)
- [~] Le filigrane du logo consomme l'image en MASQUE (`source-in`), jamais en aplat : le
      peintre recoit l'image DEJA teintee, et le helper qui teinte existe deja dans le depot
      (`tintedIconCanvas`, employe par les socles d'arme et les vignettes de grenade). Rien a
      ecrire ; le branchement est un item de E3.
- [x] Logo absent ou non charge : pas de filigrane, le panneau reste entier (parite DOM)
- [x] Couleurs resolues depuis les tokens du theme, aucune valeur hex dans le module — les
      encres arrivent en parametre (`readInk` / `resolveToken` cote appelant). Seule couleur
      ecrite en clair : le noir de l'OMBRE PORTEE, qui n'est pas une couleur de theme
      (precedent de la meme feature : `calloutsLayer.ts`, `zoneStatesLayer.ts`)
- [~] `document.fonts.ready` attendu avant la premiere image : le peintre ne charge aucune
      police, c'est un prealable de la BOUCLE — porte par l'item « Prealables asynchrones »
      de E3, ou il a ete ajoute

**Gate** : `cd apps/web && npx vitest run src/features/match-replay/overlayPaint` vert,
`cd apps/web && npx eslint src/features/match-replay/overlayPaint.ts` sans erreur, puis
recette visuelle utilisateur (l'image exportee ressemble a ce que la page affiche).

**Etat 2026-08-28 : E2 CLOSE cote technique.** Gate vert — 12 tests sur contexte espionne
(ordre fond-vers-texte, capitales, tiret dans l'encre attenuee, texte JAMAIS dans la
couleur de camp, allie a gauche, filigrane sous le texte a 20 %, pas de voile sur
l'inter-manche, rien peint sans statut) ; `make check-types` vert ; eslint sans erreur.
**Recette visuelle utilisateur EN ATTENTE** — elle ne peut se prononcer que sur un export
reel, donc apres E3.

### E3 — Le pilote d'export

`replayExportPlan.ts` (pur) : des bornes et de la cadence, la liste des images.
`useReplayExport.ts` : pour chaque image — poser `frameRef.current`, appeler `draw()`,
peindre la surimpression active, encoder. Par lots, avec une remise de main a
l'evenementiel entre les lots (JAMAIS `requestAnimationFrame` : il est bride en onglet
d'arriere-plan et l'export s'arreterait des que l'utilisateur change d'onglet).

Prealables verifies avant de lancer : calques statiques cuits (chaleur, sol, zones) et
icones de grenade chargees — ils arrivent en asynchrone, et un export lance trop tot
peindrait des images incompletes.

A la fin, quoi qu'il arrive : `draw()` sur l'image d'avant l'export, pour rendre l'ecran
a l'utilisateur tel qu'il l'avait laisse.

Items :

- [x] `buildExportPlan(bornes, doc, fps)` : liste d'images + horodatages, pure et testee.
      Les images sont FRACTIONNAIRES (`draw()` interpole) : arrondir saccaderait un rejeu
      fluide a l'ecran. La derniere image tombe EXACTEMENT sur la borne, sans quoi le clip
      s'arreterait juste avant l'ecran de fin de match, qui ne se peint qu'a cette borne.
- [x] Boucle par lots, remise de main par `yieldToEvents()` (`eventLoopYield.ts`) entre les
      lots — NI `requestAnimationFrame` NI `setTimeout` : les deux sont brides en onglet
      cache (mesure en « Decouvertes », D-1)
- [x] Progression publiee (image N / total) et annulation honoree entre deux lots
- [~] Prealables asynchrones : `document.fonts.ready` FAIT, logo d'equipe teinte par
      `tintedIconCanvas` FAIT. **Calques statiques cuits et icones chargees : NON traite**,
      voir D-5 — les drapeaux de disponibilite vivent dans quatre hooks distincts et les
      remonter couterait plus de lignes de canvas que tout le cablage de l'export, pour un
      cas inatteignable en pratique (la commande d'export vit dans la barre de lecture, qui
      ne se rend qu'apres le montage du canvas).
- [x] Restauration de l'image d'avant l'export en sortie, y compris sur annulation/erreur
      (bloc `finally`, teste)
- [x] Le cliquet de taille de `ReplayCanvas.tsx` n'est pas franchi — 679/679. Paye par une
      extraction reelle (`hoverLayers.ts`, -9 lignes) et par le branchement de l'export DANS
      `useReplayCapture`, qui est deja « ce qui sort du rejeu ».

**Gate** : `cd apps/web && npx vitest run src/features/match-replay/replayExportPlan
src/features/match-replay/useReplayExport` vert, `make check-types` vert, puis export
reel MUET d'un match court, verifie a l'oeil par l'utilisateur.

**Etat 2026-08-28 : E3 CLOSE cote technique.** 23 tests (15 sur le plan d'echantillonnage,
8 sur la couture) ; suite complete de la feature verte : 112 fichiers / 1692 tests, cliquet
de taille inclus ; `make check-types` vert ; eslint propre sur le perimetre.
**L'EXPORT REEL NE PEUT PAS ENCORE ETRE DECLENCHE** : le plan place sa recette visuelle en
E3 alors que le bouton qui la rend possible est en E5 (cf. D-4). La recette d'E2 et celle
d'E3 se prononceront donc ensemble, apres E5.

### E4 — Le mixage sonore hors ligne

`replayAudioMix.ts` : piste `buildSoundTimeline` filtree a la plage, plus la fanfare de
fin si la borne de fin est dans la plage ; decodage des assets ; `OfflineAudioContext` ;
chaque evenement programme a `(ms - debut)/1000` avec son enveloppe et sa variante a
graine (D7) ; plafond de voix applique en pur (comptage des voix vivantes a chaque
instant, meme regle que le temps reel) ; `startRendering()` ; encodage AAC ; muxage.

Items :

- [ ] `planAudioMix(timeline, bornes, endMatch)` : evenements retenus, pur et teste
- [ ] Plafond de voix applique en pur, MEME regle que le temps reel (`SOUND_MAX_VOICES`)
- [ ] Enveloppe par `soundEnvelope()` reutilisee, jamais recopiee
- [ ] Tirage des variantes a graine (D7), teste : deux appels donnent le meme resultat
- [ ] Asset absent : silence memorise, aucun rejet de l'export entier
- [ ] Piste AAC encodee et muxee avec la video ; case son decochee = MP4 sans piste audio

**Gate** : `cd apps/web && npx vitest run src/features/match-replay/replayAudioMix` vert,
`make test-web` vert, puis recette d'ECOUTE utilisateur (les sons tombent sur les bons
instants, aucun mur de bruit sur un echange nourri).

### E5 — L'UI, l'i18n et le repli

Bouton « Exporter la video » dans `ReplayTransport` (a la place du bouton d'enregistrement
quand WebCodecs est la, D5), dialogue de plage + case son + progression + annuler.
Chaines FR et EN par typage (`Record<Locale, T>`), regle CLAUDE.md n°1.

Items :

- [ ] Bouton « Exporter la video » dans `ReplayTransport`, icone SVG inline `currentColor`
- [ ] Le bouton d'enregistrement temps reel ne se rend QUE sans `VideoEncoder` (D5)
- [ ] Dialogue : deux bornes, case son, barre de progression, bouton Annuler
- [ ] Bornes bornees a la fenetre de gameplay, fin toujours posterieure au debut
- [ ] Toutes les chaines en FR ET EN dans `i18n.ts`, parite tenue par le typage
- [ ] Aucune valeur hex ni classe Tailwind couleur (regle n°12)

**Gate** : `make check-types`, `make test-web`, `cd apps/web && npx eslint
src/features/match-replay`, et le contrat i18n (`i18nContract.ts`) vert.

### E6 — Cloture

Items :

- [ ] Entree `.ai/thought_log.md` (date, titre, statut, decision, resultats, suite)
- [ ] `.ai/V7.5/README.md` indexe ce plan
- [ ] Chaque item de chaque etape porte un statut `[x]` / `[~]` / `[!]` — aucune case vide
- [ ] Section « Decouvertes » remplie (ou explicitement vide)
- [ ] Skill `delivery-checklist` passe avant de proposer le commit

**Gate** : `make gate-push` vert, puis recette visuelle ET sonore prononcee par
l'utilisateur — c'est lui qui a le navigateur, jamais un agent.

## Regles d'execution — elles priment sur l'envie d'avancer

1. **Ordre strict.** L'etape N+1 ne commence pas avant que N soit CLOSE : tous ses items
   statues, son gate execute et vert. « Je reviendrai sur cet item » n'existe pas.
2. **Statuts** : `[x]` fait · `[~]` couvert ailleurs (avec la reference) · `[!]` non
   traite (avec la justification ecrite). Aucune case vide a la cloture du plan.
3. **Zero fix hors perimetre.** Toute anomalie croisee en chemin se consigne dans
   « Decouvertes » et ne se corrige pas — sauf si elle BLOQUE l'etape en cours, auquel cas
   elle devient un item de cette etape, ecrit avant d'etre traite.
4. **Verification sur pieces** avant de coder chaque etape ET avant de cocher : rouvrir le
   fichier et la ligne visee. Les numeros de ligne de ce plan datent du 2026-08-28 et le
   code bouge — ils indiquent ou chercher, ils ne dispensent pas de regarder.
5. **Reprise de session** : l'avancement se lit dans les cases de ce fichier, et nulle
   part ailleurs. Reprendre = premiere case non cochee, dans l'ordre. Le contexte de la
   derniere seance se lit dans `.ai/thought_log.md`.
6. **Commit** : demander a l'utilisateur avant tout commit (regle CLAUDE.md n°16). La
   branche est `feat/v75` et on n'en change pas.

## Decouvertes — a remplir en cours d'execution

### D-1 (E1) — `setTimeout` est bride comme `rAF` : le plan se trompait, corrige

Le plan interdisait `requestAnimationFrame` dans la boucle d'export (bride en onglet
d'arriere-plan) et prescrivait `setTimeout` a la place. **C'est faux** : les navigateurs
brident AUSSI les minuteurs des onglets caches, a une seconde. Mesure faite dans l'onglet du
rejeu, `document.hidden === true`, 20 tirages :

| Primitive | Latence moyenne | Pire cas |
|---|---|---|
| `setTimeout(…, 0)` | 673,53 ms | 1009 ms |
| `MessageChannel` | 0,06 ms | 1 ms |

Consequence mesuree : le premier export d'essai (30 images) a pris **10,2 s**, soit un DIXIEME
du temps reel — l'inverse exact du but du chantier. Tout ce temps etait passe dans les attentes
de contre-pression.

Correction : module `eventLoopYield.ts` (`yieldToEvents()`, `MessageChannel` avec repli
minuteur), consomme par la contre-pression de l'encodeur et, en E3, par la boucle d'export.
Apres correction, meme essai porte a 300 images en 1280x720 : **0,54 s, soit 18,6x le temps
reel**, onglet toujours cache. L'item E3 concerne a ete reecrit.

### D-3 (E1) — Toile teintee : LE RISQUE BLOQUANT N'EXISTE PAS

Verdict NEGATIF, sur trois preuves qui couvrent toutes les sources d'image du canvas :

1. **Le fond de carte ne peut pas teinter, par construction.** Il n'est pas charge par URL
   distante : `useReplayMapImage` (`queries.ts`) recupere un blob par l'API, en fait une
   `URL.createObjectURL`, et decode l'image depuis cette URL `blob:`. Une URL d'objet creee
   par la page est de MEME ORIGINE que la page — il n'y a pas de cas ou elle teinte.
2. **Les icones et emblemes sont en chemins racine** : `/titles/{slug}/teams/{id}.png`
   (`teamLogoPath`), `/jeu/silhouette-*.png` (`weaponFullIcon`). Meme origine que le
   document, donc pas de teinture possible.
3. **Verification a l'execution** (navigateur, 2026-08-28) : trois assets reels charges,
   dessines dans un canvas, puis `ctx.getImageData()` ET `new VideoFrame(canvas)` — les deux
   passent sur les trois. Un canvas teinte aurait leve `SecurityError` sur les deux.

Aucune mesure CORS n'est donc necessaire. La confirmation finale viendra de l'export reel
en E3 (recette utilisateur), mais plus rien ne bloque l'engagement des etapes suivantes.

### D-4 (E3) — Le plan demande une recette d'export AVANT de livrer le bouton d'export

Le gate d'E3 exige « un export reel MUET d'un match court, verifie a l'oeil par
l'utilisateur ». Or l'UI qui declenche un export est l'objet d'E5. Aucune des deux etapes
n'est mal decoupee — c'est l'ORDRE des gates qui est incoherent, et il ne s'est vu qu'a
l'execution.

Traitement : E3 est close sur ses gates TECHNIQUES ; sa recette visuelle est reportee a E5,
ou elle se prononcera en meme temps que celle d'E2 (elle aussi en attente d'un export reel).
Aucun perimetre n'est reduit — seule la date d'une verification bouge, et c'est un report
VALIDE au sens du contrat (dependance explicite : la recette a besoin d'un declencheur).

### D-5 (E3) — Cuisson des calques statiques : non verifiee avant l'export, et assume

L'item « prealables asynchrones » demandait de verifier que les calques statiques (sol,
zones, chaleur) et les icones sont cuits avant la premiere image. Ils le sont par des effets
qui vivent dans QUATRE hooks distincts (`useReplayStaticLayers`, `useGrenadeIcons`,
`useReplayWeaponPads`, `useReplayFlagCarries`), et aucun ne publie de drapeau de
disponibilite. En creer un pour chacun et le remonter jusqu'a l'export couterait plus de
lignes de canvas que tout le cablage de l'export — dans un fichier deja a son plafond.

Le cas est par ailleurs inatteignable en pratique : la commande d'export vivra dans la barre
de lecture, qui ne se rend qu'apres le montage du canvas, donc apres le lancement des
cuissons ; et le mode de defaillance est benin (les premieres images d'un export lance dans
la seconde du chargement sortiraient sans fond de carte). A rouvrir SI un utilisateur
signale un artefact de debut de clip — pas avant.

### D-2 (E1) — Le lockfile perd des champs `libc` a l'installation

`npm install mp4-muxer` a retire ~90 lignes de `package-lock.json` : uniquement des champs
`libc` sur des dependances optionnelles Linux. C'est une difference de version de npm entre la
machine locale et celle qui a ecrit le lockfile, pas un changement de dependances (`package.json`
ne gagne qu'une ligne). Non traite — hors perimetre, et sans effet sur l'installation.

## Risques identifies

1. **Toile teintee** — bloquant s'il se realise ; leve en E1 avant tout le reste.
2. **Memoire** — 10 min a 30 im/s = 18 000 images. Le drainage par contre-pression est
   obligatoire, pas une optimisation.
3. **Duree percue** — meme a 5-15x le temps reel, un long match prend des minutes. D'ou la
   progression et l'annulation (D8), non negociables.
4. **Couverture navigateur** — `VideoEncoder` absent sur Firefox/Safari anciens : le repli
   temps reel (D5) est la reponse, pas un bouton grise.
5. **Divergence surimpression DOM / canvas** — les deux vivront cote a cote. Attenuation :
   les peintres consomment les MEMES modules de logique, et le garde-rail
   `replayOverlayStyles.guard.test.ts` existe deja pour l'habillage.

## Ce que ce plan ne fait pas

- Pas de definition au choix, pas de vitesse d'export, pas de partage de lien.
- Pas de fil des eliminations, de fiches ni de bandeau de score dans le clip (exclusions
  utilisateur). Les ajouter plus tard resterait possible : ce sont des peintres de plus
  dans `overlayPaint.ts`, pas une autre architecture.
