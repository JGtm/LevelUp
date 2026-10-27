# Audit des positions de force — reprise du sujet (2026-10-27)

> Audit en lecture seule, demandé par l'utilisateur le 2026-10-27 : « juste un audit nouveau,
> pas de plan ni d'exécution ». Il fait l'état de ce qui a CHANGÉ depuis la clôture du 2026-09-20
> (branche `wt/power-positions`, quatre verdicts NO-GO, document de définition), et de ce que
> ces changements permettent ou non. Il ne décide rien et ne mesure aucune position.
>
> Sources lues : la branche `origin/wt/power-positions` (documents `.ai/V7.5/positions_de_force/`,
> constantes de `internal/analysis/powerpos/geo`), la sauvegarde
> `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb.avant-purge-20261026` (ouverte en
> `read_only`, aucun serveur ne la tient), le cache local de films, et le VPS en lecture seule
> (`ls`, `du`, `df`, aucune ouverture de base).

## 1. Ce qui a changé depuis le 20/09

### 1.1 Le corpus de positions de kills est cinq fois plus gros, et il est classé

La sauvegarde d'avant purge porte **375 096 kills positionnés sur 3 399 matchs** (période
2023-04 → 2026-10-25), contre 74 909 morts sur 951 films au moment de la recherche v1/v2.

| Carte (variante classée incluse) | Matchs | Kills | dont classés | Lobbys de moyenne CSR ≥ 1500 |
|---|---|---|---|---|
| Live Fire | 341 | 41 794 | 288 | 121 |
| Recharge | 354 | 40 486 | 299 | 113 |
| Streets | 241 | 26 450 | 185 | 89 |
| Aquarius | 217 | 24 139 | 169 | 73 |
| Lattice | 159 | 20 245 | 155 | 66 |
| Vacancy | 147 | 19 330 | 144 | 73 |
| Solitude | 161 | 15 672 | 146 | 63 |
| Origin | 150 | 14 449 | 119 | 62 |
| Empyrean | 95 | 10 547 | 74 | 33 |
| Fortress | 88 | 8 794 | 68 | 15 |

Comparaison v1 (même carte) : Live Fire 80 matchs / 7 861 kills → 341 / 41 794 ; Recharge
78 / 7 344 → 354 / 40 486 ; Lattice 28 / 3 716 → 159 / 20 245.

### 1.2 Le « corpus d'élite » existe déjà en grande partie

Le document de définition posait le mur ainsi : « nos joueurs suivis, sauf un, ne tiennent pas
les positions pro » et proposait de construire un corpus Onyx en téléchargeant des films de tiers.
Mesure du jour :

- 1 798 des matchs à positions de kills ont un CSR par joueur (`match_csrs`, 13 219 identités
  distinctes : ce sont les CSR de TOUS les joueurs du lobby, pas seulement des joueurs suivis) ;
- CSR moyen de lobby : médiane **1 474** ; **782 lobbys** à moyenne ≥ 1500, **1 730** ≥ 1300 ;
- **51 % des matchs** à positions de kills sont des matchs de Nuzzles (CSR médian 1 502,
  max 1 756) ;
- `match_participants` porte aussi, pour ~72 % des participants, `kills_expected`,
  `deaths_expected`, `team_mmr`, `enemy_mmr` : une attente de performance PAR JOUEUR et par
  match, fournie par l'API.

Conséquence : le terme « Confirmation » (couche 3 de la définition) n'exige plus de construire un
corpus de tiers pour les cartes classées ; il est en grande partie dans la sauvegarde. Il reste à
vérifier que ce corpus n'est pas DOMINÉ par un seul joueur (un lobby sur deux contient Nuzzles :
ses habitudes de placement pèseraient lourd dans toute mesure non corrigée).

### 1.3 Mais ce corpus a été purgé en local le 26/10

La purge locale du 2026-10-26 (Nuzzles, calloussss, Brooks SVP) a retiré 8 024 matchs de la base
partagée et **2 624 films, 2 137 fichiers de rejeu et 2 013 faits** du cache. État croisé :

