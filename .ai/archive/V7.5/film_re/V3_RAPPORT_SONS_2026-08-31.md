# V3 — RAPPORT : sons des véhicules (chantier véhicules-tourelles)

> Worktree `LevelUp-wt-vehicules`. Aucun commit, aucune écriture DuckDB, aucun nouveau Python,
> Ghidra en LECTURE SEULE (curl HTTP). Un seul build Go (cache isolé), un module 7,24 Go à la fois.
>
> **Révision 7.** Mandat : trancher le SWITCH — un Wwise switch sélectionne-t-il un wem
> différent PAR TYPE de véhicule (auquel cas les moteurs distincts seraient dans les données) ?
> **Verdict : NON.** La banque du sound_looping partagé (`e793c135`) n'a AUCUN conteneur
> Switch ; c'est une boucle générique unique pour les 13, seul le RTPC de vitesse varie au
> runtime. Donc **le per-véhicule n'est pas dans les données statiques** (repli = capture en
> jeu). Rappel rev 6 : `sadt` réfuté comme moteur.

## 0. Ce qui est livré

- **TIR** (banques `_veh_` confirmées FNV-1) : Wasp, Scorpion, Gungoose, Warthog + Falcon
  (= mitrailleuse). Inchangé, accepté.
- **DÉPLACEMENT** : le **sound_looping de mouvement PARTAGÉ** rendu à régime médian, en
  start / loop / stop → `Mouvement_generique_partage/deplacement/`. **Un seul jeu, commun aux
  13 véhicules** (voir §3 pour la limite honnête).

## 1. `sadt` — RÉFUTÉ comme moteur (mandat principal)

- Le FourCC `sadt` (`0x73616474`, stocké "tdas" en BE) est manipulé par **`FUN_1404cc908`**,
  qui fait un simple **test de groupe** (`if (param_1 == 0x73616474)`) — **aucune structure
  audio, aucun event, aucun RTPC**. `sadt` n'est pas décodé comme un son ici.
- Décisif : le **Scorpion (char) et le Ghost (antigrav) référencent le MÊME `sadt 2b6f1d67`**.
  Un moteur par véhicule ne serait pas partagé entre un char et un antigrav. `sadt` n'est
  donc pas le porteur du moteur. La piste `sadt` est close.

## 2. Le vrai mécanisme (Ghidra) et le son trouvé

**Le déplacement est un `managed-object-looping-sound-component`** (composant ECS, str
`@143c94ac0`) posé sur l'objet-véhicule et **modulé en continu par un `managed-object-rtpc-
component`** (`@143c952c8`) via `AK::SoundEngine::SetRTPCValue` (`@14196fb90`). Wwise
`PostEvent @14196a700`.

**Où vit le son** : les 13 `vehi` référencent tous **le même** `lsnd 06ba1096`
(sound_looping) → banque **`e793c135`**. Sa structure interne (lue dans ses octets) porte les
event IDs `0xd492e38e / 0x558e8bcf / …` appariés au tag `sbnk e793c135`. Cette banque contient
**11 wems soutenus (0,98 s à 5,27 s)** — le matériau start / loop / stop du mouvement, rendu
ici à régime médian. C'est la **boucle de mouvement PAR DÉFAUT** demandée.

Livré dans `Mouvement_generique_partage/deplacement/` : 2 corps de boucle assemblés
(≈5,3–5,8 s), 2 corps de boucle bruts (4,71 / 5,27 s), 2 amorces/queues candidates (0,98 /
1,17 s), 1 aperçu bouclé 10 s.

## 3. LE SWITCH — tranché : PAS de sélection par type de véhicule

Hypothèse à tester : un Wwise switch choisit-il un wem différent selon le type de véhicule
(11 wems ≈ 13 véhicules) ? **Non, prouvé sur pièces :**

- **`e793c135` n'a AUCUN conteneur Switch (type 6).** L'arbre de la banque (mode `arbre`)
  montre **10 events, tous RandomSequence à 2 couches**, réutilisant les **mêmes 11 wems**
  (`681253652, 439608271, 1047396629, 166083965, 884547804, 121561261, 50887519`) simplement
  reshufflés. Aucune branche état→wem par type.
- **`SetSwitch @141970280`** (`AK::SoundEngine::SetSwitch(group, state, gameObject)`) n'a
  qu'un appelant réel : **`FUN_140d3d9e0`**, un wrapper générique `SetSwitch(pair.group,
  pair.value, obj)`. Aucun groupe de switch « type de véhicule » ne pilote la boucle moteur.

**Conclusion** : les 13 véhicules partagent **une seule boucle de mouvement** (`e793c135`), et
la différence Ghost (souffle) / Scorpion (chenilles) / Warthog (jeep) est produite **au runtime
par le seul RTPC de vitesse** (pitch/volume), **pas** par un choix de wem. Il n'existe donc
**pas** de « moteur du Ghost » vs « moteur du Scorpion » comme samples distincts dans les
données.

**Per-véhicule extractible : NON (prouvé).** La cible réduite est atteinte au niveau
**générique** (§2, le son livré). Pour des moteurs **distincts par véhicule**, le repli est la
**capture en jeu** (enregistrer chaque véhicule à vitesse moyenne) — c'est ma recommandation.

## 4. Boost (antigrav) — non tracé ce tour

Le boost des antigrav (Ghost/Wraith/Banshee) est un event séparé déclenché à l'accélération.
Il se trouverait par l'input de boost → `PostEvent`. Non tracé faute de budget ; piste nette
si l'utilisateur le veut.

## 5. Preuve Ghidra (HaloInfinite.exe, base `0x140000000`, HTTP lecture seule)

    AK::SoundEngine::PostEvent      @ 0x14196a700   (eventID u32, gameObject)
    AK::SoundEngine::SetRTPCValue   @ 0x14196fb90
    AK::SoundEngine::SetSwitch      @ 0x141970280
    managed-object-looping-sound-component   str @ 0x143c94ac0
    managed-object-rtpc-component            str @ 0x143c952c8
    sound_object_loop_set_rtpc -> impl FUN_1407b905c   (str @ 0x143c35570)
    sadt handler FUN_1404cc908  (cmp FourCC 0x73616474, aucun audio) -> sadt refute

## 6. TIRS — confirmés (rappel)

Wasp `e22a0d32` = 3 couches +38/+33/+31 dB sommés ; Gungoose +10/+10/+9 dB ; cadences
450/420/126 cpm. Falcon = mitrailleuse détachable (`bd807a77`).

## 7. Outillage (lecture seule, gofmt-clean, non commité)

`cmd/weapon-sounds/vehicules_sons.go` (mode `vehi-sons`). Pont Ghidra : curl HTTP GET
(decompile / xrefs / search / byte patterns).

## 8. Emplacements

- Livrables : `.ai/V7.5/film_re/sons_v3_reconstruits/` — TIR par véhicule + `Mouvement_
  generique_partage/deplacement/` + `manifeste_v3.json` (rev 6).
- Intermédiaires : `<scratch>\donnees\*.json`. `_outils` restauré.
