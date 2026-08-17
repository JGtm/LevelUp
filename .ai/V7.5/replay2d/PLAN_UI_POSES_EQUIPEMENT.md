# Plan — UI des poses d'equipement : le mur INCURVE, le capteur pulse, les objets sans nom

> Ecrit le 2026-08-18. Decisions utilisateur du 18/08 : le mur doit etre **INCURVE** — « cet
> equipement laisse passer les degats dans un sens et pas dans l'autre » : un arc dont la
> concavite regarde le poseur (les tirs allies sortent, les tirs ennemis butent sur la face
> convexe). Capteur = disque/zone radar pulsee ; `other` = rien par defaut (bascule « objets
> non identifies » dans le tiroir), en attendant le nommage structurel
> (`PLAN_NOMMAGE_EQIP_TRANSLOCATEUR.md`, lot parallele). Web pur, worktree frere
> `wt/ui-poses`, contrat `plan-execution`.

## La donnee (schema 9, deja publiee — `caca2289b`)

`doc.equipmentPlacements[]` : `{t0, t1, x, y, z, family, id, ownerSlot, poseHeading}` —
`family` ∈ manifeste (`wall`, `sensor`, `other` aujourd'hui ; d'autres familles arriveront par
le lot de nommage : le rendu doit etre TABLE par famille, une famille inconnue = rien).
`poseHeading` = cap de visee du poseur a la pose (present sur ~86 % des poses), degres monde ;
`ownerSlot` = vie du poseur (-1 si aucun a < 3 m). Couverture publiee dans `doc.coverage`.

## Decisions tranchees avant execution

