# V3C — La boucle de deplacement corrigee : l'etat EN CONDUITE, couches superposees

> Worktree `LevelUp-wt-vehicules`. Lecture seule sur les donnees du jeu, aucun commit,
> aucune ecriture DuckDB, aucun nouveau Python (PowerShell + ffmpeg + vgmstream). AUCUN
> chargement du module 7,24 Go : toute la structure etait deja dans `<scratch>/v3b/arbre_veh.json`
> (mode `eqip-arbre`, rev8) et les wems complets dans `<scratch>/v3b/wem_{scorpion,ghost,rockethog}`.

## 1. Le probleme (retour utilisateur)

« Le seul que je reconnais c'est l'ALLUMAGE du Scorpion. Mais c'est incomplet, y a pas le
bruit des CHENILLES ni le DIESEL. » Les boucles de deplacement livrees par rev8 etaient
maigres. Cause trouvee sur pieces :

- **rev8 rendait le RALENTI, pas la conduite.** L'event moteur `951f76c0`/`68b1a949` porte un
  SWITCH de regime a **5 etats** (defaut + 4 conditionnels, 6 wems/etat). rev8 rendait l'etat
  par DEFAUT (`356702912`) = le ralenti, mesure comme le plus faible et le plus « spiky ».
- **Pire pour le Scorpion : rev8 n'utilisait meme pas l'event moteur.** Sa recette
  (`render_moteurs.ps1`) rendait `d5c7daeb`+`13beda14` — deux boucles de CHENILLES seules,
  SANS aucun corps diesel. D'ou « pas de diesel ». Et deux couches de chenilles mixees a demi
  chacune, sans grave sous elles, ne se lisent pas comme des chenilles non plus.

## 2. Methode — le spectre tranche l'etat EN CONDUITE

Pour chaque etat du switch et chaque couche candidate, mesure sur le wem COMPLET (pck),
avec ffmpeg `astats` : niveau RMS global, RMS bande basse (`lowpass=400`, = corps diesel),
RMS bande 3-6 kHz (`highpass=3000,lowpass=6000`, = chenilles/mecanique), et **crest factor**
(pic/RMS : bas = grondement continu, haut = pulsations/chugs distincts).

**Les 5 etats forment une echelle de regime monotone** (Scorpion, 2 wems/etat) :

| etat switch (groupe 2275666646) | RMS global | crest | duree | lecture |
|---|---|---|---|---|
| `356702912` (DEFAUT) | -20,6 / -21,1 | 10,8 / 11,4 | 8,2 / 7,0 s | ralenti (faible, chugs distincts) |
| `1093928064` | -21,6 / -23,5 | 12,0 / 13,9 | 8,2 / 9,5 s | ralenti/bas |
| `1136871302` | -19,3 / -20,6 | 9,2 / 10,7 | 5,7 / 7,1 s | intermediaire |
| `3707760930` | **-17,7 / -18,1** | **7,7 / 8,4** | 6,2 / 6,3 s | **EN CONDUITE** (fort, soutenu, boucle longue) |
| `163696720` | -17,2 / -17,2 | 7,1 / 7,2 | 3,6 / 3,7 s | haut regime (le plus fort, mais boucles courtes) |

Plus le regime monte, plus c'est FORT et plus le crest BAISSE (les pulsations de combustion
se fondent en grondement continu) — comportement physique d'un moteur. Le haut 3-6 kHz reste
a ~-40 dB sur TOUS les etats : le moteur est un pur grave (diesel), il ne porte pas les
chenilles. **EN CONDUITE = `3707760930`** (bon compromis fort + soutenu + boucle de 6,2 s ;
`163696720` est legerement plus fort mais ne boucle que 3,6 s).

**Les chenilles sont un event distinct.** `d5c7daeb`/`13beda14` : brillants (3-6 kHz a
**-19 dB** contre -40 pour le moteur, centroide 4687 Hz), denses (crest 3,0), boucles ~3 s.
Ce sont bien les chenilles — mais posees SUR le diesel, pas seules.

## 3. Resultat par vehicule (couches superposees, sommees, niveaux calibres par mesure RMS)

Assemblage conforme a la semantique `arbre.go` : chaque couche = une variante (RandomSequence),
couches simultanees sommees, gains de chemin respectes ; l'equilibre INTER-event (moteur vs
chenilles vs boost, non fixe par la banque) est calibre par cible de RMS pour que le corps
domine et que la couche de caractere reste audible. Boucles de 8 s (les wems SONT des corps
de boucle Wwise, repetes ; fondus aux seuls bords).

