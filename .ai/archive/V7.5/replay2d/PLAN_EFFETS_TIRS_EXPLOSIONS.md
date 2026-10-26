# Plan — Muzzle flash, explosions et sons de tir (reference : Csstat)

> Ecrit le 2026-08-15. Demande utilisateur, mot pour mot : « pour les sons [...] on n'en a pas
> besoin [de la direction], on sait deja ou le joueur regarde/vise, si on sait quand il tire ca
> nous suffit » · « je veux comme des muzzle flash, avec differents effets pour plasma et tout,
> c'est plus au niveau des cones de visee (sans les remplacer) » · « pour les explosions de
> grenades, je veux des effets d'explosion pas des symboles » · reference : le projet
> `C:\Users\Guillaume\Projects\Csstat`.
> Execution sous `plan-execution`, branche `feat/v75`, commits par etape, PAS de push.

## LA MESURE QUI DEBLOQUE TOUT (superviseur, 2026-08-15)

Le rendu actuel oriente un tir par le champ `h` de l'EVENEMENT de tir — present sur seulement
**90 tirs sur 483 (18,6 %)**. D'ou l'impression d'absence d'effet.

**Mais le cap de REGARD vit aussi dans les TRAJECTOIRES** : mesure sur le film temoin,
**102 points sur 199 portent `h`** — c'est cette donnee qui alimente deja le calque « Visee ».
Le regard du joueur est donc connu **en continu**, independamment de l'evenement de tir.

**Consequence** : un muzzle flash s'oriente par le REGARD DU TIREUR A L'INSTANT DU TIR (lu
dans sa trajectoire), pas par le `h` de l'evenement. C'est exactement ce que fait la reference
Csstat — son commentaire dit « petite flamme dans la direction du REGARD du joueur qui tire ».
La couverture cesse d'etre 18,6 % et devient celle du regard, bien plus large.

## Ce que fait la reference (lu sur pieces, `C:\Users\Guillaume\Projects\Csstat`)

| element | fichier | ce qu'il fait |
|---|---|---|
| Muzzle flash | `apps/web/src/components/replay/ReplayCanvas.tsx` | flamme courte a la position du tireur, orientee par son REGARD ; `MUZZLE_DURATION = 6` frames ; les armes de MELEE en sont exclues (`MELEE_WEAPON`) ; dessine EN PLUS du cone de visee (`CONE_ANGLE = PI/4`, `CONE_RADIUS = 0.06`) |
| Explosion | `apps/web/src/components/replay/ExplosionFx.tsx` | vraies PARTICULES, timeline **1,4 s** : flash radial -> boule de feu (composition `lighter`) -> onde de choc -> eclats/shrapnel -> poussiere residuelle. Trois familles de particules typees (`Ember`, `Spark`, `DustPuff`) |

## Etape 1 — LES SONS SUR TOUS LES TIRS (le plus simple, et l'utilisateur a raison)

> Aujourd'hui le son ne part que sur les KILLS. Or un son n'a besoin QUE de l'instant.

- [x] 1.1 Declencher un son a CHAQUE tir publie (`doc.shots`, 483 sur le film temoin), avec le
      son de l'arme du tir (`s.w` -> `weaponLabels` -> cle d'arme -> fichier). Aucune direction
      requise.
      FAIT — la cle d'arme MANQUAIT au document : `WeaponLabel.Key` est publie, resolu **a la
      requete** par le service (`internal/service/replay_weapon_keys.go`), comme `mapObjectives`.
      Raison ecrite dans le code : figer la cle au build aurait laisse muets les 23 artefacts
      locaux et toute la production jusqu'a une re-cuisson complete. Verifie sur l'API locale :
      39 libelles d'arme sur 39 portent leur cle.
