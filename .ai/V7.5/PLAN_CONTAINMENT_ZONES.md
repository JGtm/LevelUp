# Plan — Containment des zones : qui tient quoi, et depuis quand

> Ecrit le 2026-08-14 a la demande de l'utilisateur. Reprend le rapport de faisabilite
> `.ai/HANDOFF_CONTAINMENT_ZONES_2026-08-08.md` (lot 4) et l'ACTUALISE : cinq choses ont
> change depuis, dont une qui rend probablement caduque l'etape n°1 d'alors.
> Execution sous `plan-execution`, branche `feat/v75`, commits par etape, PAS de push.

## De quoi il s'agit (l'utilisateur a demande ce que c'est)

**Savoir si un joueur se trouve DANS une zone d'objectif a un instant donne** — et donc qui
tenait la zone, combien de temps, et qui l'a prise. On croise trois choses :

	forme de la zone (catalogue map_objectives.json, en metres monde)
	  x  position du joueur a 100 ms pres (pipeline du rejeu 2D, par xuid)
	  x  instant + auteur des actions d'objectif (analysis/objectiveevents)

Ce que ca DEBLOQUERAIT : le score de zone au cours du temps, « qui a capture », le temps de
controle par joueur/equipe — et, cote rejeu, colorer une zone tenue au lieu de la dessiner
inerte. C'est aussi le prealable ecrit au registre pour tout score over-time zone/hill/skull.

## Ce qui est ETABLI (lot 4, ne pas refaire)

Le croisement existe, pur et teste : `mapvar.Contains`/`Volume` (containment.go), le lecteur
de catalogue (objectives_catalog.go), `AttributeZones` (zone_attribution.go), l'outil de
mesure `cmd/zone-attribution`. **Rien n'est persiste ni affiche**, pour trois raisons qui
restent valables jusqu'a preuve du contraire :

1. **Taux trop bas** : 12,2 % d'appartenance stricte, 28,6 % au meilleur decalage global —
   une colonne remplie une fois sur quatre n'est pas consommable, et une valeur presente n'y
   serait pas distinguable d'une valeur sure.
2. **La correction d'horloge n'est pas etablie** : le retard existe (8 films sur 8), mais sa
   valeur non, et 3 films sur 8 PIQUENT SUR LA BORNE du balayage (-10 s) — mesure tronquee.
3. **Aucun oracle de justesse** : tout ce qui est mesure est « une zone a-t-elle ete
   rattachee », jamais « est-ce LA BONNE ».

## CE QUI A CHANGE DEPUIS LE 2026-08-08 — a lire avant de planifier quoi que ce soit

| # | changement | effet sur ce chantier |
|---|---|---|
| 1 | **L'horloge du rejeu est RESOLUE** (lot 7.2, 14/08) : l'artefact publie `originMs`, et l'horodatage de paquet du film s'est revele etre une **horloge MOTEUR** (des milliers de secondes depuis le demarrage du jeu). Origines mesurees : **3,6 s a 50,8 s** selon le match | **HYPOTHESE CENTRALE DE CE PLAN** : le « retard d'horloge » du containment est probablement CE decalage-la. Il expliquerait pourquoi 3 films sur 8 piquaient a -10 s : le vrai decalage les depassait. Si c'est le cas, il n'y a **rien a balayer** — la correction est connue, exacte, et par film |
| 2 | **Oracle Vagabond disponible** (lot 5 cartes) : la carte porte ses 3 zones de Bastion reelles au catalogue (asset `105f5d84`) | le releve terrain du 2026-08-02 redevient rejouable = le seul oracle qui dise QUELLE zone |
| 3 | **Catalogue d'objectifs : 34 -> 72 cartes** (regenere le 13/08, lot fonds par map_id) — 158 zones Bastion, 236 zones d'extraction | le corpus mesurable s'elargit mecaniquement |
| 4 | **Bornes de dequantification : 15 -> 56 cartes** (lots bornes + map_id) | Prism, Recharge, Deadlock, Oasis, Scarr... deviennent decodables ; Live Fire reste hors jeu (module sans `sbsp`) |
| 5 | **Alias « Heavies » sorti** (lot 5) : +43 matchs | piste 4.1 du handoff : FAITE |

## Etape 1 — TESTER L'HYPOTHESE D'HORLOGE (avant tout calcul long)

> Le handoff mettait « elargir le balayage » en tete (~1 h de calcul). Cette etape la rend
> peut-etre inutile : on ne cherche plus un decalage, on en APPLIQUE un connu.

