# PLAN DE RESOLUTION — Oddball, VIP, Assaut (+ Stockpile/Extraction en bonus)

> Lot Go UNIQUE, executeur SEUL (un seul build Go a la fois — pas de workflow qui fan-out des
> builds). Base = tete de feat/v75 APRES fusion du lot C (catalogues : bornes Live Fire,
> sites assault_bomb, oddball_spawn Lattice). Branche `wt/resolution-modes`. Contrat
> plan-execution. Protocole COMMITE avant toute mesure ; seuils ci-dessous JAMAIS abaisses ;
> filmproc obligatoire ; DuckDB lecture seule ; thought_log/registre par le superviseur.
> Le protocole pre-enregistre statborg (scratchpad `PROTOCOLE_REMESURE_ODDBALL_VIP.md`,
> 28 Ko) fait foi pour VIP et la re-mesure statborg Oddball ; CE plan ajoute la VOIE DRAPEAU
> Oddball (primaire) et l'ordre.

## Ordre : R1 Oddball -> R2 VIP -> R3 Assaut -> R4 (bonus) Stockpile/Extraction.

---

## R1 — ODDBALL PORTEUR : DIFFERE (toutes les voies offline epuisees au 27/08)

**Statut : [!] corpus/ground-truth.** Bilan des voies : (1) keyframe ti=11 = etat par defaut
(i3 porteur null) ; (2) delta ti=11 = bruit (chaine 3,8 %, champs indistinguables du fantome) —
managed-objective est un descripteur HUD calcule cote client ; (3) statborg skull_grabs =
sous-puissant (D10 O4 : 4 paires non-nulles sur 4-5 films, grab rare) ; (4) proximite/traversee
= 5 campagnes ratees (D4-D9). La voie du DRAPEAU (nommer skull_grabs + transposer flag_carries)
reste VIABLE EN PRINCIPE mais bloquee par le CORPUS : il faut assez d'evenements de prise pour
nommer le slot, or les grabs sont rares sur 5 films. **Reprise = (a) l'utilisateur REJOUE quelques
Oddball (le sync capture les films ; plus de grabs -> slot nommable -> flag_carries), ou (b) verite
terrain Theater datee (Cheat Engine, jeu lance). Le calque du crane LIBRE reste le livrable Oddball.**
NE PAS relancer keyframe/delta ti=11 ni une 6e campagne proximite.

### (ANCIEN R1 — voie drapeau, conservee pour la reprise quand le corpus grandit)

**Fait etabli** : le porteur du DRAPEAU = acteur de l'evenement `flag_grabs` du statborg
(`named.go:91` : `{comp 22, sideA} -> StatFlagGrabs`), ponte slot->xuid, transpose par
`flag_carries.go` (prise ouvre le portage ; ferme par capture/mort/nouvelle prise/fin).
`namedStatSlots` n'a PAS de table Oddball (`named.go:160` : « ball pas encore nomme, le
balayage est le meme, c'est le corpus qui manque »). Les 5 campagnes ont tente l'inference,
JAMAIS la voie native de la prise.

- [ ] R1.1 NOMMER le slot `skull_grabs`. Confronter chaque emplacement statborg (comp x side)
      a l'oracle `SkullGrabs` par joueur (`match_objective_stats_latest`) sur MOITIES
      DISJOINTES (recherche 24dbb67d+92f18088 / verification 43716616+d9781168), MEME methode
      que le drapeau. Le compte etant sous-puissant (grabs rares, ~4 paires non-nulles),
      AJOUTER un critere de TIMING : les emissions du slot candidat doivent s'aligner (tol.
      1000 ms) sur les debuts de `th=10` de possession ET les naissances de vies libres du
      crane. Un slot qui accorde EN COMPTE (moities) ET EN TIMING est le candidat.
      GATE R1.1 (ecrit) : UN slot, accord de comptes >= 80 % sur la moitie verification (ou,
      si sous-puissant, >= 90 % d'alignement de timing sur >= 3/4 films avec temoin slot-decale
      <= 30 %). Log `R1_skull_grabs_nommage.log`.
- [ ] R1.2 SI slot nomme : ajouter l'entree `ObjectiveTypeSkull` a `namedStatSlots`
      (garde-rail : meme discipline « moities disjointes » que le commentaire named.go:80).