### Scorpion (banque `05a51e0a`)
- **DIESEL** = `951f76c0` etat `3707760930`, wem `85078605` (grave continu), cible -14 dB.
- **CHENILLES** = `d5c7daeb`, wem `1033065922` (brillant dense), cible -19 dB.
- **bed** grave partage `195277626`, cible -25 dB.
- **Allumage CONSERVE** : `0134da4e` (son `649399036`) -> `moteur_amorce_allumage.wav`, intact.
- Verif mix : bas **-13,8 dB dominant** (diesel = fondation) ; le 3-6 kHz remonte de -35 dB
  (diesel seul) a **-27,7 dB** dans le mix = les chenilles sont bien la ; crest 4,3 (continu).

### Warthog a roquettes (banque `a52af042`)
- **Corps RPM** = `68b1a949` etat `163696720` (seul a se detacher : -17,6 dB, crest 7,6,
  +3 dB sur les autres), wem `675731033`, cible -15 dB.
- **Combustion soutenue** = `38b83eb8` L2 `351128378` (fort, -15 dB brut), cible -16 dB ;
  + detail L3 `585735189` (-21) + chug grave L1 `128563311` (-22) + bed `195277626` (-26).
- Verif mix : bas **-13,6 dB dominant**, crest 4,3 (combustion continue).

### Ghost (banque `01862ab3`)
- **Souffle antigrav** = `47361baf` (3 couches simultanees) : mid `68830349` (dominant, -14),
  bed `192653757` (-20), thrum tonal `835658180` (-20). Mono (les wems Ghost sont mono).
- **BOOST** = `2e7f2aa2`, wem `52024965` : l'event le PLUS FORT de la banque (gain de chemin
  **+20 dB** ; brut -16,7 dB, effectif +3,6 dB), grave soutenu. Livre isole et en mix.
- Verif : souffle bas -14,5 dB, crest 4,1 ; +boost : +2,4 dB de niveau, bas -10,9 dB (poussee).

## 4. Livrables (ecrasent les boucles fausses)

Dans `sons_v3_reconstruits/<veh>/deplacement/` — les 4 fichiers rev8 faux
(`moteur_boucle_regime_median`, `moteur_corps_brut`, `moteur_amorce`, `moteur_queue`)
sont SUPPRIMES.

- **Scorpion/** : `moteur_conduite_boucle8s.wav` (diesel+chenilles+bed) ; `diesel.wav` ;
  `chenilles.wav` ; `moteur_amorce_allumage.wav` (conserve, valide).
- **Warthog_roquettes/** : `moteur_conduite_boucle8s.wav` (corps RPM + combustion) ;
  `diesel.wav` (corps RPM + combustion).
- **Ghost/** : `moteur_conduite_boucle8s.wav` (souffle) ;
  `moteur_conduite_boost_boucle8s.wav` (souffle + boost) ; `boost.wav`.

Manifeste `manifeste_v3.json` -> **rev9** (champ `switch_regime_5_etats` + `deplacement_par_vehicule`
par vehicule : etat retenu, couches, mesures, fichiers).

## 5. CR honnete — limites et incertitudes

- **Etat « en conduite » Scorpion** : `3707760930` retenu par le spectre (fort + soutenu +
  boucle longue). C'est un regime haut-mais-pas-redline ; `163696720` (plus fort, boucle 3,6 s)
  est une alternative valable si on prefere un regime plein. Choix defendable, non unique.
- **Ghost BOOST** : `2e7f2aa2` designe par PREUVE ACOUSTIQUE (event le plus fort de la banque,
  gain de chemin +20 dB, grave soutenu) — PAS confirme en jeu (pas d'input de boost testable
  hors ligne). C'est la meilleure hypothese sur pieces, a valider a l'oreille.
- **Warthog surface/roues** : AUCUNE couche de roues distincte trouvee dans la banque. Le haut
  3-6 kHz de l'etat de conduite (-31 dB, plus que les -40 du Scorpion) porte deja le mecanique.
  Rien ne manque « vraiment » — il n'y a pas d'event de roues separe a rendre.
- **Noms des etats de switch** : le groupe `2275666646` et ses etats sont des hachages FNV-1 ;
  un balayage de noms courants (idle/low/med/high/drive/rpm...) n'a rien matche. Les etats
  restent identifies par leur hachage + leur place dans l'echelle spectrale, pas par un nom.
- **Boucles** : 8 s avec fondus de bord pour audition propre — pas des boucles gapless de
  production (etape triviale si le timbre est valide). Les wems sous-jacents sont des corps de
  boucle Wwise.