| Ressource | Matchs de Nuzzles (1 742) | Autres matchs (1 657) |
|---|---|---|
| Positions de kills | sauvegarde locale seulement | sauvegarde + base courante |
| Film en local | **0** | 1 415 |
| Film au VPS | 11 | 7 |
| Rejeu (trajectoires) en local | 0 | ~1 300 (presque tous non classés) |

Aucun lobby Onyx n'a aujourd'hui de rejeu local. Les positions de kills des lobbys Onyx sont
utilisables tout de suite ; la DENSITÉ D'OCCUPATION (temps passé à chaque endroit, le signal que
l'utilisateur jugeait le bon le 20/09) ne l'est pas sans re-télécharger et re-cuire leurs films.

### 1.4 Les films anciens se re-téléchargent

Le VPS a téléchargé 503 films les 26 et 27/10 (15 Go, disque à 85 %) ; ils portent sur des matchs
de 2024-10 à 2026-10. Les films de deux ans restent donc servis par le jeu : le re-téléchargement
des films de Nuzzles est plausible (non vérifié sur ses matchs précis). Coût d'ordre de grandeur :
le cache local fait 36 Go pour 1 422 films (~25 Mo par film) → ~700 lobbys Onyx des cartes
classées ≈ 18 Go, plus la cuisson (règle en vigueur : jamais de cuisson en lot sans accord, pic
mémoire).

À noter : la table `match_player_positions` (positions échantillonnées par une passe dédiée, 235
matchs seulement) est un chemin plus léger que le rejeu complet pour l'occupation ; son coût et sa
fréquence d'échantillonnage sont à regarder avant de choisir.

### 1.5 Le décalage de Live Fire - Ranked est résolu

Cause d'exclusion de 3 854 kills en v2. Aujourd'hui, médianes et quantiles 5-95 % des positions
de kills coïncident à une unité près entre carte de base et variante classée (Live Fire, Recharge,
Streets). Les variantes classées peuvent entrer dans la mesure.

## 2. Un doute d'échelle qui touche TOUTES les lignées

Les coordonnées ont été traitées comme des mètres dans toute la recherche (disque de 2 m, grille
de 0,5 m, hauteur des yeux 1,7 m, hauteur libre 1,5 m, positions attestées de 5 à 30 m²). Plusieurs
mesures du jour sont difficiles à concilier avec des mètres :

| Mesure (sauvegarde) | Valeur brute | Lue en mètres | Lue en unités Halo (~3,05 m) |
|---|---|---|---|
| Distance tueur-victime, mêlée silencieuse, médiane | 0,3 | 30 cm entre deux centres de corps | ~0,9 m |
| Distance des kills à la tête, médiane / p90 | 7,4 / 16,3 | 7 m / 16 m | ~23 m / ~50 m |
| Étendue de Live Fire (p1-p99 des tueurs) | 34 × 30 × 4,7 | 34 × 30 m, 4,7 m de haut | ~105 × 92 m, ~14 m de haut |

Contre-indice : la hauteur de saut mesurée au lot 5.7.5 vaut 0,85, plausible en mètres, haute en
unités (~2,6 m). Ce n'est pas un étalon extérieur : c'est notre propre mesure, dont l'unité
« mètre » a été supposée.

**Tranché le 2026-10-27 (complément, même jour) : les coordonnées sont des UNITÉS MONDE Halo
(1 unité = 3,048 m).** Trois témoins indépendants, tous dans le même repère que les positions :

1. **Collision entre Spartans** (cinq rejeux d'arène locaux, paires de joueurs au même niveau,
   |dz| < 0,3) : séparation horizontale au 0,1e centile **0,26 à 0,48**, au 1er centile 0,48 à
   0,95. Deux corps ne descendent pas sous un diamètre de collision : 0,3 m est impossible pour
   un Spartan, 0,3 unité = 0,9 m est sa carrure.
2. **Palette Forge** (`.ai/archive/V7.5/ETAT_DE_L_ART_FORGE_PALETTE_ZONES.md`, 2026-08) : les
   emprises d'objets sont en unités monde ×3,048 (Warthog 6,8 × 3,1 × 2,5 m, Pelican 32,7 m), et
   ces objets se posent aux positions du `.mvar`, dans le repère des positions du film. Le même
   document affirme les positions « en mètres » : affirmation non mesurée, contredite par les
   témoins 1 et 3.
