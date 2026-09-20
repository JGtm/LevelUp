# ORACLE PRO — Positions de force des cartes du circuit HCS (Halo Infinite)

> Date : 2026-09-20. Etape 0 de `.ai/PLAN_POSITIONS_DE_FORCE_2026-09-20.md`.
> Branche `wt/power-positions`, worktree `LevelUp-wt-power-positions`.
>
> Ce document est un ORACLE DE VALIDATION pour l'etape 2 (rappel / precision de l'algorithme
> empirique). **Ce n'est pas une source de donnees** : rien ici n'a ete mesure sur nos matchs.
> Tout y est une affirmation de tiers (guides, wikis, presse esport, posts officiels 343/Halo
> Studios), citee par URL.
>
> Perimetre decide par le pilote : TOUT le circuit HCS, sans filtrage prealable par le corpus.
> Le recensement du corpus (item 0.1) est fait par l'agent de l'etape 1 et filtrera ensuite.

---

## 1. Methode

### 1.1 Perimetre et selection des cartes

Le brief demande de couvrir le circuit competitif HCS complet, saisons 2023 a 2025 et pool
courant. Les pools officiels utilises comme reference de perimetre :

- HCS Split 2 (2022) — CTF : Aquarius, Bazaar, Catalyst ; KotH / Oddball / Strongholds :
  Live Fire, Recharge, Streets ; Slayer : Aquarius, Catalyst, Live Fire, Recharge, Streets.
  Source : https://www.halowaypoint.com/news/hcs-split-2-maps-and-modes
- HCS Year 2 — ajout d'Argyle et d'Empyrean (CTF), Empyrean en Slayer.
  Source : https://www.halowaypoint.com/news/hcs-year-2-maps-modes-and-settings
- Entrees recentes en Ranked / HCS : Interference (avril), Fortress, Lattice, Origin,
  Solitude, Banished Narrows, Forbidden.
  Sources : https://www.halowaypoint.com/news/interference-update ·
  https://www.halowaypoint.com/news/ranked-hcs-update ·
  https://www.halowaypoint.com/news/nov-ranked-updates

Cartes reellement traitees : **16 recherchees, 12 avec au moins une position citee**
(Aquarius, Live Fire, Recharge, Streets, Bazaar, Catalyst, Forbidden, Argyle, Empyrean,
Solitude, Interference, Banished Narrows) ; 3 sans position exploitable (Fortress, Origin,
Lattice) ; Illusion reperee mais laissee de cote (vocabulaire inrattachable, voir section 5).

### 1.2 Vocabulaire : d'ou viennent les noms de zones

Les noms EN officiels viennent du depot principal (lecture seule) :
`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/reference/`

- `callouts_i18n.csv` — 816 lignes, colonne `en` (+ `fr`), clef de carte en colonne 1.
- `map_callouts.json` — deux espaces de clefs : `maps` (module installe, 22 cartes natives)
  et `maps_by_id` (map_id UGC, 64 cartes Forge).
- `map_fond_reglages.json` — **c'est ce fichier qui donne la correspondance nom affiche →
  clef de carte** (champ `carte`, 107 entrees). C'est la seule table de ce type du depot ;
  elle a servi a etablir le tableau 1.3 et evite de deviner (`sgh_blueprint` n'est PAS
  Live Fire, c'est Recharge).

Regle appliquee, sans exception : **aucun nom devine**. Quand un guide emploie un nom
communautaire absent du vocabulaire, il reste tel quel dans la colonne « nom du guide » et la
zone officielle vaut `?`. Quand le rattachement est lexical mais non litteral (« high-up
battlement » → `Rampart`), il est fait mais signale par la mention `(rattachement lexical)`
dans la colonne « nom du guide », et recapitule en section 5.

### 1.3 Correspondance nom de carte ↔ clef du depot (verifiee sur pieces)

| Carte HCS | `carte_cle` du depot | Zones nommees connues | Remarque |
|---|---|---|---|
| Aquarius | `ctf_aquarius` | 14 | native |
| Live Fire | `sgh_interlock` | 24 | native — memes callouts que `academy_tutorial` |
| Recharge | `sgh_blueprint` | 18 | native |
| Streets | `sgh_streets` | 28 | native |
| Bazaar | `ctf_bazaar` | 23 | native |
| Catalyst | `catalyst` | 17 | native |
| Forbidden | `ctf_forbidden` | 58 | native, miroir `Dried` / `Overgrown` |
| Illusion | `ctf_illusion` | 43 | native, miroir `Camo` / `Overshield` |
| Prism | `sgh_crystalcaves` | 31 | native |
| Cliffhanger | `ridgeline` | 16 | native |
| Argyle | `dd600260-d91c-4d77-9990-3f35873c90a1` | **5** | Forge — vocabulaire quasi absent |
| Empyrean | `d035fc3e-f298-4c14-9487-465be2e1dc1f` | 17 | Forge |
| Solitude | `f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42` | 36 | Forge (variante « Ranked » : `4a5e5612-…`, memes zones) |
| Interference | `654dff62-d618-496a-8914-06ab73d991e3` | 29 | Forge |
| Banished Narrows | `9ad226d8-8947-4c5b-95bc-d220187698c1` | 31 | Forge |
| Fortress | `0d1c9255-d912-416c-befc-5f3e5e176df2` | **0** | Forge — AUCUNE zone dans le depot |
| Origin | `b302eb62-da9a-480b-a409-3c89df8c1a04` | **0** | Forge — AUCUNE zone (variante Ranked `46a8319c-…` idem) |
| Lattice | `1a6cfc2e-ec86-48e1-9464-1ce1bff6ed48` | **0** | Forge — AUCUNE zone |

### 1.4 Sources consultees, et comment la confiance a ete tranchee

Environ 70 URLs ouvertes (moi + trois agents de recherche web travaillant en parallele sur
des lots de cartes disjoints). Typologie de ce qui a servi :

- **Wikis** : halopedia.org (pages de carte, sections `Layout` / `Callouts` / `Strategies`,
  y compris le wikitexte brut `?action=raw` quand le rendu perdait du texte).
- **Guides ecrits** : gamerguides.com, gfinityesports.com, dotesports.com,
  progameguides.com, thegamer.com, gamerant.com, ggrecon.com, gamersdecide.com.
- **Officiel** : halowaypoint.com (posts de pool HCS, previews de carte, notes de conception).
- **Presse / conception** : kotaku.com (interview des level designers de la saison 5),
  gamedeveloper.com (article du level designer de Plaza, carte source de Solitude).
- **Analyse competitive** : thegamehaus.com (« The Playbook », analyse de setup Slayer sur
  The Rig, carte source d'Interference) — la seule veritable analyse de niveau pro trouvee
  en texte indexable.

**Regle de confiance appliquee** :

- `forte` = au moins **2 sources independantes** (editeurs differents) qui nomment la MEME
  zone et lui donnent un role compatible (hauteur, lignes de vue, arme, objectif, acces).
- `faible` = une seule source ; OU rattachement de zone non atteste ; OU affirmation
  transferee depuis la carte d'origine d'un remake (`⁂` ci-dessous).

**Transferts depuis la carte d'origine** (`⁂`). Trois cartes du pool sont des remakes et
n'ont AUCUNE source tactique propre : Empyrean ≈ The Pit (Halo 3), Solitude ≈ Plaza (Halo 5),
Interference = The Rig (Halo 5), Banished Narrows ≈ Narrows (Halo 3). Les positions issues de
la carte d'origine sont marquees et **restent en confiance faible** : le layout est atteste
proche par une source, mais l'equilibrage a change (Interference remplace l'Active Camo par
l'Overshield ; Empyrean remplace l'epee par le Heatwave et le camo central par le Rocket
Launcher). Ce sont des hypotheses, pas des faits mesures sur Halo Infinite.

**Sources ecartees, et pourquoi** :

- `playhaloinfinite.com` — contenu manifestement synthetique. Il attribue a des joueurs pro
  des citations inverifiables et produit des statistiques precises sans source (« 71 %
  round-win rate », « 92 % of the time », « Snakebite stated… »), et il place sur Streets un
  « Blue Room » qui n'existe dans aucun vocabulaire de la carte. **Non cite, non compte dans
  la confiance.** Il est mentionne ici uniquement pour que la prochaine session ne le
  redecouvre pas comme une source.
- `earlyguides.com` — meme regime, ecarte sans ouverture.

**Sources lues via l'index de recherche faute d'acces direct** : dotesports.com,
progameguides.com et thegamer.com/camping-spots renvoient 403/404 au telechargement direct ;
leurs citations proviennent des extraits servis par la recherche. Fidele au texte, mais le
contexte de la phrase n'a pas pu etre controle. Signale ligne par ligne par `*(extrait)*`.

**Sources inaccessibles a l'outillage** (angle mort majeur, voir section 5) : reddit.com
(r/CompetitiveHalo) est refuse par le user-agent ; liquipedia.net renvoie 403 sur les pages
HTML ; halo.fandom.com renvoie 402 ; YouTube ne sert ni description ni transcription.

**Verification anti-invention** : chaque agent devait fournir une citation verbatim par
ligne. J'ai re-telecharge moi-meme une source par agent (halopedia Argyle `?action=raw`,
thegamehaus The Rig, gfinity Recharge) : les trois citations controlees sont exactes.