- [ ] R1.3 TRANSPOSER `flag_carries.go` au crane (`skull_carries.go`) : prise -> porteur
      ponte ; FERMETURE par mort (`skull_carriers_killed` + fil des morts), nouvelle prise,
      fin de match (pas de « capture a la base » en Oddball ; le score est continu).
      Attribution du crane : un seul crane (pas de socle par equipe) — plus simple que le
      drapeau. GATE R1.3 (= le gate historique, INCHANGE) : recouvrement
      `time_as_skull_carrier_seconds` >= 80 % par joueur ET porteur principal sur >= 3/4 films.
- [ ] R1.4 SI gate tenu : publier le PORTEUR dans `objectiveObjects` (cle porteur, contrat
      +1 ; remplacer PROPREMENT les 2 refus testes : rien pendant portage, pas de prolongation
      apres t1) ; rendu crane-sur-porteur au patron `flagCarriesLayer`. SINON : `[!]` chiffre,
      la voie drapeau est mesuree, on passe a R1.5.
- [ ] R1.5 (SECONDAIRE, seulement si R1.3 rate) : re-mesure statborg par SOMME D'EQUIPE (A4,
      immune au pont) + 6e colonne `kills_as_skull_carrier` — cf. protocole pre-enregistre.

**Corpus** : 4 films (6 si lot C a ajoute Live Fire). Levier si sous-puissant : user rejoue
Oddball. Ne PAS abaisser le gate ; documenter le manque de puissance s'il bloque.

---

## R2 — VIP : qui a ete VIP, nativement (statborg `TimesSelectedAsVip`)

**Fait etabli** : le film ne porte PAS le bit VIP (sans serialiseur, §2.10). MAIS
`VipStats.TimesSelectedAsVip` est un ENTIER DISCRET additif (somme joueurs = agregat equipe),
signature 0..N UNIQUE au VIP (zero aliasing avec kills/assists) — le meilleur candidat de
compteur repliquable (patron A4 : les evenements discrets repliquent, les durees non).

- [x] R2.0 Qualifier les 3 films VIP (`00761d27`, `9903b1c5`, `99553e4a`) : bornes + pont
      >= 50 %. FAIT : Bazaar(x2)+Catalyst, bornes presentes, pont bipede 93.1/90.3/95.6 %,
      3/3 ADMIS. Instrument `vip_v0_qualification_test.go`. Corpus fige.
- [x] R2.1 Balayer le statborg (cmd/statnames-sweep) contre `TimesSelectedAsVip` par joueur.
      FAIT : 8/8 slots nommes/film (24 paires). Confront VIP dedie (statnames-sweep -vip).
      RESULTAT : comp **22 A** reproduit `TimesSelectedAsVip` EXACTEMENT par joueur (100 % x3)
      ET somme-film exact (15/17/18, temoin decale 0). GATE R2.1 **NON TENU** : le temoin
      permute (<= 20 %/film) echoue (12.5/25.0/50.0 %) car compteur a FAIBLE variance (six
      joueurs a 2 sur 8) → self-similarite ~sum(p_v^2) ~50-62 %, le temoin ne peut pas
      s'effondrer. Seuil NON abaisse, temoin NON change. Log `V_statborg_vip.log`. `[!]` chiffre.
- [!] R2.2 SI nomme : couronne par periode. NON FAIT — gate R2.1 non tenu (clause temoin),
      pas de publication. Le signal (comp 22 A) est fort ; condition de reprise = temoin
      adapte aux compteurs a faible variance, PRE-ENREGISTRE avant re-mesure.
- [x] R2.3 Corpus mince (3 films) : reserve de robustesse ECRITE au protocole (§0.2 repris) ;
      3 garde-fous (2/3 + stabilite 3/3 + somme-film) a la place du split. Applique.

---

## R3 — ASSAUT : DEUX composants — la bombe PORTEE (objet) ET le site d'armement (ZONE)

**Precision user** : l'Assaut a des ZONES. Le site d'armement est une zone qu'on DEFEND
(patron Bastion : etat contested/arme/desamorce). Donc Assaut = un OBJET porte (la bombe) +
une ZONE (le site). Depend de lot C (sites `assault_bomb` des cartes du corpus au catalogue).

- [x] R3.1 Rejouer `assaut_a1_identite_test.go` (seuils inchanges) avec les sites CANDIDATS C2
      EN ENTREE (env A1_SITES, override scoped — les hashs candidats sont des navpoints
      GENERIQUES base/centre, mesures identiques sur Catalyst NON-Assaut : role global exclu).
      GATE A1.3 originel (UN mot, >= 2 films, temoin 0) : **TENU**. `0x3FEE4FCF` elu sur 6/8
      films exploitables, temoin 0. Log `R3_a1_rejeu.log`. RESERVES au log (aliasing sites,
      compte 13-62/film incompatible avec UNE bombe, echec sur les 2 films au plus fort compte).
