# HANDOFF — Chantier modes a porteur (etat au 2026-08-28)

> Ecrit apres la session qui a livre la couronne VIP. Destine a la session (Claude ou humaine)
> qui reprend. La memoire `project-modes-porteurs-chantier` porte le detail chiffre ; ce
> document donne l'ETAT, la SEQUENCE et ce qui est PARKED.

## 0. Etat consolide (ne pas re-faire)

- **feat/v75 = `7f169304e`** (pousse, CI lancee). Contient : COURONNE VIP publiee (schema 22,
  contrat 40, calque `vipCrownLayer.ts`, table `ObjectiveTypeVip` comp 22 A), lot C (Land Grab
  cable, Live Fire au catalogue de bornes -> corpus Oddball 5 films, Lattice). Gates verts.
- **wt/ti11-cadre** (pousse, NON fusionne) : grammaire ti=11 34 feuilles RESOLUE et validee
  (atterrissage 90,32 %) = asset de recherche. **NE PAS rejouer keyframe/delta ti=11** : l'etat
  vivant de `managed-objective` est CALCULE COTE CLIENT (ni keyframe = defaut, ni delta = bruit).
  Ne pas fusionner (publie rien, eviter code mort).
- Menage worktrees (wt/catalogues, wt/resolution fusionnes ; wt/oddball-porteur, wt/assaut
  stale) : non urgent.

## 1. Ce qui est LIVRE / ACQUIS

- VIP : couronne a l'ecran (comp 22 A = `TimesSelectedAsVip`, periodes bornees par les morts du
  kill feed, 24/24 joueurs au sub-seconde). LECON : discriminant = EXACTITUDE PAR JOUEUR, pas
  couverture/permutation ; temoin = plancher analytique sum(p_v²).
- Assaut : POSEUR acquis (statborg comp 0 A = explosions par joueur) — publiable, pas encore
  publie.
- Land Grab : cable, sert les matchs futurs.
- Grammaire ti=11 : complete (recette RE fermee, mystere R7 resolu).

## 2. SEQUENCE de reprise (le user NE PEUT PAS rejouer Oddball maintenant)

Tout est STATBORG/Go (voie prouvee par le VIP et le drapeau) SAUF l'item 1b. Builds Go
SERIALISES (une seule session Go a la fois).

1. **Assaut** — (a) publier le POSEUR (patron couronne VIP) ; (b) RE de la VARIANTE DE MODE
   pour trouver l'objet BOMBE (la bombe n'est PAS dans le map.mvar ; les sites C2 etaient des
   navpoints generiques). **1b PEUT utiliser Ghidra.** Si la bombe est nommee -> vies libres au
   patron du crane libre.
2. **Oddball A4 somme-d'equipe sur les 5 films EXISTANTS** — la SEULE voie Oddball sans parties
   fraiches : methode A4 (immune au pont, JAMAIS appliquee a Oddball) + colonne
   `kills_as_skull_carrier` (jamais confrontee) + `skull_grabs` (near-miss 80 %). Peut donner du
   TEAM-level, pas forcement l'individuel. Pur Go. Protocole pre-enregistre : scratchpad
   `PROTOCOLE_REMESURE_ODDBALL_VIP.md` (si perdu, re-deriver du registre).
3. **Extraction** (2 films BTB : FG, Refuge) — mode a ZONES : etat via ti=13. ATTENTION ti=13
   illisible en BTB/arene (lot A3) : risque reel. Pur Go.
4. **Stockpile** (2 films) — graine = objet porte, flag-analog (`seed_grabs` -> table
   `ObjectiveTypeSeed` comme VIP/flag, transpose flag_carries). Pur Go.
5. **Sons** des modes livres (couronne VIP...).

## 3. PARKED (attend le user)

- **Porteur Oddball INDIVIDUEL** : toutes voies offline epuisees (keyframe/delta ti=11, statborg
  sous-puissant, proximite 5 campagnes). VOIE DRAPEAU viable (nommer `skull_grabs` + transposer
  `flag_carries`, EXACTEMENT la recette VIP) mais BLOQUEE PAR LE CORPUS (grabs rares sur 5 films).
  **Reprise = le user REJOUE 5-10 Oddball (multiplier les prises)** -> le sync capture les films
  -> slot nommable. Corpus historique borne a 7 films DEFINITIF (vieux matchs sans manifest, API
  404).
- **Capture Cheat Engine** (plan B, si la voie drapeau echoue meme sur corpus frais) : verite
  terrain Theater, jeu lance, pont MCP `cheatengine` (memoire `reference-cheatengine-mcp-setup`).
  EXIGE un film REJOUABLE en Theater — un match FRAIS (le film Dredge 27/12 risque expire). Donc
  la capture CE demande AUSSI que le user rejoue.

## 4. Ghidra ou pas (pour la question du user)

- **Besoin de Ghidra** : seulement l'item 1b (trouver l'objet bombe dans la variante de mode).
- **PAS besoin de Ghidra** : items 2, 3, 4, 5 + 1a (pur Go/statborg). Une session sans Ghidra
  peut les prendre — MAIS builds Go serialises, donc une seule session Go a la fois (ne pas
  lancer deux sessions qui compilent en parallele : corruption du cache Go).
