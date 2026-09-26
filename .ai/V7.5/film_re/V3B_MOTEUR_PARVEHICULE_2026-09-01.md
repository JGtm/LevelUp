# V3B — Le moteur de DEPLACEMENT est DISTINCT par vehicule (refutation du rapport V3 rev7)

> Worktree `LevelUp-wt-vehicules`. Lecture seule sur les donnees du jeu, aucun commit, aucune
> ecriture DuckDB, aucun nouveau Python (scripts `_outils` existants + PowerShell/ffmpeg
> d'analyse). Un seul chargement du module 7,24 Go (mode `eqip-arbre`), OS-cache.
>
> **Mandat** : le rapport V3 rev7 concluait « les 13 vehicules partagent la meme boucle de
> mouvement (lsnd 06ba1096 -> banque e793c135) ; per-vehicule non extractible, seul le RTPC de
> vitesse varie ». L'utilisateur refute PHYSIQUEMENT : un RTPC ne change que hauteur/volume, pas
> le TIMBRE — un souffle d'antigrav, un grondement de chenilles et un moteur a combustion sont
> des contenus spectraux differents, donc des samples distincts DOIVENT exister. **Il a raison.
> La conclusion rev7 est FAUSSE.** Le moteur distinctif par vehicule est bien dans les donnees.

## 1. Verdict

Chaque vehicule a **sa propre banque**, **son propre evenement de moteur**, et **ses propres
wems**. Le `lsnd 06ba1096 -> e793c135` que rev7 prenait pour « le moteur » est un **lit commun**
(presence/grondement de fond) pose SOUS le moteur propre a chaque vehicule — exactement le
« lit commun » que l'utilisateur soupconnait. Preuve sur pieces, trois axes convergents.

## 2. La faute de rev7, en une phrase

rev7 a cherche le switch dans **la mauvaise banque** (`e793c135`, le lit commun partage — qui
n'a effectivement aucun switch) et en a conclu « aucun switch nulle part ». Le switch de
regime **existe**, mais **dans la banque propre a chaque vehicule**. rev7 avait meme TROUVE les
evenements de moteur par vehicule (`donnees/lot2_moteur.json` : 9 IDs distincts) mais n'avait pas
resolu leurs wems, parce que ces wems vivent embarques (banque) + streames (pck), et que la
version embarquee est un **prefetch tronque (~0,5 s)** ; le wem COMPLET est dans le `.pck`.

## 3. Axe 1 — banque et evenement de moteur, distincts par vehicule (structure)

Banque par vehicule (mappee par intersection des wems du pck, `donnees/lot1_vehbanks.json` ;
toutes presentes dans `pc/globals`) :

| Vehicule | pck | banque `sbnk` | evenement de moteur | architecture |
|---|---|---|---|---|
| Ghost | `sb_010_veh_cv_ghost` | `01862ab3` | `47361baf` | 3 couches SIMULTANEES : lit `192653757` + (1 parmi 4) + `835658180` ; PAS de switch (antigrav = continu, module en hauteur) |
| Scorpion | `sb_010_veh_un_scorpion` | `05a51e0a` | `951f76c0` (+ `0134da4e` allumage, `13beda14`/`d5c7daeb` boucles) | base (1 parmi 6) + son partage `195277626` + **SWITCH de regime a 4 etats** (noeud `196e9bed`, groupe `2275666646`) |
| Warthog (rockethog) | `sb_010_veh_un_rockethog` | `a52af042` | `68b1a949` (+ `38b83eb8`) | base (1 parmi 6) + son partage `195277626` + **SWITCH de regime a 4 etats** (noeud `3c24a4f9`, meme groupe `2275666646`) |

Source : `<scratch>/v3b/arbre_veh.json` (mode `eqip-arbre` sur les 3 banques).

**Le SWITCH existe** (contrairement a la conclusion rev7). Dans `951f76c0` (Scorpion) et
`68b1a949` (Warthog), un conteneur **Switch** (groupe `2275666646`, defaut `356702912`)
selectionne, **selon l'etat de vitesse**, des ENSEMBLES DE WEMS DIFFERENTS (6 wems par etat, 4
etats). Ce n'est donc pas « seulement le RTPC qui repitch une boucle unique » : le jeu **change
de samples** par bande de regime, en plus du pitch. Le Ghost, lui, n'a pas ce switch — l'antigrav
est un souffle continu module en hauteur, ce qui est coherent avec sa physique.

**Les trois evenements referencent des wems DISJOINTS.** Seuls deux elements sont mutualises :
`195277626` (un sous-lit d'ingenierie partage Scorpion+Warthog) et le lit commun
`06ba1096 -> e793c135` (present sur les 13 `vehi`). Tout le reste — les boucles qui portent le
timbre — est propre a chaque banque.

## 4. Axe 2 — le pck de chaque vehicule contient un materiau de mouvement de STRUCTURE differente

Extraction directe des pcks (`akpk_unpack.py`), mesure de duree + tenue (RMS par fenetre,
ffmpeg) :

- **Ghost** (`wem_ghost`, 248 wems) : boucles de moteur = couches SOUTENUES de 1,9 a 4,1 s,
  mono, tenue plate (souffle continu). Pas de longue boucle stereo.
- **Scorpion** (`wem_scorpion`, 84 wems) : boucles de chenilles STEADY de 2,92 s (loudest du
  pck), + un etage de 6 wems longs (8-9 s) par etat de switch. Les tres longs wems (5-9,5 s
  decroissants) sont le CANON, pas le moteur (piege « le plus gros wem »).
- **Warthog** (`wem_rockethog`, 78 wems) : boucles de combustion ~1,15-2,4 s ; les wems de
  4-4,8 s decroissants sont les ROQUETTES.

Structure du materiau de mouvement franchement differente d'un vehicule a l'autre : granulaire
soutenu (Ghost) vs boucle chenilles + etages de regime (Scorpion) vs boucles de rev combustion
(Warthog).

## 5. Axe 3 — timbres spectraux distincts, conformes a la verite terrain

Spectrogrammes (log-freq) + centroïde spectral moyen (ffmpeg `aspectralstats`) sur les wems du
moteur AUTHENTIFIES par la structure (§3) :

| Vehicule | wems mesures | centroïde | signature spectrale | verite terrain |
|---|---|---|---|---|
| Ghost | `192653757` (lit), `68830349`, `835658180` | 1796 / 751 / 1364 Hz | lit large et lisse (souffle) + basse pulsee (thrum antigrav) | souffle continu aerien + thrum — **CONFORME** |
| Scorpion | `1033065922`, `761195155` | 4687 / 2106 Hz | dense, large bande, striations verticales serrees = impacts de maillons | grondement grave + cliquetis de CHENILLES — **CONFORME** |
| Warthog | `351128378`, `955693063` | 1472 / 1698 Hz | dominante basse 0-6 kHz, cycle de rev | moteur de jeep a combustion — **CONFORME** |
| (temoin) lit commun `e793c135` | boucle 10 s | 2822 Hz | rush large-bande dense, present sur les 13 | lit de presence, PAS un moteur distinctif |

Les trois moteurs sont mutuellement distincts (timbre ET structure) et chacun colle a ce que
l'utilisateur entend. Le lit commun est plus brillant/large que n'importe lequel des trois et
sert de fond commun — il n'oppose pas Ghost/Scorpion/Warthog, ce qui est precisement pourquoi
rev7, en ne regardant que lui, ne voyait « aucune difference ».

## 6. Ce qui est livre

`sons_v3_reconstruits/<Vehicule>/deplacement/` pour **Ghost**, **Scorpion**,
**Warthog_roquettes** (les 3 a verite terrain), regime median = etat par defaut de l'evenement
de moteur, couches simultanees sommees (gains de chemin de la structure, anti-clip) :

- `moteur_boucle_regime_median.wav` — apercu bouclable ~8 s (corps repete).
- `moteur_corps_brut.wav` — le corps de boucle brut (le vrai materiau).
- `moteur_amorce.wav` / `moteur_queue.wav` — attaque / extinction (enveloppe).
- Scorpion : `moteur_amorce_allumage.wav` — vraie amorce d'allumage (evenement `0134da4e`).

Chaine de reconstruction, par vehicule :

- **Ghost** : `vehi -> banque 01862ab3 -> event 47361baf` = `192653757` + `68830349` + `835658180` (sommes).
- **Scorpion** : `vehi/weap 00015cfa -> banque 05a51e0a -> events d5c7daeb+13beda14` = `1033065922` + `369482775` ; regime : switch `196e9bed`.
- **Warthog** : `vehi/weap c7d50912 -> banque a52af042 -> event 38b83eb8` = `128563311` + `351128378` + `79651516` ; regime : switch `3c24a4f9`.

## 7. Ce qui reste vrai de rev7 (mais mal cadre)

- `sadt` refute comme moteur : VRAI (inchange).
- `06ba1096 -> e793c135` partage par les 13 : VRAI — mais c'est le **lit commun**, pas le
  moteur. rev7 l'a pris pour le moteur ; c'est le fond de presence.
- Mecanisme ECS (`managed-object-looping-sound-component` + RTPC) : VRAI — mais le composant
  poste l'evenement de moteur **propre au vehicule** (`47361baf`/`951f76c0`/`68b1a949`), pas
  seulement `06ba1096`, et un **switch de regime** selectionne des samples differents.

## 8. Suite possible (hors verite terrain, meme methode)

Les 6 autres vehicules ont deja leur evenement de moteur identifie (`donnees/lot2_moteur.json` :
Banshee `eb2e6227`, Wraith `6766338f`, Chopper `7d854715`, Gungoose `e3082ea3`, Falcon LMG
`01de6d3a`, Wasp `5baca8ee`). Meme recette : banque du vehicule -> `eqip-arbre` -> event de
moteur -> wems complets du pck. Le boost antigrav (Ghost/Wraith/Banshee) est un evenement
separe, non trace ici.
