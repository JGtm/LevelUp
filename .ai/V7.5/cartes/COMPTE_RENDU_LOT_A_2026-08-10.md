# Compte rendu — lot A « fonds de carte » (2026-08-10)

> Pour l'orchestrateur du chantier v7.5. Branche `feat/v75`, worktree `LevelUp-wt-replay2d`.
> Aucun merge, `main` intact. Detail technique : `PLAN_PORT_TRIANGLES_GO.md` §3, §7 (D3, D4) ;
> reports : `.ai/V7.5/REGISTRE_REPORTS.md` ; chiffres carte par carte :
> `RAPPORT_CUISSON_FONDS_2026-08-10.md`.

## En une phrase

Le livrable du lot A existe : **21 fonds de carte figes et versionnes**, avec leur calage publie,
produits par une chaine sortie des tests et branchee sur `PathResolver`. Le chantier peut
avancer ; ce qui reste sont des defauts de RENDU nommes et chiffres, pas des inconnues.

## Ce qui est livre

| | |
|---|---|
| assets | 21 : 19 cartes natives + Vagabond et Corpo (Forge) |
| ou | `data/titles/halo_infinite/reference/map_backgrounds/{cle}.png` + `{cle}.json` |
| format | PNG RGBA, fond transparent, 0,0920 m/px, cadre par carte |
| calage | publie dans le sidecar avec sa formule en clair, plus les stats de cuisson |
| chaine | `internal/himap/cuisson.go` · `cuisson_forge.go` · `fond_png.go` |
| orchestration | `cmd/mapfond-build` (hors ligne, GPLv3 confine) |
| lecteur | `internal/analysis/replay/map_background.go` — cote consommateur, GPL-propre |

**Rien de tout cela n'existait** : avant ce lot, les PNG etaient des artefacts de revue ecrits
par des tests a des chemins arbitraires du Bureau.

## Gates

| gate | resultat |
|---|---|
| `TestBancCliffhanger` | accord **64,7 %**, positions jouees **93,95 %** — RE-BASE (voir « la decision » ci-dessous) |
| determinisme | deux cuissons independantes rendent des PNG **identiques au bit** (17/17 verifies) |
| licence GPLv3 | `cmd/server` ne linke pas `internal/ooz` — 108 paquets parcourus, mutation jouee |
| calage publie = image produite | temoin avec mutation (inverser le bord Y rougit a 3,68 m) |
| `go build` / `go vet` | propres |
| `TestBalayageCoquille` | **assertion AJOUTEE** — il ne faisait que journaliser, il rendait `ok` quoi qu'il arrive |

## La decision qui a ete prise, et son cout

La tranche d'altitude etait **absolue** ; le niveau joue des cartes va de **-136,7 m** (Chasm) a
**+77,3 m** (Deadlock). Elle decapitait les cartes qui ne jouent pas vers zero : Chasm se
reduisait a deux traits, Highpower etait aux deux tiers vide. **Bascule en relative au sol joue,
sur decision utilisateur, images a l'appui.**

Gain : Chasm 5/17 -> 17/17 ancres, Highpower 14/51 -> 38/51, et Deadlock (+77 m) devient
possible. Cout : sur Cliffhanger, l'exces passe de 33,8 a 39,1 % et l'accord de 66,7 a 64,7 % —
la vallee entre dans le cadre. Le banc est re-base par ecrit.

## Ce qui reste, nomme et chiffre

**1. Le rendu des cartes couvertes — le vrai sujet restant.**
Verdict utilisateur au gate visuel : Illusion, Prism et Aquarius rendent les TOITS, pas l'arene.
Le z-buffer garde la surface la plus HAUTE ; sur une arene couverte c'est le plafond. Forbidden
et Chasm restent juges non satisfaisants apres la bascule.
Piste : `SurfaceReference` / `CarteParReference` (`reference.go`) retiennent deja « la bande la
plus PROCHE du sol de reference » au lieu de « la plus haute » — ecrites pour le `Volume`,
jamais branchees sur le `Rendu`. **A mesurer sur l'oracle fort AVANT de brancher** : sur
Cliffhanger, les rochers hauts font partie de l'identite de la carte.

**2. Une regression sur Catalyst, et la lecon qui va avec.**
La bascule a change l'image de Catalyst (+11,5 % de matiere). Elle n'a PAS ete detectee, parce
que le seul temoin disponible etait l'oracle des ancres — qui dit « 24/24 avant et apres » et ne
regarde pas l'aspect. **Un oracle faible presente comme un feu vert visuel : c'est le piege n°3
du chantier, rejoue.** Toute modification de la regle de rendu doit desormais passer par un
artefact visuel sur au moins Cliffhanger ET Catalyst.

**3. Question ouverte posee par l'utilisateur : regles generales contre regles par carte.**
Position argumentee, a trancher au prochain lot : une seule carte possede un oracle fort, donc
un reglage par carte ne peut etre valide qu'a l'oeil, carte par carte, avec ~100 cartes Forge
derriere. Ce qui reste defendable est une **exception declaree en DONNEE** (table + raison
ecrite + oracle), jamais une branche dans le code.

**4. Portee des cartes.**
31 dossiers installes, dont 2 tutoriels, 1 PvE et 8 canevas Forge -> **20 cartes natives**, dont
19 cuites. La 20e, `sgh_interlock` (Live Fire), ne porte aucun tag sbsp — exception instruite,
desormais une sentinelle (`himap.ErrAucunTagSbsp`) et non plus un echec.
Le gisement suivant est ailleurs : **199 `.mvar` au dump**, dont deux seulement sont branches
(Vagabond, Corpo). La chaine Forge existe et fonctionne.

## Deux pistes REFUTEES pendant ce lot, a ne pas rejouer

- **Apparier carte -> dossier par les ANCRES** : NO-GO mesure. Chaque carte Halo est batie
  autour de SA propre origine ; les boites monde se recouvrent, une position monde ne designe
  aucune carte. Le temoin a refute la regle deux fois (horizons kilometriques, puis contradiction
  avec ~20 appariements connus). Code supprime.
- **L'ecart median aux ancres comme predicteur du defaut des toits** : faux. Behemoth (-12,73 m)
  est juge nickel, Forbidden (-0,35 m) est touche.

Ce qui a marche a la place : le depot de variantes porte le lien en clair
(`deadlock_btb_drydock.mvar`), **valide sur les 18 appariements deja connus, 0 manquant**. Il
rend Deadlock, Oasis, Scarr et Corpo.

## Ce que le lot suivant devrait prendre

Par ordre de valeur decroissante :

1. **Les toits** (defaut de rendu, 4 cartes touchees) — c'est ce qui separe « ca ressemble a la
   carte » de « c'est la carte ».
2. **Brancher le fond de carte cote app** : aucun consommateur ne lit encore
   `map_backgrounds/`. Le calage est publie et teste, le lecteur existe.
3. **Le gisement Forge** : 199 `.mvar`, deux branches. Prealable : le lot B (1 113 objets sans
   modele `rtgo` direct sur Vagabond, soit 23,6 %).
