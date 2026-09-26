# Lot C — arbitrage superviseur apres la phase 1b (2026-08-18) : lot CLOS sur son objectif produit, acquis conserves

> Lu sur pieces : `LOTC_PHASE1B.md`, `LOTC_gates.log`, table ECS (ti=12 i14, ti=10 i1, i26-i29 `porte`),
> `components_managed_object.go` (202 L), commit `9f46cad93` sur `wt/zones-film`.

## Ce que la phase 1b livre (conserve, fusionne dans l'integration)

1. Trois canaux PORTES en production, par le modele du lot 0 (le deser publie, hooks nommes) :
   `radial-progress` (R(8) -> [-1,+1]), `boundary-color` (4xR(8) RGBA), `rtpc` (R(32) id + [R(22)]) ;
   14 vecteurs sur octets reels verts ; `DesyncAt` : aucun film ne recule, 7 progressent ; table ECS
   editee dans le meme commit (G1 vert) ; `traverse.go` +6 lignes de routage seulement.
2. `radial-progress` EST la jauge de capture : 95,8 % et 90,9 % des captures de zone (Strongholds)
   sont precedees d'une rampe atteignant son max a ± 2 s. Le temoin (36,6 % / 28,6 %) est SOUS le
   niveau du hasard (46,1 % / 39,9 % — 50-62 rampes en 9 min) : la discrimination est reelle (x2)
   mais le seuil ecrit (<= 20 %) supposait un canal rare. Le canal est dense.
3. `boundary-color` est une couleur ANIMEE (996 quadruplets distincts) — un rendu, pas un etat ; les
   « 4 niveaux » de la phase 1a etaient un artefact de l'echantillon singleton.
4. Les 32 drapeaux de ti=10 i0 : granularite fine (32 bits utilises, 1 162 valeurs), ni joueur, ni
   equipe, ni etat.
5. `rtpc` suit une progression (2 identifiants couvrent 95,3 %), pas un volume.

## Verdict

**G-C1 NON TENU, GATE 2 NON TENU, sur l'objectif produit du lot** (« qui tient quelle zone, depuis
quand ») : aucun des trois canaux ne porte un ETAT DE ZONE ENUMERABLE a apparier au catalogue, et
il n'y a donc pas d'objet pour le pont slot -> zone. Le lot C est CLOS `[!]` sur cet objectif — le
troisieme critere requalifie d'affilee serait un abus : ici la mesure ne dit pas « le critere est
inadapte », elle dit « le canal ne porte pas l'etat ». Les seuils restent tels qu'ecrits.

Ce qui reste OUVERT et TRACE (registre) :
- **La progression de capture** (`radial-progress`) est publiable comme JAUGE, sans identite de zone
  dans le record : en KOTH (une colline a la fois) c'est la progression de la colline unique — sous
  reserve de dedoublonner les navpoints qui rampent en parallele (memes valeurs aux memes instants =
  la meme capture vue par plusieurs marqueurs : a mesurer AVANT toute publication) ; en Strongholds
  la rampe s'apparie a la zone par la capture qui la clot (95 %), les rampes avortees restent sans
  zone. Publication `captureProgress` = decision produit a prendre APRES le verdict de la sonde F5
  (ti=13 : noms de proprietes lisibles ?) et de la phase 2 du plan item 4 (colline par periode) — les
  deux disent si un porteur d'IDENTITE de zone existe.
- **ti=13** (`managed-object-property`, `player-masked-property`) reste le seul candidat d'ETAT de
  zone par equipe en delta (slots stables, 40x) : STOP de RE maintenu jusqu'au verdict F5.
- Le plan item 4 GARDE sa phase 2 (colline par grappes de positions) : rien ici ne la couvre.

## Consequences pour les seuils futurs (lecon, a appliquer aux lots D, P, E)

Tout temoin sur un canal DENSE se juge contre son NIVEAU DE HASARD publie (fenetre x densite), pas
contre un pourcentage fixe ; un seuil sur un compte publie son denominateur.
