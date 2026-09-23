# Arbitrage de la dette consignée — fin de la vague 6 (2026-09-11)

> Document d'aide à la décision, écrit par le superviseur de la campagne v7.5 pour l'utilisateur.
> Chaque ligne est un item consigné (rapports de lot, registre, revues), jamais traité, avec ce
> qu'il change pour l'utilisateur final, ce qu'il coûte, et une recommandation. L'utilisateur
> tranche ; un agent planifiera ensuite les items retenus (ordre, lots, gates). Rien ici n'est
> engagé.
>
> Échelles. **Gain** = ce que l'utilisateur final y gagne en fiabilité ou qualité des données
> rendues (fort / moyen / faible / nul), avec la population concernée. **Effort** = XS (< 1 h
> d'agent), S (une demi-journée), M (une journée, un lot), L (plusieurs jours, un chantier),
> « + recuisson » quand les artefacts changent (22 min de coupure, tout le parc). **Reco** =
> OUI (à faire), OUI-XS (à grouper dans un lot d'hygiène), PLUS TARD (condition écrite), NON.

## 1. Ce qui change les données rendues à l'utilisateur

| # | Item | Source | Gain utilisateur | Effort | Reco |
|---|---|---|---|---|---|
| D1 | **Temps d'occupation de zone par joueur** (Strongholds, Total Control, KOTH) : calque inexistant, 3 507 s à l'oracle sur 6 films, 0 publiée | audit §4.2, L9 ; lot 6.9 décidé | **Fort** : un indicateur tactique entier sur trois modes (qui tient les zones, combien de temps), confrontable à l'oracle joueur par joueur | L + recuisson | **OUI** — lot 6.9, déjà décidé ; premier de la liste après la release |
| D2 | **Propriétaire des armes apparues sans lâcheur** : le point 6 du record de création nomme le propriétaire de 92,5 % des armes aujourd'hui « joueur non nommé » (21,4 % des objets du calque) | rapport 6.6 découverte 2 | **Moyen** : l'infobulle des armes au sol nomme qui a apparu/lâché l'arme dans un cas sur cinq de plus ; même oracle que 6.6 | S + recuisson | **OUI** — à coupler à la prochaine recuisson (avec D1 ou D4) |
| D3 | **Fonds de carte pour les cartes Forge** (objets `.mvar` moins volumes de mort) : environ 100 cartes sans fond de carte | reports de fond | **Fort** : un rejeu sans fond de carte est illisible ; ~100 cartes concernées (toutes les Forge, dont Forest où l'on jongle) | L (prototype puis chaîne) | **OUI, après la release** — le plus gros gain visible de cette liste ; commencer par mesurer combien de matchs du parc et de la prod sont sur Forge |
| D4 | **Reliquat drapeau 1** : le seul span `carried_open` du parc est borné à la fin de l'axe du rejeu, pas de la partie jouable (18 s, 1 joueur) | registre « Reliquats du drapeau » | Faible aujourd'hui (1 joueur), mais **classe reproductible** : tout porteur qui a le drapeau en main quand le match finit | S (descendre `playable_duration_seconds` au calque) | **OUI** — au prochain lot drapeau, avec D5 |
| D5 | **Reliquat drapeau 2** : un siège du statborg pour trois occupants successifs (départ, bot, remplaçant) ; le remplaçant perd ses 4,1 s de portage | registre « Reliquats » ; rapport 6.11 §5 | Faible aujourd'hui (1 joueur), mais **classe fréquente** : les remplaçants en cours de match sont courants en arène (B1 en a compté 5 films sur 64) | M (identité par fenêtre d'occupation dans `objectiveevents/`) | **OUI** — même lot que D4 ; c'est la dernière cause d'identité connue |
| D6 | **Reliquat drapeau 3** : le grain du tic (élargir chaque période d'une demi-image aux deux bornes, comme accepté pour le crâne) | audit §3.4 ; rapport 6.13 §7 | Faible : ≤ 1,5 s par joueur, aucun joueur ne dépasse son oracle de plus ; changement de définition | S | **NON** — le drapeau est à 102,8 % ; ce serait de la cosmétique statistique |
| D7 | `cde26226` reste à 1,040 pour une cause non identifiée | rapport 6.13 | Faible : un film, 4 % | M (instruction) | **PLUS TARD** — seulement si un second film montre la même forme |
| D8 | **Neuf films publient des vies sans identité** (`94a28b8b` 5 vies, etc.) ; alarme `SansCandidat` au message inexact ; `b0fe12b1` ne transmet qu'une lecture i48 | dette vague 5 ; recuissons 51 et v6 | **Moyen** : des joueurs anonymes dans le rejeu sur 9 films du parc (14 %), donc en prod aussi | M (instruire un film, puis la classe) | **OUI** — prochain chantier identité, avec D5 ; commencer par `94a28b8b` |
| D9 | **Socle central d'Illusion étiqueté équipe 0 au catalogue** | rapport 6.11 D1, 6.13 | **Moyen** : Illusion est une carte fréquente ; un socle mal attribué peut rattacher un portage au mauvais drapeau | XS (catalogue) + mesure sur les 2 films Illusion | **OUI-XS** |
| D10 | `objectivesLayer.ts` construit un pulse par frag et assistance (15 648 par image sur `8bc6074f`) ; le dénominateur de couverture affiche « 100 % » sur un calque dont 99 % est un seul compteur | audit §12-1 | **Moyen** : fluidité du rejeu sur les longs matchs, et un chiffre de couverture trompeur | S (filtrer les familles, corriger le dénominateur) | **OUI** — petit, visible |
| D11 | `zone_offensive_kills` / `zone_defensive_kills` jamais nommés (comme l'était `flag_secures`) | audit §12-5 | Faible tant qu'aucune page ne les affiche | M (rétro-ingénierie, même protocole que `flag_secures`) | **PLUS TARD** — le jour où une page de zones les affiche (D1) |
| D12 | Huit poses d'appareil de mur sur 295 sont des lâchers à la mort promus `deployed` | dette vague 5 | Faible : 2,7 % des poses de mur | S | **NON** |
| D13 | Champ de réparation : « utilisé » sous-compté (3 → 1 sous us6), les consommations aussi | dette vague 5 | Faible : une famille rare | M | **NON** |
| D14 | Mystère Fiesta / Husky Raid : la moitié des tirs non fatals perdus | reports de fond | Moyen sur la précision par arme en Fiesta, nul sur les kills | L | **PLUS TARD** — après D3 ; à ne rouvrir que si la précision par arme revient au produit |
| D15 | `ListMapsByTitle` dédoublonne par `name_canonical` : deux libellés d'un même asset sortent deux fois (le web dédoublonne) | dette vague 5 | Faible (invisible) | XS | **OUI-XS** |
| D16 | `7fce3219` sans pont par le triplet ; 258 images muettes de `b8a44fe8` (trou de réplication) | rapports B2, 6.13 | Nul : le premier est ponté par ailleurs, le second est rendu « objet libre » depuis B2 | — | **NON** |

## 2. Ce qui protège la fiabilité sans changer une donnée (garde-rails, tests, outillage)

| # | Item | Source | Gain | Effort | Reco |
|---|---|---|---|---|---|
| G1 | Deux tests non déterministes sous charge (`PalmaresRelationsPage`, `mapcatalog` verrou Windows) | dette vague 5 | **Moyen** pour la fiabilité de la CI (un flake = une vague qu'on relit pour rien) | M (preuve de concurrence à la place d'un seuil de durée) | **OUI** — lot flakes |
| G2 | Films sans chunks en cache (`0a247154`, `24dbb67d`, `60ae07c4` nommé au corpus d'équivalence, `92f18088`, `1c4c63c2`, `a349fea8`) | audit, B2, 6.2 | Moyen pour la preuve : un témoin du corpus est illisible sans que rien ne l'annonce | S (re-télécharger les films, ou remplacer le témoin) | **OUI-XS** — au minimum, faire échouer le corpus quand un témoin n'a pas de chunks |
| G3 | `config/replay_corpus.toml` : la raison du témoin Oddball décrit un résidu fermé depuis le 8 septembre | audit §12-6 | Nul | XS | **OUI-XS** |
| G4 | Revue 6.R : invariant `Balanced()` vrai par construction (doc à requalifier) ; `closedBy*` baissent sur les films à `noTrack` (à documenter) | revue §Ronde 2 | Nul | XS (doc) | **OUI-XS** — avec D4/D5 qui touchent ces fichiers |
| G5 | Garde-rail anti-copie à frontière de mot contournable par renommage (`noLocalUsageCopies.guard.test.ts`) | dette vague 5 | Faible | S | **PLUS TARD** |
| G6 | `seed_demo_corpus.go` fait un `UPDATE kill_positions` hors garde-rails ; ADR 0026 ne documente pas `decode_pass` | dette vague 5 | Faible (outil de démo, pas la prod) | S | **OUI-XS** pour l'ADR ; l'UPDATE : allowlister avec justification datée ou migrer |
| G7 | Commentaire de `GroundWeapon.W` inversé (espace de clés) ; liste de chargeurs lue aussi sur l'arme portée en delta (non mesurée) | rapport 6.6 | Nul / faible | XS / M | **OUI-XS** pour le commentaire ; la mesure delta : **NON** (les munitions exactes sont acquises) |
| G8 | Aucun champ de fin de vie sur l'objet arme au sol (dissolver neutre à 99,8 %) | rapport 6.10 bis | Nul (votre intuition confirmée) | — | **NON** — clos |
| G9 | Pont MCP Ghidra sans sockets Unix ; HTTP 8089 fonctionne | rapport 6.10 bis | Nul pour l'utilisateur, utile aux agents | XS (note d'outillage, déjà en mémoire) | **NON** — rien à faire |

## 3. Hygiène de code (aucun effet sur les données)

| # | Item | Gain | Effort | Reco |
|---|---|---|---|---|
| H1 | `MatchMetrics` mort (`internal/domain/stats.go`) ; `decoupe_masque.go` code `.png` en dur ; trois handlers Huma avec empreinte propre | Nul | XS + XS + (non migrable) | **OUI-XS** pour les deux premiers, groupés ; le troisième reste |
| H2 | Fichiers au-dessus de 500 lignes gelés par baseline (`match_view_raw.go`, `ReplayCanvas.tsx`, tests de recherche) ; `equipmentUsageLogic.ts` à 494 | Nul | M à L | **NON** — baseline, ne pas accroître |
| H3 | 30 avertissements eslint gelés (hooks React) | Nul | S à M | **NON** |
| H4 | Outils de recherche du chantier véhicules (`weapon-sounds`, `vs-measure`, `vehicle-sprite`) | Nul | XS | **OUI-XS** — supprimer (git garde l'historique), sauf si l'auteur les veut |
| H5 | Registre : L66 prémisse caduque, L542 close partiellement, ~600 lignes non relues | Nul | S | **PLUS TARD** — à la prochaine campagne |

## 4. Déjà décidé, rappelé pour mémoire (ne pas rediscuter)

- Armes spéciales au grain session (P5) et `deployed_*` : non retenu / conservé (10-09).
- Grand combat, portage BTB, index de participant : non retenu (10-09).
- Zones Total Control : non traitable sur mesure (B2), entrée retirée le 27 août.
- Seuil du rejeu public 88 % : à trancher à la release (Notion).
- Précision par arme : abandon définitif (vague 5).
- Second VPS / ouvrier distant : activation prod du rejeu, à la release.

## 5. Ordre recommandé, si tout ce qui est « OUI » est retenu

1. **Lot d'hygiène XS** (une demi-journée, un agent) : D9, D15, G2 (au minimum le garde), G3, G4, G6 (ADR), G7 (commentaire), H1, H4. Aucune recuisson.
2. **Lot drapeau et identité** (M, un agent) : D4, D5, D8. Recuisson des films touchés.
3. **Lot rejeu web** (S) : D10.
4. **Lot 6.9 zones** (L) : D1, puis D2 dans la même recuisson.
5. **Lot flakes** (M) : G1.
6. **Chantier Forge** (L, après la release, à planifier à part) : D3, puis D14 s'il y a lieu.

Total « OUI » hors chantier Forge : environ quatre jours d'agent et deux recuissons. Le chantier Forge est le seul de taille comparable à une vague.