- [ ] 1.1 Etablir sur quelle horloge vivent les DEUX entrees d'`AttributeZones` : les
      trajectoires (artefact de rejeu, frame 0 = `originMs`) et les actions d'objectif
      (`objectiveevents` — verifier sur pieces d'ou vient leur `t`).
- [ ] 1.2 Rejouer `cmd/zone-attribution` sur le corpus en appliquant la correction CONNUE du
      film au lieu du balayage : decalage = origine de l'artefact (et/ou T0 du match selon
      d'ou viennent les actions). Publier le taux AVANT / APRES par film.
- [ ] 1.3 VERDICT : si le taux monte franchement, l'etape 2 (balayage elargi) est ANNULEE et
      la correction devient une donnee, pas une constante ajustee. Sinon, mesurer ce qui
      RESTE comme retard apres correction — c'est cela qu'il faudra expliquer.

Gate 1 : chiffres par film, AVANT/APRES, sur les 8 films du corpus d'origine au minimum.

## Etape 2 — L'ORACLE DE JUSTESSE (Vagabond) — la seule qui repond « est-ce la BONNE zone »

- [ ] 2.1 Rejouer le releve terrain du 2026-08-02 sur Vagabond (3 matchs Strongholds au
      registre) avec `cmd/zone-attribution` : pour chaque action, comparer la zone attribuee
      a la zone RELEVEE a la main.
- [ ] 2.2 Publier une matrice de justesse (bonne zone / mauvaise zone / aucune), pas un
      simple taux d'attribution. C'est le chiffre qui manque a tout le chantier.
- [ ] 2.3 Si la justesse est bonne mais le taux bas : le probleme est la COUVERTURE (on
      n'attribue pas assez), pas la JUSTESSE (ce qu'on attribue est juste) — deux defauts
      qui n'appellent pas les memes suites, et le handoff ne pouvait pas les distinguer.

Gate 2 : la matrice, sur les 3 matchs Vagabond.

## Etape 3 — ELARGIR LE CORPUS (devenu presque gratuit)

- [ ] 3.1 Recompter le corpus mesurable avec le catalogue a 72 cartes et les bornes a 56
      (le handoff comptait 8 films sur un catalogue de 34 cartes / 15 bornes).
- [ ] 3.2 Rejouer la mesure sur ce corpus elargi. A 8 films, un ecart de 3 points ne se
      distinguait pas du bruit ; c'est le seul moyen de sortir de cette zone d'incertitude.

## Etape 4 — DECIDER : persister, ou classer

- [ ] 4.1 Critere de persistance, ecrit AVANT de mesurer (pour ne pas l'ajuster apres) :
      **justesse >= 95 % sur l'oracle** ET **taux d'attribution >= 80 % sur le corpus
      elargi**. En dessous : on ne persiste pas, on ecrit pourquoi, et on classe.
- [ ] 4.2 Si le critere passe : table append-only (recette ADR 0026 : INSERT pur +
      `written_at`, lecture par vue `_latest` UNIQUEMENT, garde-rail anti-ART) alimentee par
      le pipeline, puis exposition. NE PAS improviser le schema : suivre la recette.
- [ ] 4.3 Debouche visuel immediat s'il passe : le calque d'objectifs du rejeu (livre au lot 4
      de la parite) dessine deja les zones — les colorer par equipe TENANTE au fil du temps
      est un branchement, pas une feature nouvelle.

## Hors perimetre (ecrit noir sur blanc)

- **KOTH** (colline mobile) : hors de portee par construction, et deja acte « hors v7.5 ».
- **La lettre A/B/C des zones** : ni la variante de carte ni le film ne la portent — elle
  demande un releve Theater. `Zone.SpatialRank` reste un rang stable, JAMAIS affiche comme
  une lettre du jeu (garde deja documentee).
- **Live Fire** : module sans `sbsp`, pas de bornes, donc pas de trajectoires — hors corpus.
- Toute persistance avant le gate 4.1.

## Ce qui peut faire echouer ce plan (a dire tot)

1. L'hypothese d'horloge est fausse : les actions d'objectif ne vivent pas sur l'horloge que
   `originMs` corrige. On le saura a l'etape 1.1, en lecture de code, avant tout calcul.
2. L'oracle Vagabond est trop petit (3 matchs) pour trancher une justesse a 95 %. Dans ce
   cas, le dire — et ne pas maquiller un intervalle de confiance large en verdict.
3. Le taux reste bas apres correction : alors le defaut n'est pas l'horloge mais la
   PRESENCE des positions (un joueur hors du film, une action sans auteur decodable) — un
   autre chantier, a ouvrir avec ses propres chiffres.