- [x] 1.2 DENSITE — DECISION UTILISATEUR DU 2026-08-15 : « tu me les mets TOUS autant que
      possible pour le moment, je verrai si c'est trop ensuite ». Donc **aucun filtrage
      editorial** : tout tir dont l'arme a un son le joue. Le SEUL plafond admis est
      TECHNIQUE — le nombre de voix simultanees que le moteur audio tient sans saturer
      (mecanique deja livree au lot 5). Publier le nombre de sons joues contre le nombre de
      tirs, et le nombre de voix refusees par le plafond technique : c'est ce chiffre qui
      permettra a l'utilisateur de dire « c'est trop » en connaissance de cause.
      MESURE (simulation a 1x, une voix tenue 1 s, `SOUND_MAX_VOICES` = 8) :
      | perimetre | tirs | sons publies | voix refusees |
      |---|---|---|---|
      | film temoin 000d5950 | 483 | 483 (100 %) | 46 |
      | 23 artefacts locaux | 17 904 | 17 068 (95,3 %) | 4 897 (28,7 % des sources) |
      Le plafond MORD. Il est laisse a 8 : le relever est un changement d'un chiffre, et c'est
      une decision d'oreille de l'utilisateur, pas du code.
- [x] 1.3 Les armes MUETTES restent muettes (4 mesurees : Bandit EVO, MA5K Avenger, canon a
      combustible, carabine Vestige) — jamais le son d'une autre arme.
      MESURE : ce sont exactement les 4 armes des 836 tirs muets du corpus (Vestige 231,
      Bandit 225, MA5K 171, SPNKr a combustible 59). Aucune autre arme n'est muette.
- [x] 1.4 Le reglage existant (bouton Son, coupe par defaut, volume persiste) commande aussi
      ces sons. Un seul interrupteur, pas deux.
      FAIT par construction : les tirs entrent dans la MEME piste (`buildSoundTimeline`), donc
      le meme curseur, le meme hook, le meme bouton — aucun second reglage n'a ete ajoute.

Gate 1 : nombre de sons joues / tirs publies, publie ; ecoute utilisateur sur un echange nourri.
GATE 1 : chiffres publies ci-dessus. Gates techniques passes (tsc purge, eslint, vitest 21
fichiers / 327 tests, `go test` service + analysis/replay + contracttest, `go vet`, ratchet
golangci-lint 0 issue). **RESTE : l'ecoute par l'utilisateur.**

## Etape 2 — LE MUZZLE FLASH (remplace le rendu actuel des tirs)

