# PLAN — publication de l'origine des ramassages (lot final)

Branche `wt/origine-equipement`. Contrat : skill `plan-execution` (ordre strict, une etape a la
fois, aucun report d'action faisable, statut sur chaque item).

Prerequis leve avant l'etape 1 : `origin/feat/v75` etait deja en **SchemaVersion 31** (chantier
`wt/pickup-nommage`, le NOM de l'objet ramasse). L'amont a ete integre par MERGE (21 commits,
aucun conflit) AVANT toute numerotation. Le schema de ce lot est donc **32**, pas 31.

Decouverte a l'integration : le lot parallele conclut, dans
`.ai/V7.5/film_re/NOTE_ORIGINE_LEVITATION_2026-09-01.md`, qu'il manque « un catalogue de points
d'apparition d'equipement extrait des `.mvar` » et que « son absence est ce qui bloque la branche
point d'apparition depuis deux lots ». C'est exactement ce que la recette de ce worktree produit.

## Etape 1 — typer les 16 points de la recette

- [x] 1.1 Types resolus — mais par la TROISIEME voie, les deux premieres etant refutees
- [x] 1.2 `fosp` sonde : NON ELUCIDE, consigne (ses references ne resolvent dans aucun module)
- [x] 1.3 GATE A passe : 66/66 ramassages non-arme nommes par le manifeste ; les points
      d'equipement/grenade recoupent bien les classes 2/3
- [x] 1.4 GATE B passe : 47,6 % des armes sur les socles d'armes prouves contre un temoin de
      densite a 20,0 % — 2,4x le hasard

## Etape 2 — regenerer `map_weapon_pads.json` par la recette

- [ ] 2.1 La recette passe du test de recherche au code du generateur, avec garde-rail
- [ ] 2.2 Telechargements : uniquement les `.mvar` MANQUANTS, plafond raisonnable, auth JGtm
- [ ] 2.3 GATE NON-REGRESSION : les entrees ARMES des 72 cartes STRICTEMENT identiques ;
      diff structurel avant/apres = ajouts seulement (`kind` nouveaux equipment/grenade)

## Etape 3 — publier l'origine dans le document de rejeu

- [ ] 3.1 Champ `origin` sur les ramassages non-arme : `spawner` / `ground` / absent
- [ ] 3.2 Schema 32 (courant verifie = 31), contrat openapi + generated + contracttest
- [ ] 3.3 Gate golden nomme
- [ ] 3.4 Compteurs de couverture, DONT le compteur de CARTE ABSENTE du catalogue
      (decision produit : le trou se VOIT, il ne se telecharge pas pendant la cuisson)

## Etape 4 — cuisson pilote, UN SEUL film

- [ ] 4.1 Catalyst `01e1f945`, une invocation, avant-plan, `filmproc.Arm` 3 GiB,
      zero process concurrent verifie AVANT, ecriture dans le worktree
- [ ] 4.2 Comptes `origin` de l'artefact confrontes aux mesures de recherche

## Etape 5 — cloture

- [ ] 5.1 Gates Go complets
- [ ] 5.2 Gates web si types regeneres
- [ ] 5.3 Note + thought_log
- [ ] 5.4 Commits locaux (pas de merge : une revue adversariale relit avant)

## Doctrine rappelee par le coordinateur

La cuisson reste OFFLINE-PURE : jamais de telechargement pendant la generation d'artefact. Les
trous de catalogue se COMPTENT et se comblent par la CLI / le sync, pas par la cuisson.

## Journal

### Etape 1 — CLOSE. Deux voies refutees, la troisieme livre.

**Voie 1, la chaine de tags — REFUTEE.** type_id -> `food` -> `foki` -> `fosp` -> objet
engendre : elle s'arrete au `fosp`. Ses references ne resolvent dans AUCUN module indexe
(96 804 entrees), et le manifeste d'equipement du titre ne recoupe pas une seule reference des
16 types. Resultat brut : 13 INDETERMINE sur 16. `fosp` reste NON ELUCIDE.

**Voie 2, les naissances `ti=37` — REFUTEE, et le diagnostic est net.** Sur 283 naissances lues,
0 nommee : 244 sans identite transmise et 39 avec identite mais hors manifeste — **39
identifiants distincts, une occurrence chacun**. C'est la signature d'un identifiant d'INSTANCE,
pas d'un identifiant de catalogue. `AbilityID` n'est pas la cle du manifeste sur ce canal.

**Voie 3, le canal natif des ramassages — ELLE LIVRE.** `BipedPickup.CatalogID` est un
identifiant de catalogue (mesure du depot), et c'est la table du manifeste qui a etabli les
classes 2/3 au schema 31. Sur Catalyst : 199 ramassages, 66 non-arme, **66 nommes sur 66**,
41 apparies a moins d'un metre d'un point catalogue.

**Le temoin est le TAUX DE BASE du film** : 66 non-arme sur 199 = 33,2 %. Un point qui rend
33 % de non-arme ne porte aucune information.

| type_id | armes | gren. | equip. | non-arme | lecture |
|---|---|---|---|---|---|
| `0x5F379533` | 13 | 0 | 0 | 0,0 % | socle PROUVE `power` — coherent |
| `0x6253CFC0` | 7 | 1 | 3 | 36,4 % | socle PROUVE `rack` — au taux de base, bruit pur |
| `0x5E86D110` | 0 | 0 | 4 | 100,0 % | socle PROUVE `powerup` — 4 `powerup_overshield` |
| `0xADEEE6D8` | 3 | 15 | 1 | 84,2 % | point NON-ARME, contenu 15/16 grenade |
| `0xE42158DF` | 1 | 0 | 11 | 91,7 % | point NON-ARME, contenu 11/11 equipement |
| `0x0CD504B0` | 10 | 3 | 0 | 23,1 % | SOUS le taux de base — pas un point d'equipement |
| `0x5F3FA667` | 4 | 0 | 0 | 0,0 % | sous le taux de base |
| `0xAEDF9CF0` | 1 | 0 | 2 | 66,7 % | n = 3, pas de majorite |
| `0x2BEF1E2D` | 2 | 0 | 0 | — | n < 3 |
| `0xF8EC7E1E` | 1 | 0 | 1 | — | n < 3 |

**CE QUI VALIDE LA METHODE** : les trois socles PROUVES tombent exactement ou ils doivent, et
aucun n'a servi a calibrer quoi que ce soit. C'est le controle le plus fort du lot.

**SEUIL MANQUE, ET PUBLIE COMME TEL.** Le seuil ecrit avant la mesure etait « n >= 3 ET 80 %
d'une meme classe sur les trois ». `0xADEEE6D8` rend 15 grenades sur 19 = 78,9 % : il le manque
d'une observation. Le seuil n'a PAS ete deplace apres coup. La lecture retenue pour le catalogue
est en DEUX temps, et le second est une hypothese, pas une preuve :

1. *ce point est-il non-arme ?* — critere ECRIT AVANT (part non-arme > taux de base) :
   `0xADEEE6D8` (84,2 %) et `0xE42158DF` (91,7 %) le passent franchement ;
2. *grenade ou equipement ?* — juge DANS les non-armes, effectifs publies : 15/16 grenade pour
   `0xADEEE6D8`, 11/11 equipement pour `0xE42158DF`. Une carte, un film : a confirmer.

**Decouverte** : les deux `type_id` trouves a la main au lot precedent, par pure geometrie, sont
exactement les deux que le canal natif type — l'un grenade, l'autre equipement. Deux chaines
independantes, meme resultat.

## Decouvertes (notees, NON traitees)

(vide)