3. **Mêlée et étendues** : distance tueur-victime de la mêlée silencieuse 0,3 (0,9 m en unités) ;
   Live Fire 34 × 30 × 4,7 (105 × 92 × 14 m en unités).

Conséquences :
- le disque de « 2 m » de la v1 couvrait ~6 m de rayon (~110 m²), et les « 5-30 m² » de l'oracle
  géométrique étaient ~45-280 m² ;
- la géométrie lançait ses rayons de vision à ~5 m au-dessus du sol (`HauteurYeuxM` 1,7) et
  exigeait ~4,6 m de hauteur libre (`HauteurLibreM` 1,5) : la vision était surestimée (rayons
  par-dessus les murets) et des sols praticables rejetés — ce qui est cohérent avec l'observation
  du 20/09 « une corniche à 1,6-2,0 au-dessus des marches les rejetait » ;
- le verdict « bruit » de la v1 sur la case isolée reste vrai (c'est une affaire de comptes, pas
  d'unité), mais les largeurs de zone essayées et la lignée géométrique sont à relire.

Hors périmètre de cet audit, noté seulement : la constante `SpartanJumpHeightM` du décodeur porte
le même présupposé dans son nom.

## 3. La piste « K/D par coordonnées » au vu des nouvelles données

Déjà testée (v1, 20/09) : rapport de duel lissé sur disque, rétréci vers 0,5. Deux constats
tenaient alors : (a) case par case, dispersion d'une pièce équilibrée ; (b) lissé, le score
retrouve bases et points de défense, pas les positions des pros.

Ce que le nouveau corpus change :
1. **La largeur n'est plus le verrou principal.** Avec ~5 fois plus de kills, le même nombre
   d'engagements par zone s'obtient sur un disque de rayon ~2,3 fois plus petit. Sur les seuls
   lobbys Onyx (~40 % des matchs classés d'une carte), la zone nécessaire pour ~100 engagements
   retombe de l'ordre d'une position, plus d'une pièce. La largeur se choisit par reproductibilité
   (deux moitiés de matchs tirées au sort doivent donner la même carte), pas à la main.
2. **Le verrou réel est la confusion.** Le K/D d'un lieu mesure qui s'y trouve (Nuzzles dans un
   lobby sur deux) et ce qui s'y passe (défense de base, kills à la réapparition), pas le lieu. Les
   nouvelles colonnes permettent de corriger le « qui » : CSR de chacun des joueurs du lobby,
   `kills_expected` / `deaths_expected`. Un modèle de duel où chaque kill oppose deux positions et
   deux niveaux connus isole l'avantage propre à l'emplacement. Le « ce qui s'y passe » demande de
   séparer par mode (Assassin / objectif) et d'exclure les morts proches d'une réapparition.
3. **Limite de principe, inchangée** : le K/D ne voit que les lieux DISPUTÉS. Une position qui
   dissuade (on n'y monte pas parce qu'elle est tenue) produit peu de kills. C'est pourquoi la
   densité d'occupation reste le signal complémentaire, et pourquoi le §1.3 compte.

## 4. Ce qui n'a pas changé

- **Le juge.** L'oracle pro v2 : 28 positions fortes sur 8 cartes, dont plusieurs cartes à 2
  fortes (rappel de 0,5 = indéterminé). Vacancy (147 matchs), Interference, Serenity et Argyle
  sont désormais bien fournies mais sans oracle. Tout verdict GO reste plafonné par ce juge.
- **L'absence de maillage de navigation natif** (cartes natives) et de collision décodée : la
  géométrie reste un sol dérivé du rendu.
- **Le socle de méthode** du document de définition (géométrie qui classe, valeur du mode qui dit
  pour quelle tâche, confirmation par les bons joueurs ; sélection relative, sans plancher absolu
  ni plafond de positions).

## 5. État de l'outillage

- `wt/power-positions` part de `feat/v75` au 2026-09-20 (`89982c830`) ; `feat/v75` a été retirée
  le 22/10 et l'historique réécrit. La branche diffère de `main` sur ~7 500 fichiers : l'outil
  (`cmd/mappower-build`, `internal/analysis/powerpos`) est à re-poser sur une base récente avant
  de pouvoir tourner ; il n'a pas été compilé dans cet audit.
- Les mesures v1/v2 lisaient la base par `OpenReadForQuery` sur un `data/` désigné par
  `--data-root` : le même mécanisme peut pointer sur un dossier autonome contenant une COPIE de la
  sauvegarde, sans toucher au `data/` principal ni au serveur local.

## 6. Données : sauvegarde locale ou VPS

| | Sauvegarde locale `avant-purge-20261026` | Base du VPS |
|---|---|---|
| Contenu | 10 026 matchs, gelée au 25/10 | sur-ensemble vivant (Nuzzles toujours suivi, 2,58 Go) |
| Accès | lecture immédiate, aucun processus ne la tient | tenue en écriture par le serveur de prod : une copie à chaud d'un fichier DuckDB ouvert peut être incohérente ; passer par un instantané (restic) ou un export |
| Films | 0 pour Nuzzles | 11 pour Nuzzles (cache sans éviction, disque à 85 %) |
| Risque | la note du 26/10 prévoit de SUPPRIMER cette sauvegarde après vérification de l'appli | toute opération lourde sur un VPS de 2 Go de RAM |

Lecture : pour les positions de kills et les CSR, la sauvegarde locale suffit et ne coûte rien ;
le VPS n'apporte que deux jours de matchs en plus. **La sauvegarde est aujourd'hui la seule copie
locale de ce corpus ; sa suppression prévue le ferait disparaître du poste** (il resterait au
VPS). Pour les films de Nuzzles, ni l'un ni l'autre ne suffit : il faudrait les re-télécharger.

## 7. Décisions de l'utilisateur (message du 2026-10-27)

1. **Dossier de recherche autonome** : « extraire un dossier de recherche autonome, au besoin on
   peut retélécharger les matchs et cuire les rejeux nécessaires pour obtenir un max de
   données ». Élargir aussi calloussss et Brooks SVP au-delà de 2026 (la purge locale les avait
   limités aux matchs de 2026). **Fait** : `C:\Users\Guillaume\Downloads\Scripts\LevelUp-recherche-positions\`
   (copie vérifiée de la sauvegarde, README). Rien re-téléchargé à ce stade.
2. **Lever le doute d'échelle** : oui. **Fait** (§2) : unités monde, ×3,048.
3. **Poids de Nuzzles** : accepté, en notant que l'élargissement du point 1 change la répartition.
4. **Films des lobbys Onyx** : oui, re-téléchargement et cuisson (cf. point 1).

Données complémentaires mesurées dans la copie de recherche :

| Joueur | Matchs (sauvegarde) | dont 2025 | Matchs classés avec CSR | CSR médian | CSR max |
|---|---|---|---|---|---|
| Nuzzles | 7 274 | — | 4 149 | 1 502 | 1 756 |
| calloussss | 325 | 15 | 285 | **1 686** | **1 805** |
| Brooks SVP | 425 | 10 | 342 | 1 464 | 1 548 |

calloussss est le joueur au CSR le plus haut des trois : l'élargissement de son historique est
le gain le plus direct pour le corpus de bons joueurs.

Plan qui en découle (projet, non validé, aucune exécution) :
`.ai/PLAN_POSITIONS_DE_FORCE_V3_2026-10-27.md`. Réservoir mesuré au passage : 2 991 matchs
classés de la sauvegarde n'ont jamais eu de positions de kills (1 474 de 2021-2023, 583 lobbys
Onyx) ; Live Fire 871 classés dont 288 décodés.

Reste à décider avant exécution (relève du plan, pas de cet audit) : la voie de
re-téléchargement pour des joueurs SANS profil suivi (la règle « aucun sync sans profil suivi »,
ADR 0035, s'applique à l'instance LevelUp ; le dossier de recherche n'en est pas une), la
branche et le worktree du code de recherche, le budget disque et mémoire de la cuisson.
