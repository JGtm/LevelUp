# Investigation — les zones de callout des cartes FORGE : ou sont-elles ?

> Ouverte le 2026-08-14. Affirmation utilisateur, a prendre pour vraie : **« toutes les maps
> en ont, Forge ou non »** — il les voit EN JEU. Le catalogue livre n'en a que pour 22 cartes
> natives ; les cartes communautaires en sont absentes. Elles sont donc AILLEURS, et ce
> document dit ou chercher, dans l'ordre de vraisemblance, avec ce qui est deja ECARTE.

## Ce qui est ETABLI (mesure du 2026-08-14)

| fait | mesure |
|---|---|
| Le tag `levl` des 8 CANEVAS Forge ne porte aucune zone nommee | 0 callout sur 8 canevas (balayage `cmd/mapcallouts-build`) |
| Le catalogue livre couvre 22 cartes natives | 816 zones, libelles FR/EN 816/816 |
| Le `.mvar` porte bien une TABLE DE CHAINES LIBRES | 5 411 entrees sur 58 cartes, 983 distinctes |
| **Mais ces chaines ne sont PAS des callouts** | ce sont des noms d'objets/prefabs du createur : `Prefab`, `ai_move_zone_*`, `fo_ai_zone_*`, `! CATALYST WATERFALL WIP`, `0- Podium`... — noms de TRAVAIL, pas des lieux prononcables |
| Le `.mvar` porte des labels NON RESOLUS | **92 hashes distincts**, les plus frequents a 921 / 536 / 523 occurrences |
| Le jeu a une icone `callout` | `static/weapons-assets/halo_infinite/jeu/`, index 54 du kill feed |

## Piste 1 — L'OBJET FORGE « callout » (la plus vraisemblable)

Dans Forge, le createur POSE un objet de zone nommee et en choisit le libelle **dans une
liste predefinie** (c'est ce qui permet au jeu de l'annoncer et de le localiser — une saisie
libre ne serait ni prononcable ni traduite). Consequence : le nom ne serait PAS une chaine du
`.mvar` mais un **identifiant** — exactement la forme des 92 hashes non resolus.

Methode, DEJA EPROUVEE deux fois sur ce chantier (volumes de mort Forge, modeles d'objets) :
- [ ] 1.1 `type_id` du `.mvar` -> tag `food` de `forge_objects-rtx-new.module` (resolution
      20/20 mesuree au lot 5 cartes). Chercher le ou les type_id dont le tag porte « callout »
      / « named location » / « zone ».
- [ ] 1.2 Empreinte par frequence : un objet callout doit apparaitre PLUSIEURS FOIS par carte
      et sur BEAUCOUP de cartes (comme les volumes de mort : « 1 par carte sur 61 »). Trier
      les type_id par (nb de cartes portantes x occurrences) et regarder les candidats.
- [ ] 1.3 Si un type_id sort : lire ses champs (le decodeur `mapvar` lit deja pos/forward/
      shape/labels) et voir ou vit le libelle — label hashe ? champ dedie ? index dans une
      table du canevas ?

## Piste 2 — Resoudre les 92 hashes (peu couteuse, a faire de toute facon)

- [ ] 2.1 Meme mecanique que la resolution des libelles de mode (murmur3 x86-32 du nom
      snake_case) : tester une liste de candidats — `callout`, `callout_zone`, `named_area`,
      `named_location`, `nav_marker`, `zone_name`, `location`, `area_name`, + les noms de
      lieux generiques du jeu (`base`, `mid`, `tower`, `ramp`, `courtyard`...).
- [ ] 2.2 Les hashes les plus frequents (921, 536, 523 occurrences) sont les plus rentables :
      un label present des centaines de fois est structurel, pas anecdotique.

## Piste 3 — Un catalogue GENERIQUE cote canevas ou globals

- [ ] 3.1 Si les cartes Forge partagent des callouts generiques (« Base rouge », « Milieu »,
      une grille), ils vivraient dans un tag du CANEVAS ou dans les `globals` — pas dans la
      variante. Verifier : le tag `levl` du canevas a-t-il un bloc `volumes` NON VIDE avec des
      volumes **sans** `named location` (on sait deja que ce bloc existe et qu'il contient
      plus d'entrees que de noms) ?
- [ ] 3.2 Ce serait coherent avec une observation deja au dossier : sur les cartes natives,
      des volumes non nommes existent (« 1 m2, des marqueurs » sauf va_behemoth). Les
      re-regarder sur un CANEVAS, ou la question n'a jamais ete posee.

## Ce qui est DEJA ECARTE (ne pas y revenir sans element neuf)

- La table de chaines du `.mvar` : mesuree, ce sont des noms d'objets de travail.
- Le catalogue d'objectifs (`map_objectives.json`) : il ne garde QUE les objets porteurs d'un
  ROLE d'objectif — un objet callout y serait filtre a l'entree. Ne pas conclure de son
  silence.

## Garde-fous

- **Aucun nom devine** : un hash non resolu reste un hash. La regle du chantier (icone d'une
  autre arme, lettre A/B/C inventee) vaut ici mot pour mot.
- Offline-pur et universel : pas de Cheat Engine, pas de capture runtime.
- Si les callouts Forge sont **choisis dans une liste**, la liste elle-meme est une donnee du
  jeu a extraire une fois (comme `uslg` pour les 816 libelles natifs) — pas a retaper.

## Pourquoi c'est essentiel (dit par l'utilisateur, et confirme par le chantier)

Les callouts sont le seul nommage de lieu que le jeu donne. Sans eux, une carte Forge —
c'est-a-dire la MAJORITE des cartes jouees — n'a ni zone nommee au rejeu, ni « ou »
descriptible dans une stat. Et ils sont aussi le candidat le plus serieux pour delimiter la
ZONE JOUABLE, la ou sept criteres geometriques ont ete refutes.
