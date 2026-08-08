# Corpus objectifs — octets reels de film Halo Infinite

Ce dossier ne contient AUCUN code. Ce sont des octets de film Halo Infinite, preleves sur
des matchs reels, et les mesures qui en ont ete tirees. Il n'appartient a aucun paquet Go :
c'est deliberé, pour qu'il survive au decodeur qui l'a produit.

## D'ou il vient, et pourquoi il est ici

Ce corpus a ete constitue le 2026-08-07 (lot 2 de la v7.5) pour contraindre le decodeur
`internal/analysis/objectivescore`. Ce paquet a ete **supprime le 2026-08-08** (lot 3 de la
v7.5) : son successeur `internal/analysis/objectiveevents.ScoreCurve` fait mieux sur tous
les axes — per-joueur, a la milliseconde, sans calibration.

Le corpus, lui, ne meurt pas avec lui, et c'est le point : **ces octets ne decrivent pas un
decodeur, ils decrivent un format**. Le jour ou la source evenementielle peuplera reellement
la courbe de score des modes a zones et KOTH (chantier de pipeline film, cf. `.ai/HANDOFF_MODE_SCORE_CHAINE_2026-08-01.md`
§6), il faudra des octets reels dates, sourcés et versionnes pour la contraindre. Les
reprelever couterait un acces au cache film d'un poste precis et a des matchs precis ; les
garder coute 400 Ko.

## Ce qu'il y a dedans

| Fichier | Nature |
|---|---|
| `minibobine_objectifs/strongholds_7344d24f/` | 8 chunks TYPE-2 ancres + manifeste + provenance |
| `minibobine_objectifs/koth_0a247154/` | 8 chunks TYPE-2 ancres + manifeste + provenance |
| `minibobine_objectifs.golden` | mesures sur les deux mini-bobines ci-dessus |
| `films_reels.golden` | mesures sur 6 films complets du cache disque |

Chaque chunk est un **prefixe d'octets bruts**, tronque juste apres son premier paquet
TYPE_2 complet, range en zlib (la forme employee par le cache film lui-meme). **Aucun octet
n'est reecrit.** La troncature etait licite parce que le decodeur d'origine etait sans etat
d'un chunk a l'autre ; la recette de fabrication refusait d'ecrire si le prefixe ne decodait
pas exactement comme le chunk entier.

Les mini-bobines portent les **8 DERNIERS** chunks ancres de leur film, pas les 8 premiers :
mesure faite a l'epoque, sur les 8 premiers de `0a247154` le score KOTH vaut encore 0-0 — une
bobine qui fige des zeros ne distingue pas un decodeur juste d'un decodeur muet.

## Les oracles, et ce qu'ils valent

Chaque film du corpus porte un oracle **sourcé** dans `films_reels.golden` (ligne
`# oracle :`). Un oracle sans provenance ne vaut pas mieux qu'une fixture auto-validante.

| Film | Mode | Final API | Oracle |
|---|---|---|---|
| `7344d24f` | Arena:Strongholds | 193-112 | calibration d'origine + test cache-backed |
| `696a9d7c` | Arena:Strongholds | 200-94 | `ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md` §2 (Vagabond, 561 s) |
| `0a247154` | Ranked:King of the Hill | 4-2 | seul cas VALIDE EXACT du decodeur supprime (variante B) |
| `01e1f945` | KOTH:Arena | 3-2 | `ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md` §2 (Catalyst, 540 s) |
| `606d9844` | KOTH:Arena | 105-8 | `ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md` §15 |
| `8076f97f` | KOTH:Arena | 78-105 | `ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md` §15 |

## Statut des deux `.golden` : documents de mesure, plus des oracles de test

Aucun test ne lit plus ces fichiers — ils sont partis avec le paquet. Ils sont conserves
pour ce qu'ils **mesurent**, et cette mesure est le verdict qui a decide la suppression :

- **La courbe Strongholds n'est pas un score per-equipe.** La valeur brute team0 plafonne a
  **50** sur les deux films Strongholds, dont les finals API sont 193 et 200 ; team1 vaut
  **32** sur les deux, pour des finals 112 et 94. Elle ne « retombait sur le final » que
  parce qu'on la calibrait dessus.
- **Les colonnes calibrees ne repondent d'aucune position de bit.** La calibration remettait
  la derniere frame exactement sur le final API : toute assertion « la derniere frame vaut le
  final » etait vraie par construction, quel que soit l'offset lu. Ce sont `bitpos`, `brut0`
  et `brut1` qui repondent du token, de sa fenetre et des offsets.
- **Ce que le token lit est reel et reste utile** : `token=0x7B6`, fenetre `[835,912)`,
  positions de bit stables par film. C'est cette partie-la qui contraint le format.

Un futur decodeur evenementiel qui pretendrait lire un score de zones ou de KOTH dans ces
memes octets doit pouvoir se confronter a ces valeurs.

## Regeneration

Il n'y a plus de recette executable : elle vivait dans `minibobine_test.go`, supprime avec le
paquet. Pour la retrouver, l'historique git la conserve :

    git log --diff-filter=D -- 'apps/go-api/internal/analysis/objectivescore/*'
    git show <sha>:apps/go-api/internal/analysis/objectivescore/minibobine_test.go

Les films complets, eux, n'ont jamais ete versionnes (cache disque local, variable
`FILM_CACHE_ROOT`).