1. **Mur = ARC de cercle**, centre sur la position, ouverture ~110°, rayon ~1,6 m monde (a
   l'echelle de la carte : `worldToCanvas`), **concavite vers le poseur** : la corde regarde
   le poseur, la convexite regarde l'ennemi — c'est-a-dire que le milieu de l'arc est place
   DEVANT la position dans la direction `poseHeading` (le mur se pose devant soi), l'arc
   s'ouvre vers l'arriere. Trait 2 px, couleur de l'equipe du poseur, alpha 0,9, leger halo.
   Sans `poseHeading` : un cercle pointille du meme rayon (aucune orientation inventee).
   Duree : [t0, t1] mesures, rien de plus.
2. **Capteur = disque pulse** : rayon monde declare `SENSOR_RADIUS_M` (le film ne porte pas la
   portee — valeur nommee, commentee comme choix d'ecran, pas de donnee), remplissage a alpha
   faible + anneau, pulsation lente (periode 1,6 s) fonction du temps (rejouable, patron
   `explosionFx`), statique sous `prefers-reduced-motion`. Couleur de l'equipe du poseur.
3. **`other`** : rien par defaut ; bascule « Objets non identifies » (OFF) dans le tiroir qui
   dessine un point neutre (encre `--muted`) avec infobulle « objet d'equipement non
   identifie » au survol. Quand le nommage arrivera, chaque nouvelle famille aura SA regle de
   rendu ou rien.
4. **Sons** : `Drop Wall - Activate` et `Threat Sensor - Activate` a `t0` (bibliotheque
   utilisateur, recette validee : PCM s16le 48 kHz stereo, borne 4 s pour la categorie
   equipements), categorie « Equipements » des filtres ; garde-rail assets ; silence propre
   pour toute famille sans fichier.
5. **Bascule « Equipements poses »** (ON par defaut) dans le tiroir, persistee
   (`usePersistedFlag`), FR/EN ; le calque se pose SOUS les marqueurs de joueurs et AU-DESSUS
   du fond/zones/chaleur.
6. Tokens uniquement (skill `color-tokens`) ; `worldToCanvas` et DPR comme `grappleLayer.ts` ;
   pas de React dans la geometrie (logique pure testee + `draw*Layer`).

## Phases

- [x] 1 `equipmentPlacementsLayer.ts` : logique pure (poses actives a une frame, geometrie
      de l'arc — points de l'arc en monde puis projection —, du disque, du point neutre) +
      tests (arc devant le poseur : le milieu de l'arc est a `poseHeading` de la position ; sans
      cap -> cercle pointille ; fenetre [t0, t1] ; famille inconnue -> rien).
      24 tests verts (`equipmentPlacementsLayer.test.ts`).
- [x] 2 Rendu canvas dans `ReplayCanvas.tsx` (ordre des calques), bascules dans le tiroir
      (`ReplaySettingsDrawer.tsx`, `useReplaySettings.ts`), i18n FR/EN. Le calque se pose
      entre les projectiles et `drawTracksLayer` — au-dessus du fond/zones/chaleur/objectifs,
      sous les marqueurs de joueurs. 29 tests verts au tiroir (5 neufs).
- [x] 3 Sons : 2 WAV (`wall_activate`, `sensor_activate`), table `EQUIPMENT_PLACEMENT_SOUND_STEMS`
      par famille dans `replaySound.ts` (jointure `family`), garde-rail assets, tests.
      Recette REPRODUITE A L'OCTET sur `camo_activate.wav` avant de produire quoi que ce soit
      (meme md5) : `-t 4 -ar 48000 -ac 2 -c:a pcm_s16le -map_metadata -1 -fflags +bitexact
      -flags +bitexact`.
- [x] 4 Infobulle au survol : « Mur de protection » / « Capteur de menaces » / « objet
      d'equipement non identifie » + le nom du poseur (FR/EN), `ReplayPlacementTip.tsx` +
      `usePlacementHover.ts`. Le survol se rejoue sur la DONNEE (`placementAt`), pas sur les
      pixels ; la plus petite zone gagne, sinon un mur pose dans un capteur serait
      inatteignable au pointeur.

**Gate** : purge `.tmp`, typecheck, lint, vitest — exit 0 ; zero hex ; gate VISUEL utilisateur
sur `000d5950` (13 murs, 19 capteurs nommes) et `06dfe6d9` (Threshold, 892 poses dont 0
nommee aujourd'hui — c'est le film qui montre l'effet de la bascule « non identifies »).

**Gate technique : PASSE le 2026-08-18** (worktree `wt/ui-poses`, `apps/web`) :

    npm run typecheck                                        exit 0
    npm run lint                          exit 0 — 0 erreur, 19 warnings preexistants
    npm run test                          4014 verts / 3 echecs = TIMEOUTS de garde-rails
                                          qui balaient le disque (skillTiers, lab-removal,
                                          generated-types-fresh) sous la charge de la suite
                                          complete ; rejoues ISOLES : 3 fichiers, 7 tests,
                                          verts en 3,19 s. Aucun rapport avec ce lot.
    grep hex / classes Tailwind couleur dans les 4 fichiers neufs   0

**Gate VISUEL : reste a faire** (utilisateur, temoins `000d5950` et `06dfe6d9`).

## Regles dures

Aucune orientation inventee (pas de cap -> cercle) ; famille inconnue -> rien ; sons par
famille avec fichier ; textes journal/registre fournis au CR (worktree frere) ; JAMAIS
`git add -A` ; jamais d'attente passive ; pas de push.

## Journal

### 2026-08-18 — les quatre phases closes, le gate visuel reste

**CE QUE LE PLAN DISAIT ET QUE LE CONTRAT NE DISAIT PAS.** Le plan nomme les champs
`ownerSlot` et `poseHeading` ; le contrat publie (`generated.ts`, schema `EquipmentPlacement`)
les nomme `owner` et `h`. Verifie sur pieces AVANT de coder — c'est exactement le cas que la
regle 4 vise. Le code lit les noms du contrat ; l'alias `ReplayEquipmentPlacement` est pose
dans `lib/api/types.ts` avec, en commentaire, ce que `h` EST (le cap de VISEE du poseur,
convention `Point.h`) et ce qu'il n'est pas (l'orientation de l'objet, que le film ne porte pas).

**LES CHOIX GEOMETRIQUES, ET CE QUI LES FONDE.**

    mur      arc de 110°, rayon 1,6 m monde, milieu de l'arc a `h` de la position
             (donc devant le poseur, concavite vers lui) ; trait 2 px + halo 5 px a 0,22
             sans cap  -> cercle POINTILLE ferme du meme rayon, aucune direction affirmee
    capteur  disque `SENSOR_RADIUS_M` = 8 m, remplissage 0,10 + anneau 0,55 a 1,25 px,
             respiration ±6 % de periode 1,6 s, FIGEE sous mouvement reduit
    autres   point neutre de 2,5 px a l'encre `--muted-foreground`, alpha 0,5

Une corde de 2,6 m (2 x 1,6 x sin 55°) : l'ordre de grandeur du mur du jeu. Le rayon du
capteur est DECLARE et le code le dit — le film ne porte aucune portee de detection.

**UN PLANCHER DE LISIBILITE A ETE AJOUTE, et il faut le dire** (`WALL_MIN_RADIUS_PX` = 6) :
sur une carte de Big Team Battle (~2,5 px/m), 1,6 m de rayon tombe a 4 px — une eraflure. Le
rayon MONDE effectif est releve jusqu'a ce que l'arc atteigne 6 px d'ecran. La pose reste a sa
place exacte ; seule sa TAILLE cesse de suivre l'echelle sous ce seuil. C'est un choix d'ecran
de plus, ecrit comme tel.

**DEUX ECARTS AU TEXTE DU PLAN, assumes et motives.**
1. Le plan ecrit « encre `--muted` » pour le point neutre. `--muted` est une SURFACE du systeme
   (oklch 0,269 en sombre, 0,96 en clair) : un point de cette teinte serait invisible sur la
   carte dans les deux themes. L'encre neutre du systeme est `--muted-foreground` (0,708 /
   0,52), celle que le canvas emploie deja partout (`floorStyle.edge`) — c'est elle qui est
   servie, sans nouvelle variable a lire.
2. Le plan proposait « legende OU infobulle » ; c'est l'infobulle qui est livree, avec le nom
   du poseur, comme demande au lot.

**LE SURVOL SE REJOUE SUR LA DONNEE, PAS SUR LES PIXELS** : un canvas peint ne sait plus ce
qu'il porte. `placementAt` refait la projection et prend la PLUS PETITE zone touchee — sans
cette regle, le disque de 8 m du capteur rendrait inatteignable tout mur pose dedans. L'image
courante est lue dans la reference de la boucle de lecture, jamais dans un etat React : la
publier a chaque image pour un survol couterait le budget d'animation. Consequence assumee et
unique : si une pose expire sous un pointeur IMMOBILE, son infobulle tient jusqu'au prochain
mouvement. A l'arret — le cas ou l'on inspecte — la lecture est exacte.

**LES SONS : LA RECETTE A ETE PROUVEE AVANT D'ETRE APPLIQUEE.** `camo_activate.wav` a ete
REPRODUIT A L'OCTET depuis sa source (`EQUIPMENT/Active Camo/Active Camo - Activate.wav`, PCM
f32le 44,1 kHz) avec la recette annoncee — meme md5 `18124f76148f15b4563167e7f0fdfdf6`. Les
deux fichiers neufs en sortent : `wall_activate.wav` (2,940 s) et `sensor_activate.wav`
(1,265 s), tous deux PCM s16le 48 kHz stereo, en-tete RIFF de 44 octets. Le capteur passe le
garde-rail de duree de justesse (1,265 s pour un plancher a 1,2 s) : c'est la duree de sa
source, elle n'est pas retronquee.

**UNE TABLE SONORE DISTINCTE, et ce n'est pas une duplication.**
`EQUIPMENT_PLACEMENT_SOUND_STEMS` ne fusionne pas avec `EQUIPMENT_SOUND_STEMS` : la seconde
porte des EPISODES D'ETAT sur le porteur (un debut ET une fin mesures), la premiere des OBJETS
POSES dont seul le geste est un evenement. La disparition d'un mur n'est pas un acte, c'est la
fin d'une duree — rien ne la sonne. Les fondre obligerait chaque famille a declarer une
desactivation qu'elle n'a pas.

**LES DEUX BASCULES, et leurs defauts opposes** : « Equipements poses » ALLUMEE (un mur change
la lecture d'un echange, comme une zone de capture), « Objets non identifies » ETEINTE — sur un
film BTB, aucune pose n'est nommee aujourd'hui, les allumer y poserait des centaines de points.
Les deux suivent la regle du bouton Zones : une bascule qui ne commanderait rien n'est pas
affichee (`available` = au moins une pose que la TABLE sait dessiner).

**DECOUVERTE non traitee, portee au registre** : l'inline `{ bounds, width, height, pad }` du
cadrage etait recopie SIX fois dans `ReplayCanvas.tsx` (draw + cinq effets de cuisson). Ce lot
en a memoise UN (`canvasView`, partage par le dessin et le survol — ils doivent lire la meme
projection) et laisse les cinq effets tels quels : hors perimetre.
