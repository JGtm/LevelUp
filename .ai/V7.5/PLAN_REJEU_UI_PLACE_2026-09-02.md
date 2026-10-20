# PLAN — Rejeu 2D : la regression des fiches, puis la place gagnee (lecteur + tiroir)

> Branche `wt/rejeu-ui-place` depuis `feat/v75`. Revue UI de l'utilisateur du 2026-09-02,
> apres le lot hauteur elastique. Trois etapes, la regression d'abord.

## Constat

**Fiches tronquees** — DOUBLE CAUSE, confirmee par le calcul. Une fiche coute ~100 px, un
en-tete d'equipe ~30 : 4 joueurs = 442 px. Le bloc est plafonne a `xl:max-h-[62%]` de la
rangee, dont la hauteur est celle de la colonne carte.

| Effectif | Besoin | Avant (rangee 773) | Apres (ecran contraint, rangee 653) |
|---|---|---|---|
| 4 v 4 | 442 | 479 dispo — tenait a 37 px pres | **405 dispo — tronque** |
| 5 v 5 | 542 | **deja tronque** | tronque |
| BTB | ~1250 | **deja tronque** | tronque |

Le defaut preexistait sur les gros effectifs ; la hauteur elastique a mange les 37 px de
marge du 4v4. Le correctif traite les deux.

## Decisions

- **D1 — Les fiches cessent d'etre un pourcentage de la carte.** Plafond en px
  (`min(30rem, calc(100% - 12rem))`) : au plus 30 rem, et jamais au point de laisser moins
  de 12 rem au fil. Un pourcentage d'une hauteur devenue variable ne peut pas tenir une
  promesse exprimee en fiches.
- **D2 — Le temps quitte les bornes pour le curseur** (proposition utilisateur, retenue
  contre la mienne) : les libelles debut/milieu/fin disparaissent, le temps suit le point
  qui avance. UN element au lieu de deux, au bon endroit. Ecriture imperative par
  `clockRef`, comme aujourd'hui : la bulle ajoute une position horizontale, pas un rendu.
- **D3 — « Noms » et le son « Objectifs » disparaissent, toujours actifs.** Les flags,
  leurs cles de stockage et leurs cles i18n partent avec (zero code mort). Consequence
  assumee : qui les avait eteints les verra se rallumer.
- **D4 — « Lecture auto » perd sa section, pas sa place.** Ma proposition de la ranger dans
  le menu Vitesse est ABANDONNEE (rejet utilisateur, justifie : un menu « Vitesse » parle de
  vitesse). Elle devient la premiere ligne du tiroir, sans en-tete a elle.
- **D5 — Chaque pixel rendu par le lecteur devient de la carte**, sans rien recalculer :
  `freeSpaceFor` deduit le chrome par soustraction (lot precedent).

## Etape 1 — La regression des fiches

- [x] 1.1 `replay.tsx` : plafond des fiches en px borne par la place du fil (D1).
- [x] 1.2 Test de non-regression sur le contrat de disposition.

Gate : `npm run typecheck` + `npx vitest run src/features/match-replay`

## Etape 2 — Le lecteur rend ~47 px

