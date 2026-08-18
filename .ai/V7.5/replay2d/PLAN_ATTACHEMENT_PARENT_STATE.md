# Plan — Attachement d'un objet a un autre (`object-parent-state`, i10) : joueur DANS un vehicule, drapeau PORTE

> Ecrit le 2026-08-18 par la session de pilotage, a la demande de l'utilisateur (« dans le code on
> ne sait pas si un joueur est dans un vehicule ; comment les deux composants sont rattaches ? » ;
> « avoir la trajectoire du drapeau plus finement ; quand il sort de la zone jouable il respawn a
> son emplacement »). Contrat `plan-execution`. Mesure d'abord, production ensuite. Worktree FRERE.

## Comment un moteur fait ca (le modele general, et ce qu'on a deja sous la main)

Un objet « attache » (Spartan assis dans un vehicule, drapeau tenu, arme en main) n'est pas relie
par un evenement mais par une RELATION PARENT-ENFANT portee par l'ENFANT : un handle vers l'objet
parent + un point d'attache (siege / marqueur) + eventuellement une transformation locale. Tant
qu'il est attache, l'enfant NE replique PLUS sa position monde (elle se deduit du parent) ; quand il
se detache, il redevient un objet du monde qui replique sa position. C'est exactement ce que nos
mesures ont montre par ricochet : un objet pose « cesse d'emettre sa position » (poses, armes au
sol), et le drapeau porte se lit CHEZ LE PORTEUR (marqueur `0x00010005`, item 4 phase 0).

Dans le film, ce lien a un nom et il est DEJA PARSE : le composant **`object-parent-state-component`
(i10)**, present sur le bipede (ti=35), l'equipement (37), le corps rigide (38), le vehicule (40),
36 et 39 — table ECS `filmdec/testdata/ecs_table.tsv` (« le PARENT de l'entite (porte par un vehicule,
attache a un autre objet). Non mesure. »). Le deser Go `consumeObjectParentState`
(`filmdec/components_object_state.go:47`, miroir de `FUN_140c1e4d0`) lit : porte R(1) ; si porte :
un quant-stat 16 bits (handle du parent presume), R(16), R(16) optionnel, 2 bits, matrice 3 x R(16)
(offset/orientation d'attache), vitesse R(19) optionnelle, R(8), R(1) ; sinon un bloc `1408f0ac4` +
R(11) optionnel ; queue commune. **Tout est CONSOMME et JETE** : la grammaire est la, les valeurs ne
sont pas gardees. Aucune grammaire manquante, aucun Ghidra necessaire pour la premiere mesure ; Ghidra
ne sert que si le sens d'un champ (quel bit = quel handle) resiste a la mesure.

## Decisions tranchees

1. On lit les VALEURS d'i10 (hook de mesure, comme `SetEquipmentCreationHook`), sur le chemin DELTA
   (image par image) — pas d'image-cle, pas de bit-exact.
2. Deux oracles EXACTS existent deja : (a) CTF : `FlagGrabs`/`FlagSteals`/`FlagCaptures`/`FlagCarriersKilled`
   par slot a la ms (item 4) — le drapeau OBJET doit passer porte=1 avec un handle qui resout au slot
   du porteur au grab, et porte=0 a la fin ; (b) vehicules : un film BTB a vehicules (`084a804d`
   Fortitude Heavies ; verifier `ti=40` present) — un bipede a bord doit passer porte=1 avec un handle
   vers un slot `ti=40`, et sa position propre doit cesser d'emettre (ou emettre celle du vehicule).
3. Le drapeau OBJET : chercher d'abord dans les creations `ti=42` ECARTEES par le croisement d'identite
   (mot MPP hors catalogue d'armes) sur les 3 films CTF, pres des positions/instants de drop et de
   `flag_spawn` — s'il y est, on a sa trajectoire propre a l'image (lache, jete, hors zone => nouvelle
   creation au socle) ; sinon balayer les autres archetypes (37, 38) avec le meme oracle.
4. Seuils AVANT mesure : (a) >= 90 % des grabs suivis dans les 2 images d'un passage porte 0->1 de
   l'objet drapeau (ou du porteur : sens a etablir) avec handle -> slot du porteur ; temoin : un autre
   slot au hasard <= 5 % ; (b) vehicules : >= 90 % des periodes « bipede immobile dans le repere du
   vehicule » (position du bipede == position du vehicule a < 1,5 m sur >= 3 s) ont porte=1 avec
   handle -> ce vehicule ; temoin <= 5 %. Sinon negatif ecrit.
5. Production seulement apres les deux mesures : `attachments` [{child slot, parent slot, t0, t1,
   kind}] dans le document (schema +1), le drapeau publie sur SA piste quand il est libre et sur celle
   du porteur quand il est porte, le joueur en vehicule marque (icone) — lot separe, plan amende.

## Phases

- [x] 0.1 Hook de lecture d'i10 (valeurs brutes par record : ti, slot, gen, t, porte, champs) — instrument
      sous garde `ATT_FILM`, lecture seule. UNE edition de production, minimale et declaree :
      `filmdec.SetObjectParentStateHook` + le type `ObjectParentState` dans
      `components_object_state.go` ; le deser garde desormais les valeurs qu'il jetait, AUCUN bit
      lu ne change (memes lectures, meme ordre, memes largeurs ; la sonde est appelee en `defer`
      apres coup). Instrument : `replay/attachement_phase0_socle_test.go` (marche stateful
      `DecodeFrameViews`, rattachement EXACT lecture -> record par `CompResult.StartBit`).
- [ ] 0.2 CTF (`64e8adfa`, `530820e5`, `53ce4390`) : le drapeau OBJET — creations `ti=42` ecartees
      (mot MPP hors catalogue) : combien, ou (distance aux `flag_spawn`), quand (distance aux grabs /
      drops de l'oracle) ; ses passages porte 0/1 vs l'oracle ; handle -> slot porteur ; seuil 4(a).
- [ ] 0.3 Vehicules (`084a804d` + un second film BTB choisi sur preuve `ti=40` present) : bipedes
      porte=1, handle -> `ti=40`, coincidence de positions ; seuil 4(b) ; temoin.
- [ ] 0.4 Publier au journal du plan (denominateurs, verdicts) ; sens des champs d'i10 etabli ou
      « non etabli » champ par champ.

**Gate 0** : au moins UN des deux oracles tenu (>= 90 %, temoin <= 5 %) => plan de production ecrit
(decision 5) ; aucun => negatif ecrit, condition de reprise = Ghidra sur `FUN_140c1e4d0` (sens des
champs) puis remesure.

## Regles dures

Mesure avant production ; seuils jamais rebaisses ; un seul decodage filmdec par process ; aucune
base ; films par chemin absolu ; JAMAIS `git add -A` ; ni journal ni registre depuis le frere (textes
au CR) ; RE image-cle fermee (rien ici ne la rouvre : chemin delta seulement).

## Decouvertes (hors perimetre — notees, NON traitees)

_(vide)_

## Journal du plan

- 2026-08-18 — plan ecrit ; phase 0 lancee (agent Opus, worktree frere `../LevelUp-wt-attache`).
