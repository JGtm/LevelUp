# Creneaux a verifier dans le Theater — propulseur et repulseur

Date : 2026-09-03. Destinataire : l'utilisateur (JGtm). Ce document n'est pas un rapport
technique : c'est une **liste d'endroits ou poser le curseur** dans le visionneur du jeu, et
un formulaire de releve.

Tous les matchs listes sont des matchs **ou JGtm a joue** — le Theater ne montre rien d'autre.
Sauf mention contraire, **c'est JGtm lui-meme qui agit** dans la partie 1.

---

## Comment lire le temps — et pourquoi on peut s'y fier

**La convention : le temps donne est le temps ECOULE DEPUIS LE DEBUT DU FILM**, curseur a
zero au tout debut de l'enregistrement (donc avant le decompte d'avant-match).

Elle a ete verifiee trois fois, et chaque verification est independante des deux autres :

| Controle | Ce qui a ete confronte | Resultat |
|---|---|---|
| **Le manifeste du jeu** | Chaque morceau du film porte, dans le manifeste publie par le jeu, l'instant ou il commence. On a recalcule ces instants a partir des horodatages internes. | Ecart maximal **4 a 17 ms** sur 23 films (21 a 38 morceaux chacun). |
| **Le coup d'envoi** | Le moment ou tout le monde s'elance en meme temps (la grille se leve), detecte sur les trajectoires. | Tombe entre **0:26 et 0:40** sur la quasi-totalite des films — la duree normale d'un decompte d'avant-match filme. |
| **Deux canaux du film, l'un contre l'autre** | Le film dit par un canal « ce joueur vient de consommer sa derniere charge de propulseur », et par un autre canal « ce joueur vient d'emettre une impulsion ». | **9 fois sur 9**, les deux tombent a **1 seconde ou moins** l'un de l'autre. |

**Ce dernier point est le plus parlant** : deux mecanismes qui n'ont rien en commun dans le
film datent le meme geste au meme instant sur l'horloge qu'on vous donne. Si la conversion
etait fausse, ils seraient decales ensemble.

**Le seul doute qui reste, et il est dit** : on n'a pas pu ouvrir le Theater pour lire l'heure
affichee a l'ecran. Si le visionneur affichait autre chose que le temps depuis le debut de
l'enregistrement (par exemple le chrono du match, qui demarre au coup d'envoi), **tout le
tableau serait decale d'un bloc de 26 a 40 secondes**, la valeur indiquee dans la colonne
« coup d'envoi ». C'est un decalage constant par film, pas un desordre : le premier creneau
que vous verifierez le tranchera pour tous les autres.

**Une seconde de tolerance suffit.** Les instants sont donnes a la seconde ; le geste lui-meme
dure environ une demi-seconde. Reculez de 2 secondes avant l'instant indique et regardez.

---

## Partie 1 — PROPULSEUR : 11 creneaux MESURES

Ce que le film enregistre a ces instants est etabli (rapport R8) : une **impulsion** emise par
le composant de capacite spartiate, sur un joueur qui porte le propulseur. La mesure physique
qui l'accompagne (« pic ») est la vitesse maximale atteinte au sol dans la demi-seconde autour
de l'instant. **Repere de comparaison : un instant pris au hasard dans le meme film donne
3,1 a 3,5 m/s ; le grappin donne 4,1 a 4,9.** Tout ce qui suit est a 8 m/s et plus.

**Ce que vous devez voir** : JGtm fait une embardee breve et rectiligne — une poussee laterale
ou vers l'avant qui casse net sa course, sans saut, sans grappin.