- [x] 2.1 `ReplayTransport` : ecart frise/commandes `mt-5` -> `mt-1.5` (20 -> 6 px).
- [x] 2.2 Marges du socle : `pb-3.5` retire, lecteur sorti du `p-3` de la carte pour aller
      bord a bord (sinon l'effet « barre pleine largeur » rate).
- [x] 2.3 Boutons -10 % : lecture 44 -> 40, ronds 34 -> 30.
- [x] 2.4 Labels « Image » et « Exporter » retires. L'export SE RE-ELARGIT pendant le calcul
      pour garder sa progression (defaut deja corrige le 2026-08-28, ne pas le reintroduire).
- [x] 2.5 `ReplayTimelineTracks` : bornes debut/milieu/fin supprimees, temps sous le curseur (D2).
- [x] 2.6 Tests.

Gate : idem + `npm run lint`

## Etape 3 — Le tiroir passe de ~900 a ~500 px

- [x] 3.1 Retrait de « Noms » et du son « Objectifs » (D3), flags et cles compris.
- [x] 3.2 « Lecture auto » sans section (D4).
- [!] 3.3 Calques en 2 colonnes — NON TRAITE. Justification au journal : la grille a deux
      colonnes a DEJA ete essayee et retiree le 2026-08-29, pour une raison mesuree. Refaire
      le geste inverse quatre jours plus tard sans element neuf serait une re-divergence, pas
      une amelioration. Decision rendue a l utilisateur.
- [x] 3.4 Chaleur : « Ce que la chaleur mesure » et la portee en controles segmentes,
      une ligne chacun au lieu de huit.
- [x] 3.5 Tests.

Gate : idem

## Hors perimetre

- **Prereglages de son** (idee utilisateur du 2026-09-02, retenue mais NON traitee ici) :
  remplacer les bascules par categorie par 2-3 prereglages, les categories devenant une
  option « personnalise ». A instruire comme lot separe — l'etendre ici serait etendre un
  plan deja valide. Le retrait de « Objectifs » (D3) va dans le meme sens et ne le gene pas.
- **Zones des cartes Forge** : agent lance en parallele, worktree dedie.
- **Zoom** : apres, avec surimpression d'angle (glisser + croix directionnelle).

## Decouvertes

## Journal

### 2026-09-02 — Etapes 1, 2 et 3 CLOSES (3.3 statuee [!])

Gate final, dans la session : `npm run typecheck` EXIT=0 ; `vitest run src/features/match-replay
src/features/match-view` -> **166 fichiers, 2328 tests, 0 echec** ; `npm run lint` EXIT=0,
**0 erreur** (23 avertissements prexistants, aucun dans un fichier touche).

**Etape 1 — fiches.** Plafond `xl:max-h-[min(30rem,calc(100%-12rem))]` : au plus 30 rem, jamais
au point de laisser moins de 12 rem au fil. Un garde-rail neuf
(`rosterHeight.guard.test.ts`, 3 tests) refuse la FORME `xl:max-h-[NN%]` et pas seulement la
valeur — un pourcentage se relit tres bien, rien dans sa forme ne dit qu'il est adosse a une
hauteur devenue variable, et une revue future le reintroduirait de bonne foi.

**Etape 2 — lecteur, ~47 px rendus au terrain.** `mt-5` -> `mt-1.5` ; socle sorti du `p-3` de la
carte (bord a bord, arrondis tombes, rembourrage 14 -> 12/10) ; bouton lecture 44 -> 40, ronds
34 -> 30, pastilles 32 -> 28 ; libelles « Image » / « Exporter » / « REC » retires (avec leurs
cles i18n et le contrat), l'export se re-elargissant toujours pour sa progression.

LE TEMPS A REMPLACE LES TROIS BORNES. `--played` se pose desormais sur le PARENT du champ et
non sur le champ : les proprietes personnalisees heritent, donc le degrade de la piste ne change
pas d'un pixel, mais la bulle — un frere du champ — peut la lire. Posee sur le champ, elle
serait restee invisible a tout ce qui n'est pas lui (un `input` n'a pas de descendants). La
bulle est `aria-hidden` : le champ porte deja `aria-label={t.time}`, deux elements du meme nom
pour une seule information genent la navigation au lecteur d'ecran et rendent les tests
ambigus. `axisClocks` et les trois props sont supprimes.

**Etape 3 — tiroir.** « Noms » et le son « Objectifs » retires, toujours actifs : flags, cles de
stockage, cles i18n et entrees de contrat partis avec eux. Un sous-type
`TogglableSoundCategory = Exclude<SoundCategory, 'objective'>` tient desormais la paire au
COMPILATEUR — sans lui, retirer un libelle i18n compilait et cassait a l'execution. « Lecture
auto » perd son en-tete de section (et `playbackTitle` avec). Les deux axes de la chaleur
passent en `SettingsSegments` (neuf) : 8 lignes -> 2, et la forme dit enfin qu'il s'agit d'un
choix parmi N — `role="radiogroup"` / `role="radio"`.

**3.3 NON TRAITEE — decision rendue a l'utilisateur.** Verification sur pieces avant de coder
(regle 4 du contrat) : `ReplaySettingsLayers.tsx` porte un commentaire date du 2026-08-29 qui
dit que la grille a DEUX COLONNES a existe (posee le 2026-08-24, « un element par ligne c'est
inefficace ») et qu'elle a ete RETIREE pour une raison mesuree — le tiroir fait `w-72` (288 px)
moins `px-3`, soit 264 px utiles ; deux rails cote a cote a ~130 px tronquent « Objets laches au
sol », un interrupteur se lisant sur son rail (libelle a gauche, etat a droite). La proposition
que l'utilisateur a validee ignorait cet historique, que je n'avais pas verifie avant de la
faire. La refaire sans element neuf serait une re-divergence.

Options a trancher : (a) laisser une colonne — le tiroir gagne deja ~200 px par les autres
items ; (b) elargir le tiroir a `w-80`/`w-96`, au prix de plus de carte recouverte ; (c) grouper
par sujet en UNE colonne — plus lisible, mais +3 en-tetes, donc PLUS haut.

## Decouvertes (non traitees)

- L'`input[type=range]` de la frise expose son numero d'IMAGE comme valeur. `aria-valuetext`
  porterait le mm:ss lisible ; c'est le bon geste d'accessibilite, hors perimetre ici.