- [ ] 2.1 Orientation par le REGARD DU TIREUR a l'instant du tir : lire `h` dans la trajectoire
      du joueur (patron `posOfPlayerAt` du calque des morts, meme interpolation). Publier la
      couverture obtenue contre les 18,6 % actuels. Si le regard manque a cet instant : flash
      NON oriente (une bouffee ronde), jamais une direction inventee.
      MESURE (regle finale : `heldReading` du champ `h`, fenetre de maintien du cone = 5 s,
      sur la vie QUI COUVRE l'instant du tir) :
      | perimetre | tirs dessines | par l'EVENEMENT | par le REGARD |
      |---|---|---|---|
      | film temoin 000d5950 | 483 | 90 (18,6 %) | **482 (99,8 %)** |
      | 23 artefacts locaux | 17 904 | 2 891 (16,1 %) | **17 687 (98,8 %)** |
      Age de la lecture : median 0 ms, p90 100 ms, 94,4 % sous 200 ms. Le `h` de l'evenement
      n'est PAS utilise, meme quand il est la : sur les 2 866 tirs qui portent les deux,
      l'ecart median est de 0,3 deg et 84,4 % sont sous 5 deg — les deux disent la meme chose,
      et une seule regle vaut mieux que deux.
- [x] 2.2 Dessin A LA BASE du cone de visee et SANS le remplacer (demande explicite) : le cone
      reste ce qu'il est (`AIM_LENGTH 46`, `AIM_HALF_ANGLE 0.3`), le flash se pose au marqueur
      du joueur, court et vif.
      FAIT : `drawAimCone` n'est pas touche ; l'eclair naît 5 px devant le marqueur, dans
      l'axe du regard (« au bout du canon, pas dans le torse »). La TRACE de 62 px du lot 3.2
      disparaît avec — c'est elle que l'eclair remplace, pas le cone.
- [x] 2.3 UNE FORME PAR FAMILLE D'ARME, en reprenant les familles DEJA resolues
      (`killEffects` / `[shot_effects]` : ballistic, plasma, light, shock, explosive, melee,
      needles) : la poudre claque bref et blanc, le plasma s'etale et decroit mollement, le
      dur-lumiere est un rai continu, l'arc electrique est brise, les aiguilles sont une
      gerbe. **La MELEE n'a PAS de flash** (regle deja etablie : un coup de marteau n'est pas
      un tir — la reference Csstat l'exclut aussi explicitement).
      FAIT (`muzzleFlash.ts`, 7 formes) ; la melee est ECARTEE en amont (`buildShotFx`), et
      un test verifie que les 7 signatures emises sont distinctes. Mesure : 0 evenement de tir
      de famille `melee` sur les 17 904 du corpus — la regle ne coûte rien et ferme le cas.
- [x] 2.4 Duree courte, de l'ordre de la reference (6 frames ~ 0,6 s a 100 ms/frame — a REGLER
      A L'OEIL avec l'utilisateur, c'est un choix de rendu, pas une mesure).
      La fenetre reste `SHOT_HOLD_MS` = 600 ms (l'ordre de la reference), mais l'INTENSITE
      tombe au CARRE du temps restant : l'eclair claque et s'eteint bien avant la fin de la
      fenetre, tout en restant lisible a 2x. **A REGLER A L'OEIL** au gate 2.
- [x] 2.5 Respect de `prefers-reduced-motion` (variante statique).
      FAIT : sous mouvement reduit l'eclair ne progresse plus et son intensite ne depend plus
      de l'age (test dedie). Le canevas etant dessine en JS, la preference se lit en code —
      la feuille de style ne peut rien pour lui.

### 2.6 — LA COULEUR REVIENT PAR FAMILLE : arbitrage du lot 3.2 ROUVERT par l'utilisateur

> Precision utilisateur du 2026-08-15, qui TRANCHE et remplace la decision precedente :
> « le muzzle flash c'est bien pour les armes CINETIQUES ; pour les autres faut des effets de
> type PLASMA BLEUTE OU ROUGEATRE pour les armes Banished a energie, voire d'ENERGIE VIOLETTE
> pour les armes Forerunner. »

- [x] 2.6a Le lot 3.2 avait acte « famille = FORME, couleur = TIREUR » (regle color-tokens) et
      abandonne la palette du POC. **Cet arbitrage est ROUVERT sur decision utilisateur** :
      la NATURE de la decharge se dit aussi par la COULEUR. Ce n'est pas un retour en arriere
      par oubli — l'ecrire comme tel dans le code, avec la date et la raison.
      FAIT : la reouverture est ecrite, datee et citee mot pour mot en tete de `fxInk.ts`, du
      bloc `--replay-fx-*` de globals.css et de la table `[shot_tints]` du titre.
- [x] 2.6b Trois familles de rendu, telles que l'utilisateur les nomme :
      - **CINETIQUE** (UNSC balistique) -> muzzle flash de flamme, bref et chaud ;
      - **BANISHED A ENERGIE** -> plasma **bleute OU rougeatre** — les deux teintes existent
        dans le jeu (plasma Covenant bleu, armes Brute rouge/orange). MESURER la repartition
        des armes du corpus entre ces deux teintes AVANT de choisir, et publier la table ;
      - **FORERUNNER** -> **energie violette**. (Le POC disait « cyan-or » : l'utilisateur
        tranche autrement, sa version fait foi.)
      Les autres familles deja resolues (choc, explosif, aiguilles) gardent leur forme ; leur
      teinte suit la meme logique de NATURE, a proposer avec la table.
      TABLE MESUREE AVANT DE CHOISIR (17 904 tirs des 23 artefacts locaux, teinte resolue par
      la chaîne reelle famille -> weapon_key -> `[shot_tints]`) :
      | teinte | tirs | part | armes |
      |---|---|---|---|
      | kinetic | 16 380 | 91,5 % | MA40 10 393, Sidekick 4 645, BR75 606, Bandit 225, MA5K 171, S7 123, Dechiqueteur 77, VK78 75, Bulldog 52, Empaleur 13 |
      | plasma_cool | 443 | 2,5 % | Vestige 231, Carabine a impulsion 82, Crematier 67, SPNKr a combustible 59, Pistolet a plasma 4 |
      | needle | 409 | 2,3 % | Needler 409 |
      | plasma_hot | 234 | 1,3 % | Ravageur 135, Fusil traqueur 76, Calcineur 23 |
      | electric | 186 | 1,0 % | Disrupteur 130, Fusil electrique 56 |
      | blast | 102 | 0,6 % | M41 SPNKr 87, Hydra 15 |
      | forerunner | **0** | 0 % | Rayon de Sentinelle — AUCUN tir dans le corpus |
      | (sans teinte) | 150 | 0,8 % | armes hors registre, teinte neutre |
      **BANISHED A ENERGIE : 677 tirs — 443 bleutes (65,4 %) contre 234 rougeatres (34,6 %).**
      Les deux teintes sont donc GARDEES : une seule effacerait un tiers de la population.
      Le VIOLET Forerunner est declare mais n'a AUCUN tir a l'appui dans ces 23 films — dit
      plutot que masque ; il ne se verra que le jour ou un film portera un Rayon de Sentinelle.
      Les 3 familles cinetiques Banished (Dechiqueteur, Empaleur, plus le Fusil traqueur cote
      chaud) sont rangees a la main : les deux premieres crachent du METAL, la troisieme une
      decharge chaude — le classement suit la nature du projectile, pas le camp.
- [x] 2.6c **COMMENT sans violer la regle du depot** : la regle interdit les valeurs hex et les
      classes Tailwind couleur dans `features/`. Les teintes de famille passent donc par des
      TOKENS SEMANTIQUES dedies (un token par famille d'effet), resolus au dessin — jamais une
      couleur ecrite en dur dans le composant. Precedent a suivre : `canvasInk.ts`, qui gere
      deja de l'encre de canevas dans ce cadre. **Ne pas contourner la regle : l'etendre
      proprement.**
      FAIT, en TROIS maillons et un garde-rail :
      - le SERVEUR ne publie jamais une couleur, seulement une NATURE (`[shot_tints]` du
        titre, liste fermee validee, publiee dans `weaponLabels[id].tint` a la requete) ;
      - le THEME porte les valeurs (`--replay-fx-*`, une par nature, dans les deux themes) ;
        ce ne sont PAS des tokens `--ac-*` a dessein : ceux-la sont remappes par les palettes
        d'accessibilite, et un plasma bleu devenu orange sous Okabe-Ito detruirait
        l'information meme que la teinte porte. La raison est ecrite dans globals.css ;
      - la FEATURE lit ces variables (`fxInk.ts`, patron `canvasInk.ts`) — 0 hex, 0 oklch, 0
        classe couleur dans `features/`, verifie par test.
      Garde-rail `fxInk.guard.test.ts` : le vocabulaire du valideur Go, celui de la table du
      titre et celui du client sont la MEME liste, et chaque teinte a sa variable dans les
      deux themes. Sans lui, une teinte ajoutee cote serveur retomberait en neutre en silence.
- [x] 2.6d **L'EFFET DE TIR NE PORTE QUE L'ARME.** Correction utilisateur du 2026-08-15, mot
      pour mot : « les couleurs des effets de tirs ne prennent pas la couleur du tireur, tu as
      confondu, elle prend seulement l'ARME en compte ». La couleur d'un effet de tir est donc
      determinee UNIQUEMENT par la famille de l'arme — ni coeur, ni lisere, ni repli aux
      couleurs d'equipe. Ne PAS inventer de compromis « pour distinguer les joueurs » : le
      marqueur du joueur et son cone de visee portent deja son identite, l'effet dit la NATURE
      DE LA DECHARGE et rien d'autre.
      (Une version anterieure de cet item demandait l'inverse ; c'etait une erreur de
      transmission du superviseur, corrigee ici avant execution.)
      FAIT : `drawShotsLayer` ne reçoit plus AUCUNE couleur de joueur (ni `colorOfSlot`, ni
      repli) — la signature de la fonction rend la faute impossible, et un test verifie que
      les seules couleurs peintes sont la teinte de l'arme et le coeur incandescent. Les
      effets de MORT, eux, gardent la couleur du tueur : ils sont hors perimetre de ce plan.

Gate 2 : couverture d'orientation publiee ; verification a l'oeil par l'utilisateur sur un
echange nourri, avec au moins un tir balistique, un plasma et une melee (qui ne doit RIEN
afficher).
GATE 2 : couverture publiee (2.1). Gates techniques passes (tsc purge, eslint, vitest 24
fichiers / 354 tests dont 15 sur l'eclair, 8 sur la resolution du regard et 4 de garde-rail
de teintes ; `go test` service + mappings + replaylabels + contracttest ; ratchet
golangci-lint 0 issue). Chaine verifiee de bout en bout sur l'API locale : les 39 libelles
d'arme du temoin portent leur cle ET leur teinte (15 kinetic, 6 plasma_hot, 5 plasma_cool,
4 blast, 4 electric, 2 needle, 1 forerunner, 2 sans teinte). **RESTE : l'oeil de
l'utilisateur** — en particulier le REGLAGE DE DUREE (item 2.4).

## Etape 3 — LES EXPLOSIONS DE GRENADE (des particules, pas un symbole)

- [x] 3.1 Porter une explosion PARTICULAIRE sur le modele de `ExplosionFx.tsx` : flash radial,
      boule de feu en composition additive, onde de choc, eclats, poussiere residuelle. Le
      port est un PORTAGE, pas une reinvention — lire le fichier avant d'ecrire.
      FAIT (`explosionFx.ts`) : le fichier de reference a ete lu avant d'ecrire ; timeline de
      1,4 s, memes coefficients de traînee (0,86 braises / 0,9 eclats / 0,95 poussiere), meme
      flash de 70 ms et meme onde de 380 ms en easeOutCubic. UNE DIFFERENCE ASSUMEE : la
      reference anime A ETAT, image par image. Ici la lecture recule et saute au curseur —
      chaque particule est donc REJOUEE depuis son germe a chaque image (fonction du temps),
      ce qu'un test verrouille. Les quatre couleurs de feu ecrites en dur dans la reference
      deviennent DEUX encres du theme (coeur incandescent -> teinte du type).
- [x] 3.2 OU elle se pose : au dernier point replique du vol de la grenade (le lien
      grenade<->projectile est publie depuis le lot 2.3). **RAPPEL MESURE** : le film ne porte
      AUCUN evenement de detonation — l'ecran dit « derniere position connue », jamais
      « impact ». L'explosion est une MISE EN SCENE assumee a cet endroit, et ce point reste
      ecrit dans le code.
      FAIT : la position vient de `buildGrenadeRestFx` (inchange). Le point est ecrit en tete
      de `explosionFx.ts` ET dans `drawGrenadeRestLayer`. La note d'ecran a ete RENFORCEE
      (FR et EN) : « l'explosion se joue a la derniere position connue du projectile — le film
      n'enregistre aucune detonation, la mise en scene est assumee a cet endroit ».
      Mesure : 1 117 lancers dans le corpus, 862 lies a un projectile, 801 explosions posees.
- [x] 3.3 PAR TYPE : Frag et Plasma explosent ; **Dynamo (rang 2) ne « explose » pas** — elle
      pose une nappe electrique persistante (~2,5 s), regle deja livree au lot 2.3 et
      confirmee par l'utilisateur (« les grenades lightning/dynamo n'explosent pas mais ont un
      effet electrique qui dure quelques secondes »). Spike : a trancher a l'oeil.
      FAIT (`restKindOf`, teste) : Frag `blast`, Plasma `plasma_cool`, Dynamo NAPPE inchangee,
      rang inconnu = halo discret. **SPIKE : TRANCHEE EN EXPLOSION `needle`** — elle se plante
      puis projette ses pointes, et c'est le seul type dont la fin de vol est certifiee
      `at-rest` (15 vols lies sur 21) : l'objet s'immobilise AVANT de se declencher. Repartition
      des 862 vols lies : frag 684, plasma 68, dynamo 61, spike 49. **A CONFIRMER A L'OEIL** —
      la ramener au halo tient en une ligne de `restKindOf`.
- [x] 3.4 Cout : une explosion particulaire par grenade, sur un canevas deja charge. Mesurer
      le cout d'image et BORNER le nombre de particules ; une explosion ne doit jamais faire
      tomber la lecture sous la fluidite.
      BORNE POSEE ET MESUREE : 28 particules par explosion (12 braises + 10 eclats + 6
      bouffees), au plus 84 pas de simulation. Cout releve sur toute la timeline et FIGE par
      test : **19 degrades radiaux et 227 primitives** au pire instant d'UNE explosion. Le
      corpus ne montre jamais plus de **5 explosions simultanees** dans la fenetre de 1,4 s,
      soit un pire cas reel de 95 degrades et ~1 135 primitives par image — sur un canevas
      dont le fond, la structure, les zones et les objectifs sont deja cuits hors ecran.
      Une explosion terminee ne coute pas une seule operation (verifie par test).

Gate 3 : verification a l'oeil sur un match a grenades ; mesure du cout d'image avant/apres.
GATE 3 : cout d'image publie (3.4). Gates techniques passes (tsc purge, eslint 0, vitest 25
fichiers / 368 tests dont 10 sur l'explosion, plus `lib` 909 tests et `match-view` 186 tests
pour la note d'ecran bilingue). **RESTE : l'oeil de l'utilisateur sur un match a grenades**,
et la confirmation du choix Spike (item 3.3).

## Decouvertes (consignees, NON traitees — regle 7 du contrat d'execution)

- `internal/analysis/weaponv3` depasse le `-timeout 60s` de la cible locale `make go-api-test`
  (il met ~63 s et passe sans la borne). Le paquet ne depend que de `internal/analysis`,
  qu'aucun commit de ce lot ne touche. Ligne au REGISTRE_REPORTS ; la CI reste le juge.
- Le POC de rejeu teintait ses familles par sept couleurs `hsla` en dur ; l'en-tete de
  `shotEffects.ts` explique encore pourquoi la teinte avait ete ABANDONNEE au lot 3.2. Ce
  texte reste vrai POUR LES EFFETS DE MORT (hors perimetre de ce plan, non touches), mais il
  sera a relire le jour ou la teinte de famille descendra aussi sur eux.

## Hors perimetre

- La direction du tir VERS LA VICTIME : elle n'existe pas dans le film (mesure : les
  evenements de tir ne portent aucune victime). Ce plan ne la cherche pas — il utilise le
  REGARD, ce qui est la demande de l'utilisateur et ce que fait la reference.
- Les effets de MORT (livres, orientes tueur->victime) : non touches.
- Les 4 armes sans son : rien a cabler sans une source sonore supplementaire.

## Ce qui peut faire echouer ce plan

1. **La densite sonore** : 483 tirs par match. Sans la borne de l'item 1.2, l'effet est
   contre-productif. C'est le vrai risque de l'etape 1.
2. **La couverture du regard** : 102 points sur 199 en portent un sur le film temoin. Si le
   regard manque justement aux instants de tir, le gain sur les 18,6 % sera faible — a
   MESURER a l'item 2.1 avant de conclure.
3. **Le cout d'image des particules** : une explosion Csstat est riche ; le canevas du rejeu
   porte deja fond, structure, callouts, objectifs, trajectoires, effets. Borner, mesurer.