| # | Match (identifiant court + complet) | Carte probable | Coup d'envoi | Instant | Pic mesure | A voir / remarque |
|---|---|---|---|---|---|---|
| 1 | `1cd3848a` · `1cd3848a-e334-4795-835a-433a21d4009f` | Behemoth / Fragmentation / Launch Site | 0:26 | **2:15** | 12,4 m/s | **Le creneau le plus sur du lot.** Le film dit aussi, par un canal totalement independant, que JGtm consomme sa derniere charge de propulseur a 2:16. Les deux se suivent d'une seconde. |
| 2 | `3ba5a548` · `3ba5a548-336b-406f-ab58-7f8071337af4` | Streets | 0:26 | **3:09** | 8,3 m/s | **Meme double temoin** : derniere charge consommee a 3:10. |
| 3 | `1cd3848a` · (idem #1) | idem | 0:26 | **2:05** | **13,7 m/s** | Le pic le plus fort de tout le corpus JGtm. Fait partie d'une salve : 1:52, 1:55, 2:03, 2:05, 2:15 — les cinq valent le coup d'oeil d'affilee. |
| 4 | `46c3f91d` · `46c3f91d-8bef-433f-a517-3ea1c72aa76c` | Catalyst / Deadlock | 0:27 | **6:10** | 11,1 m/s | Salve de quatre : 6:10, 6:13, 6:17, 6:26. |
| 5 | `bfcd1175` · `bfcd1175-936c-4165-abd6-a48b58753887` | Recharge | 0:28 | **8:13** | 10,9 m/s | Le film le plus dense : 16 impulsions de JGtm entre 6:38 et 9:05. |
| 6 | `bf2a9f05` · `bf2a9f05-2818-4209-abb8-9e57d74ee78e` | Bazaar | 0:37 | **6:27** | 9,9 m/s | Precede de 6:23 ; suivi de 6:46 et 7:00. |
| 7 | `fccc61cd` · `fccc61cd-285d-4ccd-be82-908c348cbf69` | Behemoth / Fragmentation / Launch Site | 0:27 | **7:42** | 9,6 m/s | Suivi de 7:52, 7:54, 8:00, 8:04. |
| 8 | `8a485699` · `8a485699-0deb-40d2-aa3b-fa40e84b1e1d` | Behemoth / Fragmentation / Launch Site | 0:28 | **7:08** | 9,6 m/s | Salve 7:01 / 7:04 / 7:08 / 7:17. |
| 9 | `d1dfbc02` · `d1dfbc02-6f97-4bb2-822a-5dec1dadff2b` | Prism / Scarr | 0:35 | **2:09** | 8,9 m/s | Precede de 1:53. |
| 10 | `9e8fb31b` · `9e8fb31b-ea96-4848-a3b0-03117171d01e` | Cliffhanger | 0:26 | **2:03** | 8,8 m/s | **Le contre-test** : ce film ne rend que DEUX impulsions de JGtm sur toute la partie (2:03 et 6:35). Si vous en voyez beaucoup plus a l'ecran, cela chiffre ce que la lecture RATE. |
| 11 | `b1f01a33` · `b1f01a33-7e25-4dd8-b3ec-39e293d2cee6` | Prism / Scarr | 0:27 | **7:55** | 8,7 m/s | Suivi de 7:58. |

> Les noms de carte sont **indicatifs** : ils viennent de la geometrie du film, et plusieurs
> cartes partagent le meme gabarit. La carte exacte et la date sont dans LevelUp (voir la
> partie 3).

### Ce que la mesure a rendu, film par film

Dix films decodes, tous joueurs confondus :

| Film | Impulsions totales | dont JGtm |
|---|---|---|
| `8a485699` | 106 | 23 |
| `d1dfbc02` | 72 | 17 |
| `bfcd1175` | 70 | 16 |
| `bf2a9f05` | 68 | 23 |
| `b1f01a33` | 63 | 13 |
| `1cd3848a` | 62 | 14 |
| `fccc61cd` | 59 | 16 |
| `46c3f91d` | 54 | 8 |
| `3ba5a548` | 47 | 14 |
| `9e8fb31b` | 22 | 2 |

**146 impulsions de JGtm au total. 89 d'entre elles portent, en plus, la lecture explicite
« JGtm portait le propulseur juste avant » ; les 11 creneaux ci-dessus sont toutes de
celles-la.** Les 57 autres ne sont pas fausses pour autant : le canal qui dit ce qu'un joueur
porte ne s'exprime qu'environ une fois toutes les 20 secondes, donc pres de la moitie des
instants tombent entre deux de ses annonces.

**Aucun film n'est reste muet.** Le moins bavard, `9e8fb31b`, rend quand meme 22 impulsions
dont 2 de JGtm — d'ou son role de contre-test au tableau ci-dessus.

---

## Partie 2 — REPULSEUR : 14 creneaux CANDIDATS, non mesures

> **A LIRE AVANT LE TABLEAU.** Contrairement au propulseur, **le film n'a livre aucun signal
> d'usage du repulseur.** Tout ce qui suit est donc de la meme nature : des endroits ou
> l'effet ATTENDU d'un repulseur semble s'etre produit. **Ce ne sont pas des detections. Elles
> ne prouvent rien.** Elles peuvent aussi bien s'expliquer par une grenade, une explosion, un
> saut, ou une position mal reconstruite. Le but est exactement l'inverse d'une preuve : que
> vous alliez voir, et que vous disiez ce qui s'est reellement passe.

### 2.a — Un joueur proche est projete alors qu'un porteur de repulseur le visait

Regle appliquee (ecrite avant de mesurer) : un porteur de repulseur, un autre joueur a moins
de 6 metres, dont la vitesse passe brutalement de moins de 3 m/s a plus de 6,5 m/s **en
s'ecartant du porteur**, sans grappin en cours. La colonne « vise » dit si le porteur
regardait la victime au moment ou elle part (1,00 = pile dessus, 0 = de cote).

**CANDIDATS NON MESURES — a confirmer visuellement.**

| # | Match | Instant | Porteur du repulseur | Joueur projete | Vitesse avant → apres | Distance | Vise | Interet |
|---|---|---|---|---|---|---|---|---|
| C1 | `af3500aa` · `af3500aa-d13b-4ae6-858e-2416576e5212` | **1:34** | Madina97294 | **JGtm** | 2,6 → 11,0 m/s | 2,4 m | 0,99 | **Le meilleur candidat** : c'est vous qui partez, a bout portant, et le porteur vous regarde. Coup d'envoi du film a 1:18. |
| C2 | `a03a5e65` · `a03a5e65-ede1-4a35-95ef-1bb6e07c6faa` | **2:38** | Chocoboflor | **JGtm** | 2,6 → 6,8 m/s | 0,3 m | 0,97 | Corps a corps. Coup d'envoi 0:28. |
| C3 | `af3500aa` · (idem C1) | **3:48** | Seb4Slay3r | **JGtm** | 2,4 → 6,5 m/s | 2,0 m | 1,00 | Vous partez exactement dans l'axe du porteur. |
| C4 | `4577fcc4` · `4577fcc4-174c-47fc-a3ab-91140dd03486` | **3:25** | Madina97294 | Chocoboflor | 2,3 → 9,9 m/s | 4,5 m | 0,89 | La plus forte projection hors JGtm. Coup d'envoi 0:29. |
| C5 | `d9781168` · `d9781168-5fd6-4b00-a862-56e3a0a1f956` | **7:10** | (non nomme) | OFB4203689 | 2,6 → 8,5 m/s | 5,8 m | 1,00 | A la limite de portee. Coup d'envoi 0:40. |
| C6 | `9f57c612` · `9f57c612-0a93-4c6e-8955-c1caf0a23833` | **4:47** | scuderiasven | DueUnicycle7430 | 1,7 → 7,9 m/s | 1,9 m | 0,97 | Coup d'envoi 0:38. |
| C7 | `3d58eb37` · `3d58eb37-8efe-4042-8315-5b6a0e3a7525` | **5:36** | Davis7911 | Sizzlechest0 | 2,1 → 7,3 m/s | 2,2 m | 0,92 | Coup d'envoi 0:30. |
| C8 | `a36c8bed` · `a36c8bed-e6a0-4ba2-be69-916074e8df6f` | **0:41** | MusicMan29 | Kabab1544 | 2,9 → 6,6 m/s | 4,3 m | inconnu | Tout debut de partie. Coup d'envoi 0:37. |
| C9 | `879a4dba` · `879a4dba-8228-4920-83ff-814c5061a3d7` | **4:07** | F0up0udav3851 | seasonedbacon2 | 2,9 → 6,5 m/s | 4,0 m | 0,07 | **Le plus faible** : le porteur regardait ailleurs. Garde pour l'ecart de qualite. |

*Neuf candidats seulement sur tout le parc : la regle est severe (elle rejette notamment les
sauts de position a 90 m/s, qui sont des reapparitions et non des poussees).*

### 2.b — Le film dit « il vient de consommer sa derniere charge de repulseur »

Une **autre** source, independante de la physique. Le film annonce parfois qu'un joueur ne
porte plus son equipement, ce qui veut dire qu'il l'a consomme. Le defaut : cette annonce
n'arrive pas toujours a l'instant du geste — d'ou une **fenetre** plutot qu'un instant.

**CANDIDATS NON MESURES — a confirmer visuellement.** JGtm est au roster de tous ces matchs
mais **n'est pas l'acteur** : il faudra suivre le joueur nomme.

| # | Match | Fenetre a regarder | Joueur | Largeur |
|---|---|---|---|---|
| C10 | `3923bede` · `3923bede-dc47-4f54-9ef5-4a77a4eaf350` | **2:27 → 2:31** | Madina97294 | 4 s — la plus etroite du lot |
| C11 | `e1259a69` · `e1259a69-967e-46e0-864d-8e32425d70f4` | **7:34 → 7:45** | NonPlusExtra | 11 s |
| C12 | `c75f33b8` · `c75f33b8-2448-4f4b-8f00-91803a7d36fa` | 5:54 → 6:43 | (non nomme) | 49 s — large |
| C13 | `a6ae19fb` · `a6ae19fb-9db4-48e9-a601-551ed1a05d0c` | 6:21 → 7:29 | Ninja Maccers | 68 s — large |
| C14 | `0d265ab0` · `0d265ab0-5624-4edb-b5a9-502d4cb24139` | 2:46 → 4:19 | DRghie | 94 s — tres large |

> **Le controle qui donne du poids a cette source** : appliquee au PROPULSEUR sur les films
> deja decodes, elle tombe **9 fois sur 9** a une seconde d'une impulsion mesuree. Elle sait
> donc dater un usage. Reste que pour le repulseur, on n'a rien pour la recouper.

*Le parc entier ne porte que 7 annonces de ce type pour le repulseur. Les cinq ci-dessus sont
datables ; les deux autres sont dans le film `fb1a1a72`, dont l'horloge n'a pas pu etre calee
(l'artefact le dit lui-meme : `originResolved` vaut faux). **On ne peut donc pas dire a quelle
seconde les regarder, et on ne l'invente pas.***

### 2.c — Les vies ou JGtm porte lui-meme le repulseur

**52 vies** au total. Meme sans aucune signature, ce sont les plages ou vous seul pouvez dire
si vous vous en etes servi — et combien de fois. Les plus longues (donc les plus riches) :

| Match | Plage a regarder | Duree |
|---|---|---|
| `72b0a25e` · `72b0a25e-c94d-42c0-85ca-195a320c7b73` | 2:47 → 3:59 | 73 s (puis vous prenez un camouflage) |
| `0d265ab0` · `0d265ab0-5624-4edb-b5a9-502d4cb24139` | 7:18 → 8:15 | 57 s |
| `4577fcc4` · `4577fcc4-174c-47fc-a3ab-91140dd03486` | 2:52 → 3:48 | 56 s |
| `daaa17d6` · `daaa17d6-77a9-4656-a3c2-217be53a1952` | 0:47 → 1:41 | 54 s |
| `7b0d89c4` · `7b0d89c4-a068-4ea2-82b8-8f13ba0311d6` | 8:00 → 8:47 | 47 s |
| `0d265ab0` (autre vie) | 8:41 → 9:23 | 42 s |
| `d9781168` · `d9781168-5fd6-4b00-a862-56e3a0a1f956` | 6:08 → 6:48 | 40 s |
| `3d58eb37` · `3d58eb37-8efe-4042-8315-5b6a0e3a7525` | 2:10 → 2:40 | 30 s |

**Le releve le plus utile de tous serait celui-ci** : prendre UNE de ces vies, la regarder en
entier, et noter chaque fois que vous declenchez le repulseur. Cela donnerait la premiere
verite terrain complete — et c'est elle qui permettra de dire si le film enregistre le geste
quelque part, ou pas du tout.

---

## Partie 3 — Mode d'emploi

### Retrouver le match

L'identifiant court (`1cd3848a`) est le debut de l'identifiant complet du match. Dans LevelUp,
ouvrez la fiche du match par son identifiant complet : vous y trouverez **la carte, le mode et
la date**, ce qui vous permettra de le reperer dans la liste des films du Theater.

### Poser le curseur

1. Le temps donne est **le temps depuis le debut du film**, curseur a zero.
2. **Reculez de 2 secondes** avant l'instant indique, puis avancez au ralenti.
3. **Verifiez d'abord la colonne « coup d'envoi »** : si le decompte d'avant-match se termine
   bien a l'instant indique, toute la colonne des instants est juste pour ce film. Sinon,
   dites-moi ce que le visionneur affiche a ce moment-la — la correction est un simple
   decalage constant.
4. Pour la partie 1, la camera doit suivre **JGtm**. Pour la partie 2, elle doit suivre le
   **porteur** nomme (et regarder qui part).

### Ce qu'il faut noter — le formulaire

Pour chaque creneau, une ligne suffit :

| Creneau | Vu ? | Instant reellement vu | Qui agit | Quel equipement | Certitude |
|---|---|---|---|---|---|
| ex. #1 `1cd3848a` 2:15 | oui | 2:14 | JGtm | propulseur | certain |
| ex. C1 `af3500aa` 1:34 | non | — | — | c'etait une grenade | certain |

- **Instant reellement vu** : a la seconde. C'est cette colonne qui mesure la justesse de
  l'horloge.
- **Certitude** : `certain` / `probable` / `je ne sais pas`. Un « je ne sais pas » est une
  reponse utile — ne le forcez pas.
- **Les « non » valent autant que les « oui ».** Un creneau ou il ne se passe rien dit que la
  lecture invente ; c'est exactement ce qu'on cherche a savoir.

### Le chiffre qui manque encore, et que vous seul pouvez donner

Combien de fois vous servez-vous du propulseur dans une partie ? On sait que le film en rend
au moins un certain nombre (146 sur dix parties pour vous), **mais on ignore combien il en
rate.** Si, sur un seul film — `9e8fb31b` est le plus court et le plus pauvre, donc le plus
economique — vous comptez tous vos usages du propulseur, la comparaison donnera ce chiffre.

---

## Les instruments (pour memoire technique)

| Fichier | Role |
|---|---|
| `apps/go-api/internal/analysis/filmdec/r9_creneaux_research_test.go` | `TestR9Horloge` (controle de la conversion de temps contre le manifeste du jeu, + nom de carte) et `TestR9CreneauxPropulseur` (impulsions nommees, datees, avec leur pic) |
| `r8_i59_tags_research_test.go` et le socle `r8_*` du lot R8 | detection reutilisee telle quelle, non rejouee |

Commande (depuis `apps/go-api`, `<D>` = `data/cache`, `<W>` = racine du worktree) :

```
CGO_ENABLED=0 R9_FILMS=<D>/film_chunks R9_ARTIFACTS=<D>/replays/halo_infinite \
  R9_MANIFESTS=<D>/film_manifests \
  R8_BOUNDS=<W>/data/titles/halo_infinite/reference/map_quant_bounds.json \
  R9_IDS=1cd3848a go test ./internal/analysis/filmdec/ \
  -run '^TestR9(Horloge|CreneauxPropulseur)$' -count=1 -timeout 60m -v
```

Cout : environ 25 s par film pour les creneaux, moins d'une seconde pour le controle
d'horloge. Aucune ecriture, aucune base de donnees ouverte, aucun commit.

Les trois analyses de la partie 2 sont des lectures d'artefacts uniquement (aucun film
decode) ; leurs regles sont recopiees dans le corps du document, chaque seuil avec sa raison.