- [!] R3.2 SI tenu : publier les VIES LIBRES. NON FAIT — la fraction nee-au-site de 0x3FEE4FCF
      (8-69 %/film, via A1) est loin du controle crane-libre (>= 90 %) ; comme le drapeau libre
      (75.6 %, non publie), les vies libres ne se publient pas. Reserves d'identite non levees.
      Condition de reprise = confirmer l'identite par un controle temporel independant + lever
      l'aliasing des sites, avant tout calque `objectiveObjects`.
- [!] R3.3 SITE D'ARMEMENT = ZONE (canal ti=13 tag 4). NON TRAITE — hors de la voie STATBORG de
      ce lot (mandat = bombe/poseur par statborg). Le ti=13 est documente illisible au lot A
      (contamination arene, A3) ; de plus les sites candidats sont des POINTS sans forme (pas de
      FORME de zone a mesurer). Reste au chantier decodage ti=13.
- [x] R3.4 POSEUR de bombe (A4, deja acquis) : DOCUMENTE. Comp 0 A des slots joueurs =
      explosions par joueur (`A4_statborg_assaut.log` : comp 0 A REPLIQUE, 4/4 recherche +
      4/4 verification). PORTEUR de bombe = voie drapeau (R1, DIFFERE) — non traite.

---

## R4 — Stockpile & Extraction (les deux confirmes par le user : Extraction = ZONES)

**Extraction = mode a ZONES** (points de conversion, precision user) — PAS une sonde a
l'aveugle : appliquer la machinerie `zoneStates`/Bastion aux points d'extraction.
- [!] R4.1 Extraction — HORS MANDAT de ce lot (executeur : R2 VIP + R3 Assaut uniquement,
      R1 differe). Non traite.
- [!] R4.2 Stockpile — HORS MANDAT de ce lot. Non traite.
- BONUS strict : ne pas retarder R1-R3.

---

## JOURNAL — cloture lot RESOLUTION (executeur VIP + ASSAUT), 2026-08-27

Contrat plan-execution respecte : protocole commite AVANT mesure (`2a4bac4b3`), seuils geles,
temoins obligatoires, un seul build Go a la fois, filmproc, DuckDB lecture seule (via payloads
bruts figes — un serveur tenait la shared DB RW).

- **R2 VIP** : comp **22 A** NOMME comme reproduisant `TimesSelectedAsVip` (signal 100 % x3
  par joueur + somme-film exact + temoin decale 0). GATE R2.1 NON TENU : la clause du temoin
  permute echoue par FAIBLE VARIANCE de la donnee (self-similarite ~50-62 %). Verdict `[!]`,
  pas de couronne. Commit `7c39d9e2c`. Log `V_statborg_vip.log`, oracle `V_oracle_vipstats.json`.
- **R3 ASSAUT** : gate A1.3 TENU (`0x3FEE4FCF`, 6 films, temoin 0) via sites C2 EN ENTREE.
  FINDING majeur : les sites candidats C2 sont des navpoints GENERIQUES (aliasing mesure sur
  Catalyst). Reserves fortes (compte eleve, echec sur 2 films). Publication vies libres NON
  faite (born-at-site 8-69 % << 90 %). Poseur A4 documente. Commit `b1adefc74`. Log
  `R3_a1_rejeu.log`.
- **Publication** : AUCUNE (les deux modes echouent leur gate de publication). Pas de bump de
  schema, pas de calque web, pas de re-cuisson temoins — sans objet.
- **Verdict global** : 0 seuil abaisse, 0 temoin change apres coup, 2 mesures rigoureuses, 2
  verdicts honnetes ; comp VIP nomme, identite bombe etablie sous reserves. Deux conditions de
  reprise ecrites (temoin faible-variance pour VIP ; lever l'aliasing + controle temporel
  independant pour la bombe).

---

## GATES DU LOT
Protocole commite avant mesure (un seul commit) ; logs figes ; si publication : go test
packages touches + contracttest, tsc -b (cache purge), vitest match-replay, lint web, parite
schema web/Go ; go vet + go build exit 0 ; arbre propre ; pas de push. CR : verdict chiffre
par mode, comptes de temoins re-cuits, commits, textes journal/registre, decouvertes.