---

## 2. Positions de force, carte par carte

Legende des raisons : `hauteur`, `lignes de vue`, `arme` (arme forte ou power-up a portee),
`objectif`, `spawn`, `acces` (peu d'entrees / facile a defendre).

### 2.1 Aquarius — `ctf_aquarius`

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| Top Mid | « Top Mid » | hauteur (sommet du mur central qui divise la carte) + power-up Camo/Overshield + lignes de vue sur le couloir central | forte | halopedia, gamerguides, gfinity, thegamer |
| Hydro | « Hydro » | arme : Stalker Rifle / Shock Rifle, engagements longue portee | forte | gfinity, gamerguides |
| Planters | « Planters area » | arme : fusil a pompe, bas du centre | faible | gamerguides |
| Yellow Base | « Yellow Base » | objectif (drapeau) + spawn ; deux sorties vers Utility et Refrigeration | faible | gamerguides |
| Blue Base | « Blue Base » | objectif (drapeau) + spawn | faible | gamerguides |

Citations d'appui : « a stairway on the right half of it that leads to the top of the central
wall that divides the map » (halopedia) ; « The Power Item that spawns in Top Mid! It'll
either be Active Camo or Overshield » (gfinity) ; « these two weapons can drastically improve
your time-to-kill at all ranges » a propos du Stalker/Shock Rifle en Hydro (gfinity) ; « a
central lane that has various vertical positions to occupy, and a power weapon spawn in the
middle » (thegamer).

**Contre-exemple cite** : rester pres du drapeau dans sa propre base — « don't stand near the
flag for too long as the enemy team can easily pick you off from their base » (gamerguides).

### 2.2 Live Fire — `sgh_interlock`

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| Tower | « Tower » | hauteur ; **peu d'acces** (un seul escalier) ; surveille Hallway et Overlook ; sniper a proximite | forte | gfinity, thegamer (camping spots), gamerant |
| Hallway | « Hallway » / « center of the map » | arme forte : S7 Sniper ou Skewer, au centre | forte | gfinity, gamerguides, gamerant |
| Platform | « cylindrical platform » / « Overshield platform » (rattachement lexical) | power-up Overshield / Active Camo | forte | gamerguides, gamerant |
| Overlook | « Overlook » | lignes de vue depuis / vers Tower ; spawn de pistolet | faible | gfinity |
| House Window | « House Window » | objectif (Oddball) : le porteur y frappe les entrants | faible | gamerguides |
| House Elbow | « House Elbow » | acces a couvrir quand l'equipe tient House | faible | gamerguides |
| Tunnel | « Tunnel » | couvert pour regenerer les boucliers (Slayer) | faible | gamerguides |
| House | « House » | couvert pour regenerer les boucliers (Slayer) | faible | gamerguides |

Citations d'appui : « It's often going to be a tough ask to take them out from below since the
high ground is a massive advantage » (gfinity, a propos de Tower) ; « There is only the
stairwell behind it to reach the top, which can be easily guarded » (thegamer) ; « At the
center of the map, either the Sniper or Skewer will spawn at the start of the match »
(gamerant) ; « a cylindrical platform with the Overshield on top » (gamerguides).

**Contre-exemple cite** : le point C en Strongholds, « on the opposite side of the map » des
points A et B, donc difficile a tenir avec les deux autres (gamerguides). Et « Don't fall in
the Canal! » (gfinity).

### 2.3 Recharge — `sgh_blueprint`

C'est la carte la mieux documentee du lot.

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| Attic | « Attic » | hauteur : tir plongeant sur Batteries, Bridge, Pit, Storage et Overhang | forte | gfinity, dotesports |
| Maintenance Bay | « Maintenance Bay » | hauteur (partie interieure d'Attic), couvre l'essentiel de la carte ; « controler ce point fait gagner des Slayer » | forte | gfinity, dotesports |
| Pit | « Pit » | power-up Active Camo / Overshield, au centre | forte | gamerguides, gfinity |
| Hydro | « Hydro » | arme forte (Energy Sword / Gravity Hammer) + objectif (drapeau C) + spawn | forte | gamerguides, gfinity, dotesports |
| Whirlpool Dam | « Whirlpool » | hauteur couvrant la plupart des entrees de Batteries ; armes Needler / Sentinel Beam entre les deux portes | forte | dotesports, gfinity |
| Orange Pipes | « Orange Pipes » | acces : deux entrees faciles a defendre, donne sur Attic ; arme (Mangler) | forte | dotesports, gfinity |
| Control Room | « Control Room's second floor » | hauteur : surplombe Pit et Batteries, excellente vue sur Attic | faible | dotesports |
| Elevator | « top of Elevator » | hauteur + objectif (drapeau A, Oddball) + peu d'acces | faible | dotesports |
| Batteries | « Batteries » | objectif (drapeau B), partie haute du centre | faible | dotesports |
| Long Hall | « Long Hall » | arme (Mangler) ; relie Whirlpool Dam et Orange Pipes | faible | gfinity |
| Overhang | « Overhang » | arme (Mangler) | faible | gfinity |
| Blue Pipes | « Blue Pipes » | itineraire Camo → epee (« combinaison presque imparable ») | faible | dotesports |
| Platform | « Hydro platform » (rattachement lexical) | hauteur donnant des angles sur la majorite de la carte | faible | hyperxarenalasvegas |

Citations d'appui : « At Attic and Maintenance Bay, for example, you can easily rain fire over
everyone in Batteries, Bridge, Pit, Storage, and even up at Overhang » (gfinity) ;
« Maintenance Bay is also the interior portion of Attic and is a great spot to cover most of
the map » et « consistently controlling this spot can help you win Slayer matches »
(dotesports) ; « The Spawn Point for the Active Camo or the Overshield is located in the Pit
which is the center of the map » (gamerguides).

**Contre-exemples cites** : Batteries, Bridge, Pit, Storage et Overhang sont explicitement
decrits comme les CIBLES du tir plongeant d'Attic ; les flancs bas Pit / Control Room
permettent de contourner le drapeau A « with ease » (dotesports) ; « power weapon camping
occurs in long hallways » (hyperx).

### 2.4 Streets — `sgh_streets`

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| Main Street | « middle / center of Main Street » | arme forte : M41 SPNKr ou Cindershot au centre ; « whoever gains control of the power weapon is at a strong advantage » | forte | gfinity, progameguides |
| Subway Balcony | « a balcony » / « Subway Balcony » | lignes de vue (poste de marquage) + arme (Shock / Stalker Rifle) | forte | gamerguides, progameguides |
| Subway | « the Subway » | acces : on sait d'ou l'ennemi peut venir ; repli sur | faible | gamerguides |
| Station Inside | « Station Inside » | objectif (Oddball), peu d'acces | faible | gamerguides |
| Officer George's Desk | « behind the desk » | objectif : le porteur frappe les entrants d'Old Town Stairs | faible | gamerguides |
| Old Town | « Old Town » | objectif (point B des Strongholds) + grenades plasma | faible | gfinity |
| Arc Street | « Arc Street » | equipement : Thruster, mobilite sur toute la carte | faible | gfinity |
| Station Tower 2 | « Station Tower 2 » | arme (fusil tactique VK78) | faible | progameguides |
| Arc Street Bend | « Arc Street Bend » | arme (fusil tactique, spawn alternatif) | faible | progameguides |
| Commercial District | « Commercial District » | arme (Bulldog / Heatwave) | faible | progameguides |
| Cafe | « Station Inside and Cafe » | arme (Pulse Carbine) | faible | progameguides |
| Main Street Alley | « Main Street Alley » | objectif : spawn de l'Oddball, a rusher en debut de manche | faible | gamerguides |
| Plaza | « the central plaza » | flanc et rotation rapide par les escaliers lateraux | faible | hyperxarenalasvegas |

**Contre-exemples cites** — et ils portent sur la MEME zone que la position d'arme la plus
forte : « Avoid walking through the center of the map unless you're going for the Power
Weapon » et une rubrique « Areas to Avoid: Center of the map » (gamerguides) ; « the Old Town
Bar is a very enclosed space » avec des flanqueurs sortant d'Arc Street Hallway (gfinity).
C'est un desaccord apparent a garder en tete pour l'etape 2 : Main Street est a la fois le
point d'arme decisif et le corridor le plus mortel. Un algorithme empirique devrait y voir
beaucoup de kills ET beaucoup de morts.

### 2.5 Bazaar — `ctf_bazaar`

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| Market | « Market » | centre : « Controlling Market is crucial for winning on Bazaar » ; « central market area… crux of the fighting » | forte | dotesports, ggrecon |
| Tower | « Tower » | hauteur (surplombe Market) + arme forte : Cindershot et M41 SPNKr au debut de chaque manche | faible | dotesports |
| Antenna | « Antenna » | hauteur : directement au-dessus de Market | faible | dotesports |
| Cafe | « Cafe » | hauteur au-dessus de Market ; arme (fusil tactique) | faible | dotesports, gamerguides |
| Den | « Den » | hauteur au-dessus de Market ; arme (fusil tactique) | faible | dotesports, gamerguides |
| Palm Tree | « Palm Tree » | hauteur : directement au-dessus de Market | faible | dotesports |
| Truck | « near Truck » | power-ups Overshield / Active Camo, en face de Tower | faible | dotesports |
| East Market Bridge | « East to West Market Bridge » | lignes de vue longue portee en travers du marche | faible | gfinity |
| West Market Bridge | « East to West Market Bridge » | lignes de vue longue portee en travers du marche | faible | gfinity |
| Tower Basement | « Tower Basement » | itineraire de drapeau a couvert | faible | gamerguides |
| West Base | « West Base » | objectif + surveillance de West Alley et West Courtyard | faible | gamerguides |
| East Base | « East Base » | objectif + surveillance d'East Courtyard et East Alley | faible | gamerguides |

**Contre-exemples cites** : « Market is located in the center of Bazaar and is exposed from
all sides » et, pour Tower Basement, « enemies can stop the flag carrier in this secluded
area, making it tough for their teammates to recover the flag » (dotesports). Meme tension
que sur Streets : la zone centrale est a la fois le centre de gravite et le lieu le plus
expose.

### 2.6 Catalyst — `catalyst`

Carte la plus mal servie par l'ecrit : aucun guide accessible n'emploie ses callouts
officiels (`Mohawk`, `Spine`, `Plank`, `Overgrowth`, `Grand Hall`…).

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| Top Mid | « the central area » (rattachement lexical) | hauteur + « long sightlines and high degree of verticality » | faible | halopedia |
| Middle Ramp | « that long midway ramp » (rattachement lexical) | arme forte tres convoitee au centre de la rampe | faible | thegamer |
| ? | « a higher position » / « high ground » | hauteur — mais « keeping it is another story » | faible | gamerant |
| ? | « the closed-off areas on the side » | acces / couvert : tunnels lateraux, repli, multi-kills a la grenade | faible | gamerant |

**Contre-exemples cites** : « The center of the map which features a light bridge is the
easiest place to get killed off by enemy players », « there is no cover on the bridge or in
the center region of the map », « it's best to avoid this place outright » (gamerant).

### 2.7 Forbidden — `ctf_forbidden`

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| Dried Rat Hole | « rat holes » | objectif : evasion du porteur de drapeau ; repositionnement vertical vers le niveau bas | forte | kotaku, halowaypoint |
| Overgrown Rat Hole | « rat holes » | idem, cote miroir | forte | kotaku, halowaypoint |
| Center Bridge | « the center of the map » (rattachement lexical) | equipement de puissance au centre | forte | halopedia, halowaypoint |
| ? | « snipers on either side » | arme forte : deux snipers, un par cote | faible | halowaypoint |
| ? | « the middle of the map » | lignes de vue longues, duels sniper / fusil | faible | gamersdecide |

Citations d'appui : « These are chutes on either side of the map that you can slide into to
drop down to the lower level, making for great flag getaways » (kotaku) ; « It's a symmetrical
arena map with snipers on either side, long sightlines, and Power Equipment in the middle »
(halowaypoint) ; « The center of the map includes a power equipment spawn. » (halopedia).

### 2.8 Argyle — `dd600260-d91c-4d77-9990-3f35873c90a1`

Le depot ne connait que **5 zones** pour Argyle. La plupart des positions decrites par les
guides n'ont donc aucune zone officielle ou atterrir.

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| East Rampart | « high-up battlement » (rattachement lexical) | hauteur (acces par Gravity Lift) + equipement Grappleshot | faible | halopedia |
| West Rampart | « high-up battlement » (rattachement lexical) | idem, cote miroir | faible | halopedia |
| Platform | « floating platform in the middle » | arme : Active Camouflage sur la plateforme, Repulsor dessous ; element central de la carte | faible | halopedia |
| ? | « an easy-to-access tower » pres de chaque spawn | hauteur, difficile a atteindre pour l'adversaire ; sniper au coin | faible | thegamer (camping spots) |
| ? | « elevated walkway over a bottomless pit » | hauteur + arme (Sniper Rifle dans le coin) | faible | halopedia |
| ? | « flanking hallways » | lignes de vue ouvertes vers le centre tout en gardant du couvert | faible | halopedia |

**Contre-exemples cites** : « long, wide-open hallway that connects the two bases, giving both
sides a clear sightline between each other » et « The central area of the map is wide open and
set at a lower elevation than the rest of the map. » (halopedia).

### 2.9 Empyrean — `d035fc3e-f298-4c14-9487-465be2e1dc1f`

Remake de The Pit ; layout atteste « near-identical » par halopedia. `⁂` = transfert.

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| East Tower | « Sniper Tower » ⁂ (rattachement lexical) | hauteur + arme : « one of the strongest positions » ; vue directe sur la tour adverse | faible | halopedia (The Pit) |
| West Tower | « Sniper Tower » ⁂ (rattachement lexical) | idem, cote miroir | faible | halopedia (The Pit) |
| East Balcony | « a new balcony » (rattachement lexical) | power-up : l'Overshield a ete deplace sur un balcon surplombant le bord de carte | faible | halopedia (Empyrean) |
| West Balcony | « a new balcony » (rattachement lexical) | idem, cote miroir | faible | halopedia (Empyrean) |
| ? | « middle corridor » | arme forte : Rocket Launcher, la ou etait l'Active Camo | faible | halopedia (Empyrean) |
| ? | « Sword Room » / « Blue Room » / « Control Room » ⁂ | arme forte (epee sur The Pit, Heatwave sur Empyrean) | faible | halopedia (The Pit) |

**Contre-exemples cites** : « a player can be killed through the cracks of the metal plates on
the Sniper Tower » ; « Securing the rockets is very risky at the start of the game » ; la
Sword Room « sometimes goes uncontested due to the claustrophobic nature of the curved hallway
and limited cover » (halopedia The Pit) ; sur Empyrean, « the lower side of the map, where an
impassable fence was before, is now open » — donc chute mortelle (halopedia Empyrean).

### 2.10 Solitude — `f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42`

Remake de Plaza (Halo 5) : « very close to being a 1:1 remake in terms of spacing, but the
height and overall verticality of the map has been adjusted » (halowaypoint). `⁂` = transfert.

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| Top Nest | « sniper nest » ⁂ (rattachement lexical) | arme forte (Sniper) ; a tenir a 2-3 joueurs | faible | gamedeveloper |
| Bridge | « central walkway above the street » ⁂ (rattachement lexical) | power-up Overshield + hauteur au-dessus de la rue | faible | halopedia (Plaza) |
| ? | « the plaza » ⁂ | acces : deux goulots d'etranglement, zone defendable | faible | gamedeveloper |
| ? | « gravity lift » ⁂ | acces vertical rapide debouchant sur le nid de sniper | faible | gamedeveloper |

**Contre-exemples cites, dont un qui CONTREDIT la position ci-dessus** : le meme article du
level designer de Plaza dit du nid de sniper « there isn't a clear vantage point for players
to pick enemies off » — a mettre en regard de « 2-3 members guarding the area, with one of
them using the sniper rifle ». Par ailleurs l'Overshield du pont expose : « players risk dying
before the shield kicks in » ; et sur Solitude meme, la zone centrale ouverte : « As it's so
open in that middle area, the Stalker Rifle actually ended up annihilating anyone and everyone
who walked into that space. » (halowaypoint).

### 2.11 Interference — `654dff62-d618-496a-8914-06ab73d991e3`

Remake de The Rig (Halo 5). Le vocabulaire du depot coincide largement avec celui de The Rig
(`Tower`, `Bunker`, `Long Hall`, `Pit`, `Pipes`, `Engine`), ce qui rend le transfert plus sur
que sur les autres cartes — mais il reste un transfert. `⁂`.

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| Tower | « Tower » ⁂ | hauteur : « gives a nice overview of the entire inside area of the map » | faible | thegamehaus |
| Long Hall | « Long Hall » ⁂ | lignes de vue surveillees depuis Tower | faible | thegamehaus |
| Bunker | « Bunker spawn » ⁂ | spawn : ancrage d'equipe, « a team ideally should have control of the Bunker spawn » | faible | thegamehaus |
| ? | « Nest » ⁂ (Top Nest ou Bottom Nest, non departage) | arme forte : sniper, peu de couvert | faible | thegamehaus |
| ? | zones de Strongholds | lignes de vue multiples — le sniper y a ete RETIRE pour cette raison | faible | halowaypoint |
| ? | power-up de la carte | arme : « The power up on this map is the Overshield. » | faible | halowaypoint |

**Contre-exemples cites** : le Nest, « If the Red team is not careful about pushing this area,
the Blue team can easily toss in grenades. » ; et « The Blue team has the weaker initial
spawn. » (thegamehaus, sur The Rig).

### 2.12 Banished Narrows — `9ad226d8-8947-4c5b-95bc-d220187698c1`

Aucune source ne parle de la carte Halo Infinite. Tout vient de Narrows (Halo 3). `⁂`.

| Zone officielle EN | Nom du guide | Pourquoi | Confiance | Sources |
|---|---|---|---|---|
| Top Mid | « main bridge » / « upper bridge » ⁂ (rattachement lexical) | axe central : « generally the area where most of the fighting occurs » ; arme (rockets au centre du pont) | faible | halopedia (Narrows) |
| ? | « raised walkway » a gauche et a droite de l'entree du pont ⁂ | hauteur : vue sur tout le pont | faible | halopedia (Narrows) |
| ? | « back center » ⁂ | arme : poste de sniper, longue ligne de vue sur sa moitie de pont | faible | halopedia (Narrows) |

**Contre-exemples cites** : l'etage haut du pont est « vaster, allowing players to be hit from
multiple sides and angles » ; le man cannon trahit la rotation, « a very distinguishable sound
can be heard across the map » ; et en vol, « the lower one will most likely perish. »
(halopedia Narrows).

### 2.13 Cartes sans position exploitable

- **Fortress** (`0d1c9255-…`) — aucune zone nommee dans le depot ET aucun guide de positions.
  Seul fait cite : « two snipers up for grabs, and an opportunity to play offensively with an
  Overshield » (https://www.halowaypoint.com/news/great-journey-operation-launch), confirme
  par https://www.halowaypoint.com/news/playlist-overview-great-journey.
- **Origin** (`b302eb62-…`) — aucune zone nommee, aucune source de position. Reimagination de
  Coliseum (Halo 5), dont la page halopedia ne dit rien de tactique non plus.
- **Lattice** (`1a6cfc2e-…`) — aucune zone nommee, aucune source de position. Seul contexte :
  « players have long been asking for a new asymmetrical competitive map »
  (https://www.halowaypoint.com/news/ranked-hcs-update).

---

## 3. TABLE FINALE PARSABLE

Format strict, une ligne par position :
`| carte_cle | zone_en | nom_guide | confiance | raison | sources |`

Regles de lecture pour le test de l'etape 2 :

1. Une ligne dont `zone_en` vaut `?` **n'est pas exploitable** : ni au rappel, ni a la
   precision. Elle est conservee pour la tracabilite humaine.
2. Seules les lignes `confiance = forte` comptent au **rappel**.
3. Les lignes `forte` ET `faible` (hors `?`) comptent a la **precision**.
4. `raison` est un ou plusieurs mots-clefs parmi : `hauteur`, `lignes de vue`, `arme`,
   `objectif`, `spawn`, `acces`, separes par `+`.
5. `sources` = URLs separees par un espace.

| carte_cle | zone_en | nom_guide | confiance | raison | sources |
|---|---|---|---|---|---|
| ctf_aquarius | Top Mid | Top Mid | forte | hauteur + lignes de vue + arme | https://www.halopedia.org/Aquarius https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius https://www.gfinityesports.com/halo-infinite/aquarius-callouts/ https://www.thegamer.com/halo-infinite-best-multiplayer-maps/ |
| ctf_aquarius | Hydro | Hydro | forte | arme + lignes de vue | https://www.gfinityesports.com/halo-infinite/aquarius-callouts/ https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| ctf_aquarius | Planters | Planters area | faible | arme | https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| ctf_aquarius | Yellow Base | Yellow Base | faible | objectif + spawn | https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| ctf_aquarius | Blue Base | Blue Base | faible | objectif + spawn | https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| sgh_interlock | Tower | Tower | forte | hauteur + lignes de vue + acces + arme | https://www.gfinityesports.com/halo-infinite/live-fire-callouts/ https://www.thegamer.com/camping-spots-in-halo-infinite/ https://gamerant.com/halo-infinite-best-weapon-spawn-live-fire/ |
| sgh_interlock | Hallway | Hallway | forte | arme | https://www.gfinityesports.com/halo-infinite/live-fire-callouts/ https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire https://gamerant.com/halo-infinite-best-weapon-spawn-live-fire/ |
| sgh_interlock | Platform | cylindrical platform / Overshield platform | forte | arme | https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire https://gamerant.com/halo-infinite-best-weapon-spawn-live-fire/ |
| sgh_interlock | Overlook | Overlook | faible | lignes de vue + arme | https://www.gfinityesports.com/halo-infinite/live-fire-callouts/ |
| sgh_interlock | House Window | House Window | faible | objectif + acces | https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire |
| sgh_interlock | House Elbow | House Elbow | faible | acces | https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire |
| sgh_interlock | Tunnel | Tunnel | faible | acces | https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire |
| sgh_interlock | House | House | faible | acces | https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire |
| sgh_blueprint | Attic | Attic | forte | hauteur + lignes de vue | https://www.gfinityesports.com/article/recharge-callouts https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Maintenance Bay | Maintenance Bay | forte | hauteur + lignes de vue + acces | https://www.gfinityesports.com/article/recharge-callouts https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Pit | Pit | forte | arme | https://www.gamerguides.com/halo-infinite/guide/maps/arena/recharge https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Hydro | Hydro | forte | arme + objectif + spawn | https://www.gamerguides.com/halo-infinite/guide/maps/arena/recharge https://www.gfinityesports.com/article/recharge-callouts https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Whirlpool Dam | Whirlpool | forte | hauteur + lignes de vue + arme | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Orange Pipes | Orange Pipes | forte | acces + arme | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Control Room | Control Room's second floor | faible | hauteur + lignes de vue | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Elevator | top of Elevator | faible | hauteur + objectif + acces | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Batteries | Batteries | faible | objectif | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Long Hall | Long Hall | faible | arme | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Overhang | Overhang | faible | arme | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Blue Pipes | Blue Pipes | faible | arme + acces | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Platform | Hydro platform | faible | hauteur + lignes de vue | https://hyperxarenalasvegas.com/halo-infinite-season-six-best-loadouts-map-strategies/ |
| sgh_streets | Main Street | middle of Main Street | forte | arme | https://www.gfinityesports.com/article/streets-callouts https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ |
| sgh_streets | Subway Balcony | a balcony / Subway Balcony | forte | lignes de vue + arme | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ |
| sgh_streets | Subway | the Subway | faible | acces | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets |
| sgh_streets | Station Inside | Station Inside | faible | objectif + acces | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets |
| sgh_streets | Officer George's Desk | behind the desk | faible | objectif + acces | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets |
| sgh_streets | Old Town | Old Town | faible | objectif | https://www.gfinityesports.com/article/streets-callouts |
| sgh_streets | Arc Street | Arc Street | faible | arme | https://www.gfinityesports.com/article/streets-callouts |
| sgh_streets | Station Tower 2 | Station Tower 2 | faible | arme | https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ |
| sgh_streets | Arc Street Bend | Arc Street Bend | faible | arme | https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ |
| sgh_streets | Commercial District | Commercial District | faible | arme | https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ |
| sgh_streets | Cafe | Station Inside and Cafe | faible | arme | https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ |
| sgh_streets | Main Street Alley | Main Street Alley | faible | objectif | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets |
| sgh_streets | Plaza | the central plaza | faible | acces | https://hyperxarenalasvegas.com/halo-infinite-season-six-best-loadouts-map-strategies/ |
| ctf_bazaar | Market | Market | forte | objectif + arme | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide https://www.ggrecon.com/guides/halo-infinite-maps-list/ |
| ctf_bazaar | Tower | Tower | faible | hauteur + arme | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide |
| ctf_bazaar | Antenna | Antenna | faible | hauteur | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide |
| ctf_bazaar | Cafe | Cafe | faible | hauteur + arme | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide https://www.gamerguides.com/halo-infinite/guide/maps/arena/bazaar |
| ctf_bazaar | Den | Den | faible | hauteur + arme | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide https://www.gamerguides.com/halo-infinite/guide/maps/arena/bazaar |
| ctf_bazaar | Palm Tree | Palm Tree | faible | hauteur | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide |
| ctf_bazaar | Truck | near Truck | faible | arme | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide |
| ctf_bazaar | East Market Bridge | East to West Market Bridge | faible | lignes de vue | https://www.gfinityesports.com/article/bazaar-callouts |
| ctf_bazaar | West Market Bridge | East to West Market Bridge | faible | lignes de vue | https://www.gfinityesports.com/article/bazaar-callouts |
| ctf_bazaar | Tower Basement | Tower Basement | faible | objectif + acces | https://www.gamerguides.com/halo-infinite/guide/maps/arena/bazaar |
| ctf_bazaar | West Base | West Base | faible | objectif + spawn | https://www.gamerguides.com/halo-infinite/guide/maps/arena/bazaar |
| ctf_bazaar | East Base | East Base | faible | objectif + spawn | https://www.gamerguides.com/halo-infinite/guide/maps/arena/bazaar |
| catalyst | Top Mid | the central area | faible | hauteur + lignes de vue | https://www.halopedia.org/Catalyst |
| catalyst | Middle Ramp | that long midway ramp | faible | arme | https://www.thegamer.com/halo-infinite-best-multiplayer-maps/ |
| catalyst | ? | a higher position / high ground | faible | hauteur | https://gamerant.com/halo-infinite-lone-wolves-catalyst-map-tips/ |
| catalyst | ? | the closed-off areas on the side | faible | acces | https://gamerant.com/halo-infinite-lone-wolves-catalyst-map-tips/ |
| ctf_forbidden | Dried Rat Hole | rat holes | forte | objectif + acces | https://kotaku.com/halo-infinite-season-5-maps-bandit-evo-forge-ai-1850908494 https://www.halowaypoint.com/news/maps-overview-season-5 |
| ctf_forbidden | Overgrown Rat Hole | rat holes | forte | objectif + acces | https://kotaku.com/halo-infinite-season-5-maps-bandit-evo-forge-ai-1850908494 https://www.halowaypoint.com/news/maps-overview-season-5 |
| ctf_forbidden | Center Bridge | the center of the map | faible | arme | https://www.halopedia.org/Forbidden https://www.halowaypoint.com/news/maps-overview-season-5 |
| ctf_forbidden | ? | snipers on either side | faible | arme | https://www.halowaypoint.com/news/maps-overview-season-5 |
| ctf_forbidden | ? | the middle of the map | faible | lignes de vue | https://www.gamersdecide.com/articles/halo-infinite-best-maps |
| dd600260-d91c-4d77-9990-3f35873c90a1 | East Rampart | high-up battlement | faible | hauteur + arme | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| dd600260-d91c-4d77-9990-3f35873c90a1 | West Rampart | high-up battlement | faible | hauteur + arme | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| dd600260-d91c-4d77-9990-3f35873c90a1 | Platform | floating platform in the middle | faible | arme | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| dd600260-d91c-4d77-9990-3f35873c90a1 | ? | an easy-to-access tower near each spawn | faible | hauteur + arme + acces | https://www.thegamer.com/camping-spots-in-halo-infinite/ |
| dd600260-d91c-4d77-9990-3f35873c90a1 | ? | elevated walkway over a bottomless pit | faible | hauteur + arme | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| dd600260-d91c-4d77-9990-3f35873c90a1 | ? | flanking hallways | faible | lignes de vue + acces | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | East Tower | Sniper Tower | faible | hauteur + arme + lignes de vue | https://www.halopedia.org/index.php?title=The_Pit&action=raw https://www.halopedia.org/index.php?title=Empyrean&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | West Tower | Sniper Tower | faible | hauteur + arme + lignes de vue | https://www.halopedia.org/index.php?title=The_Pit&action=raw https://www.halopedia.org/index.php?title=Empyrean&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | East Balcony | a new balcony | faible | arme | https://www.halopedia.org/index.php?title=Empyrean&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | West Balcony | a new balcony | faible | arme | https://www.halopedia.org/index.php?title=Empyrean&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | ? | middle corridor | faible | arme | https://www.halopedia.org/Empyrean |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | ? | Sword Room / Blue Room / Control Room | faible | arme | https://www.halopedia.org/index.php?title=The_Pit&action=raw |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Top Nest | sniper nest | faible | arme + hauteur | https://www.gamedeveloper.com/design/level-design-in-halo-5-multiplayer-plaza |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Bridge | central walkway above the street | faible | hauteur + arme | https://www.halopedia.org/Plaza |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | ? | the plaza | faible | acces | https://www.gamedeveloper.com/design/level-design-in-halo-5-multiplayer-plaza |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | ? | gravity lift | faible | acces | https://www.gamedeveloper.com/design/level-design-in-halo-5-multiplayer-plaza |
| 654dff62-d618-496a-8914-06ab73d991e3 | Tower | Tower | faible | hauteur + lignes de vue | https://thegamehaus.com/esports/the-playbook-rig-slayer/2017/03/18/ |
| 654dff62-d618-496a-8914-06ab73d991e3 | Long Hall | Long Hall | faible | lignes de vue | https://thegamehaus.com/esports/the-playbook-rig-slayer/2017/03/18/ |
| 654dff62-d618-496a-8914-06ab73d991e3 | Bunker | Bunker spawn | faible | spawn + acces | https://thegamehaus.com/esports/the-playbook-rig-slayer/2017/03/18/ |
| 654dff62-d618-496a-8914-06ab73d991e3 | ? | Nest | faible | arme | https://thegamehaus.com/esports/the-playbook-rig-slayer/2017/03/18/ |
| 654dff62-d618-496a-8914-06ab73d991e3 | ? | Strongholds locations | faible | lignes de vue | https://www.halowaypoint.com/news/interference-update |
| 9ad226d8-8947-4c5b-95bc-d220187698c1 | Top Mid | main bridge / upper bridge | faible | arme + lignes de vue | https://www.halopedia.org/Narrows |
| 9ad226d8-8947-4c5b-95bc-d220187698c1 | ? | raised walkway | faible | hauteur + lignes de vue | https://www.halopedia.org/Narrows |
| 9ad226d8-8947-4c5b-95bc-d220187698c1 | ? | back center | faible | arme + lignes de vue | https://www.halopedia.org/Narrows |

**Totaux (comptes sur le fichier, pas a la main)** : **84 lignes**, toutes a 6 colonnes,
toutes avec au moins une URL ; **12 cartes** ; **17 positions `forte`** reparties sur
**6 cartes** — Recharge 6, Live Fire 3, Forbidden 3, Aquarius 2, Streets 2, Bazaar 1
(Catalyst 0, et 0 sur les quatre cartes de remake) ; **67 `faible`** ; **15 lignes**
avec `zone_en = ?` (non exploitables par le test).

### 3.1 Table des contre-exemples (positions decrites comme pieges)

Meme format, colonne `raison` = pourquoi le guide deconseille.

| carte_cle | zone_en | nom_guide | confiance | raison | sources |
|---|---|---|---|---|---|
| ctf_aquarius | Yellow Base | near the flag | faible | statique pres du drapeau : tire depuis la base adverse | https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| ctf_aquarius | Blue Base | near the flag | faible | statique pres du drapeau : tire depuis la base adverse | https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| sgh_interlock | Canal | the Canal | faible | chute : « Don't fall in the Canal! » | https://www.gfinityesports.com/halo-infinite/live-fire-callouts/ |
| sgh_blueprint | Batteries | Batteries | faible | cible du tir plongeant d'Attic / Maintenance Bay | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Bridge | Bridge | faible | cible du tir plongeant d'Attic / Maintenance Bay | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Pit | Pit | faible | cible du tir plongeant d'Attic / Maintenance Bay | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Storage | Storage | faible | cible du tir plongeant d'Attic / Maintenance Bay | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Overhang | Overhang | faible | cible du tir plongeant d'Attic / Maintenance Bay | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Long Hall | long hallways | faible | camping d'arme forte, positionnement a risque | https://hyperxarenalasvegas.com/halo-infinite-season-six-best-loadouts-map-strategies/ |
| sgh_streets | Main Street | center of the map | faible | expose sur plusieurs axes : « Areas to Avoid » | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets |
| sgh_streets | Old Town Bar | Old Town Bar | faible | espace tres clos, piege a grenades ; flanqueurs d'Arc Street Hallway | https://www.gfinityesports.com/article/streets-callouts |
| ctf_bazaar | Market | Market | faible | centre expose de toutes parts | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide |
| ctf_bazaar | Tower Basement | Tower Basement | faible | isole : le porteur y est arrete sans secours possible | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide |
| catalyst | Bottom Mid | the center of the map / the light bridge | faible | aucun couvert, snipe depuis la hauteur ; « avoid this place outright » | https://gamerant.com/halo-infinite-lone-wolves-catalyst-map-tips/ |
| dd600260-d91c-4d77-9990-3f35873c90a1 | ? | long, wide-open hallway between the bases | faible | ligne de vue mutuelle base a base | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| dd600260-d91c-4d77-9990-3f35873c90a1 | Platform | the central area | faible | ouvert et plus bas que le reste de la carte | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | East Tower | Sniper Tower | faible | tuable a travers les interstices des plaques metalliques | https://www.halopedia.org/The_Pit |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | West Tower | Sniper Tower | faible | tuable a travers les interstices des plaques metalliques | https://www.halopedia.org/The_Pit |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | ? | rockets (couloir central) | faible | prise en debut de manche tres risquee | https://www.halopedia.org/index.php?title=The_Pit&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | ? | Sword Room | faible | couloir courbe exigu, couvert limite, souvent non conteste | https://www.halopedia.org/index.php?title=The_Pit&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | ? | the lower side of the map | faible | bord desormais ouvert : chute mortelle | https://www.halopedia.org/index.php?title=Empyrean&action=raw |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Top Nest | sniper nest | faible | CONTREDIT la ligne forte : « there isn't a clear vantage point » | https://www.gamedeveloper.com/design/level-design-in-halo-5-multiplayer-plaza |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Bridge | Overshield | faible | activation differee : on meurt avant que le bouclier monte | https://www.gamedeveloper.com/design/level-design-in-halo-5-multiplayer-plaza |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Bottom Mid | that middle area | faible | tres ouvert : tenu a distance par un Stalker Rifle | https://www.halowaypoint.com/news/solitude-preview-season-4 |
| 654dff62-d618-496a-8914-06ab73d991e3 | ? | Nest | faible | peu de couvert, vulnerable aux grenades | https://thegamehaus.com/esports/the-playbook-rig-slayer/2017/03/18/ |
| 9ad226d8-8947-4c5b-95bc-d220187698c1 | Top Mid | upper bridge | faible | vaste : on y est touche de plusieurs cotes a la fois | https://www.halopedia.org/Narrows |

---

## 4. Desaccords entre sources

1. **Streets — Main Street** : c'est a la fois la position d'arme decisive
   (gfinity, progameguides) et la zone explicitement listee « Areas to Avoid » par
   gamerguides. Les deux sont vraies : on y va pour l'arme, on n'y reste pas. Pour l'etape 2,
   attendre beaucoup de kills ET beaucoup de morts sur cette zone.
2. **Bazaar — Market** : « Controlling Market is crucial » et « exposed from all sides », dans
   le MEME article. Meme regime que Streets.
3. **Catalyst — centre** : halopedia vante « long sightlines and high degree of verticality » ;
   gamerant dit « it's best to avoid this place outright ». Le desaccord porte sur le HAUT du
   centre (position) contre le BAS du centre / pont lumineux (piege) ; le vocabulaire du depot
   distingue justement `Top Mid` et `Bottom Mid`. A verifier a l'etape 2 : l'algorithme
   devrait retenir `Top Mid` et rejeter `Bottom Mid`.
4. **Solitude — nid de sniper** : le level designer de Plaza ecrit a la fois qu'il faut
   « 2-3 members guarding the area, with one of them using the sniper rifle » ET qu'« there
   isn't a clear vantage point for players to pick enemies off ». C'est la meme page : la
   position vaut pour l'arme, pas pour la vue. Confiance laissee a `faible`.
5. **Aquarius — les bases** : position d'objectif pour gamerguides, mais la meme page
   deconseille d'y stationner. Meme lecture que Streets.

---

## 5. Limites

### 5.1 Cartes sans source de positions

- **Origin**, **Lattice**, **Fortress** : aucune position trouvee, et de surcroit **aucune
  zone nommee dans le depot**. Ces trois cartes ne peuvent ni entrer a l'oracle, ni servir a
  l'etape 2. Si l'etape 1 trouve du corpus dessus, l'algorithme produira des polygones
  **invalidables** — les traiter comme cartes hors verdict.
- **Illusion** (`ctf_illusion`) : sources trouvees (halowaypoint CU29, halopedia) mais le
  vocabulaire est inutilisable. Les sources parlent de « camo hallway » central, de « side
  areas », de « kinetic launchers » ; le depot ne connait que des zones prefixees `Camo …` /
  `Overshield …` qui designent les DEUX MOITIES de la carte, pas le couloir central. Aucun
  rattachement honnete possible. Carte ecartee.
- **Prism** (`sgh_crystalcaves`), **Cliffhanger** (`ridgeline`) : hors pool HCS courant, non
  recherchees faute de budget ; le depot a leur vocabulaire si besoin plus tard.

### 5.2 Cartes dont le vocabulaire du depot ne couvre pas les noms des guides

- **Argyle** — le depot ne connait que 5 zones (`East/West Stairs`, `East/West Rampart`,
  `Platform`). « elevated walkway », « flanking hallways », « vent passageway », « bases »,
  « lower level » n'ont aucun equivalent. Oracle structurellement pauvre sur cette carte.
- **Empyrean** — « Sword Room » / « Blue Room » / « Control Room », « Camo Hall »,
  « Green Home », « High Bridge », « Overshield Alley », « middle corridor » : candidats
  possibles parmi `Deck`, `Terminal`, `Alley`, `Elevator`, `Pit`, aucun tranchable sur pieces.
  Non devines.
- **Solitude** — aucun nom du glossaire du depot (`Cafe`, `Hotel`, `Market`, `Posters`,
  `Junction`, `Sneaky`…) n'apparait dans une source tactique. A noter aussi une divergence de
  perimetre : halopedia enumere 30 zones, le depot 36.
- **Catalyst** — aucun guide ecrit accessible n'emploie ses callouts officiels. `Plank`,
  `Mohawk`, `Spine`, `Overgrowth`, `Overlook`, `Catwalk`, `Grand Hall`, `Window`,
  `Closed/Open Hall`, `Balcony` ne sont situes nulle part.
- **Forbidden** — seul `Rat Hole` est atteste litteralement. `Mid`, `Center Bridge`,
  `Nest`, `Attic`, `Mohawk`, `Diving Board` ne sont employes par aucune source ecrite.
- **Banished Narrows** — tout vient de Narrows (Halo 3), dont le vocabulaire est different.
  Seul `Top Mid` ↔ « upper / main bridge » est un rattachement lexical defendable.
- **Streets** — « Subway Entrance » n'existe pas au vocabulaire (candidats `Subway Stairs` ou
  `Subway Bend`, non tranches). Les points A et C des Strongholds ne sont jamais nommes.
- **Recharge** — « Barriers » et « Hydro Room » (gamerguides) sont absents du vocabulaire ;
  « the pipe room » est ambigu entre `Blue Pipes` et `Orange Pipes` ; « Hydro platform » →
  `Platform` est un rattachement lexical non atteste. `Sneaky`, `Whirlpool Ledge` et
  `Maintenance Elbow Stairs` ne sont employes par aucune source.
- **Bazaar** — `Crawl Space`, `Stalls`, `East/West Gate`, `Tower Stairs`, `Library` ne sont
  employes par aucune source tactique.

**Rattachements lexicaux faits (a auditer par le pilote)** : Live Fire `Platform`
(« cylindrical platform »), Recharge `Platform` (« Hydro platform »), Catalyst `Top Mid`
(« the central area ») et `Middle Ramp` (« long midway ramp »), Forbidden `Center Bridge`
(« the center of the map »), Argyle `East/West Rampart` (« high-up battlement »), Empyrean
`East/West Tower` (« Sniper Tower ») et `East/West Balcony` (« a new balcony »), Solitude
`Top Nest` (« sniper nest ») et `Bridge` (« central walkway above the street »), Banished
Narrows `Top Mid` (« main bridge »). Chacun est signale dans les tables de la section 2.

### 5.3 Angle mort de l'outillage

Le savoir competitif reel de Halo Infinite est sur **Reddit (r/CompetitiveHalo), YouTube
(guides de callouts et analyses de setup), Twitter/X et Discord**. Aucun n'est atteignable
par cet outillage : reddit.com est refuse par le user-agent ; youtube.com ne sert ni
description ni transcription ; x.com et halo.fandom.com renvoient 402 ; liquipedia.net renvoie
403 sur les pages HTML (l'API `action=parse` passe, elle). Resultat : **l'oracle repose sur
des guides grand public 2021-2023, pas sur des analyses HCS.** Les cartes de lancement
(Aquarius, Live Fire, Recharge, Streets, Bazaar, Catalyst) sont correctement couvertes ; tout
ce qui est sorti apres 2023 ne l'est pas.

Si le pilote veut un oracle de meilleure qualite sur les cartes recentes, la voie est le
serveur MCP `browser` (navigateur reel) applique a r/CompetitiveHalo et aux descriptions
YouTube — c'est une tache a part, pas un ajustement de celle-ci.

### 5.4 Ce que cet oracle vaut, et ne vaut pas

- Il est **solide** sur Recharge (6 positions fortes), Live Fire (3), Forbidden (3),
  Aquarius (2), Streets (2).
- Il est **indicatif** sur Bazaar (1 forte) et Catalyst (0 forte).
- Il est **hypothetique** sur Empyrean, Solitude, Interference, Banished Narrows : rien n'y
  est atteste sur la carte Halo Infinite elle-meme, tout vient de la carte d'origine du
  remake. **Ne pas s'en servir pour prononcer un GO/NO-GO.**
- Le critere de succes du plan (« au moins 4 cartes du circuit HCS avec rappel ≥ 0,7 et
  precision ≥ 0,6 ») est atteignable : 5 cartes portent au moins 2 positions `forte`. C'est
  le socle a viser a l'etape 2 — Recharge, Live Fire, Forbidden, Aquarius, Streets — sous
  reserve que le corpus mesure a l'etape 1 les couvre.

---

## 6. Verification du gate 0 (faite par l'executant)

| Critere du gate | Verdict |
|---|---|
| Le document existe | oui — `.ai/V7.5/positions_de_force/ORACLE_PRO_2026-09-20.md` |
| ≥ 4 cartes | oui — **12 cartes** portent au moins une position |
| Chaque position a ≥ 1 source citee par URL | oui — verifie par script : 0 ligne sans URL sur les 84 lignes de la table finale et les 26 lignes de contre-exemples |
| Le tableau final est parsable, une ligne par position | oui — 6 colonnes fixes, une position par ligne, regles de lecture ecrites en tete de section 3 |

Reste a la charge du pilote : relire et valider (notamment les rattachements lexicaux de la
section 5.2 et le statut `faible` des quatre cartes de remake).

## 7. Relecture du pilote (2026-09-20) — gate 0 VALIDÉ, avec réserves

- Rattachements lexicaux audités : acceptés, sauf **Forbidden `Center Bridge` ← « the center of
  the map »** — ambigu entre `Mid` et `Center Bridge`, rétrogradé à `faible` dans la table
  finale (ligne modifiée par le pilote). Les autres (`Platform` ×2, `Top Mid` ×2, `Rampart`,
  `Tower`, `Balcony`, `Top Nest`, `Bridge`, `Middle Ramp`) désignent une zone unique et sont
  tenus.
- **L'oracle est mince** : 16 positions fortes sur 6 cartes, 2 positions seulement sur
  Aquarius, Streets et Bazaar. Le rappel par carte sera donc à gros grain (0 / 0,5 / 1 sur ces
  cartes). Conséquence pour l'étape 2, décidée maintenant : (a) le seuil de rappel 0,7 du plan
  reste tel quel ; (b) les cartes à moins de 3 positions fortes comptent au verdict mais un
  0,5 y est lu comme « indéterminé », pas comme un échec ; (c) le verdict exige en plus que le
  calcul ne marque AUCUN contre-exemple (§3.1) comme position de force sur les 6 cartes — un
  algorithme qui colore « à éviter » a tort même s'il retrouve les positions fortes.
- **Raison « arme » seule** (Live Fire `Hallway`, Recharge `Pit`, Streets `Main Street`,
  Forbidden `Center Bridge`) : ce sont des emplacements d'arme de puissance, pas forcément des
  positions TENABLES. Elles restent dans l'oracle (les sources les décrivent ainsi) mais le
  verdict les rapporte à part : les manquer n'est pas la même faute que manquer une hauteur.
- Fortress, Origin, Lattice : sans zones nommées dans le dépôt ni source → hors verdict, comme
  écrit en §5.1. Si le corpus les couvre, le catalogue pourra les porter SANS nom, mais elles ne
  prouvent rien.
- Source écartée `playhaloinfinite.com` (contenu synthétique, statistiques inventées) :
  confirmé par le pilote, qui l'avait aussi rencontrée avant l'ouverture du chantier.
