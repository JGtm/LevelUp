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

- [ ] R2.0 Qualifier les 3 films VIP (`00761d27`, `9903b1c5`, `99553e4a`) : bornes + pont
      >= 50 %. Corpus admis fige. Log `R2_qualif.log`.
- [ ] R2.1 Balayer le statborg (cmd/statnames-sweep) contre `TimesSelectedAsVip` par joueur,
      moities disjointes. Puis `KillsAsVip`/`VipKills`/`VipAssists` (discrets). DISQUALIFIER
      `TimeAsVip`/`LongestTimeAsVip` (durees) et `MaxKillingSpreeAsVip` (max non additif).
      GATE R2.1 : un comp replique `TimesSelectedAsVip` >= 90 % sur >= 2/3 films, temoin
      (comp aleatoire) <= 20 %. Log `R2_vip_selection.log`.
- [ ] R2.2 SI nomme : les PERIODES VIP = entre deux selections, bornees par les morts du VIP
      (kill feed deja decode). Publier le MARQUEUR VIP (couronne) sur le joueur VIP courant
      par periode. Contrat +1 (meme montee que R1.4 si groupees). GATE : cohérence des
      periodes reconstituees vs `TimeAsVip` API par joueur >= 90 %. SINON `[!]` chiffre.
- [ ] R2.3 Corpus mince (3 films) : reserve de robustesse ECRITE ; pas de split si < 3 admis.

---

## R3 — ASSAUT : DEUX composants — la bombe PORTEE (objet) ET le site d'armement (ZONE)

**Precision user** : l'Assaut a des ZONES. Le site d'armement est une zone qu'on DEFEND
(patron Bastion : etat contested/arme/desamorce). Donc Assaut = un OBJET porte (la bombe) +
une ZONE (le site). Depend de lot C (sites `assault_bomb` des cartes du corpus au catalogue).

- [ ] R3.1 Rejouer `assaut_a1_identite_test.go` TEL QUEL (seuils inchanges) sur le catalogue
      complete. Candidats connus : `0x3FEE4FCF` (7/7), `0xE9E7FF79` (4). GATE = A1.3 originel
      (UN mot, >= 2 films, temoin 0). Log `R3_a1_rejeu.log`.
- [ ] R3.2 SI tenu : publier les VIES LIBRES de la bombe (famille `bomb` au manifeste EN+FR,
      garde d'exclusion socles, calque `objectiveObjects`), patron du crane libre. Re-cuisson
      temoins avec verification de CONTENU.
- [ ] R3.3 SITE D'ARMEMENT = ZONE. Avec les sites au catalogue (lot C), REJOUER A3 (canal
      ti=13 tag 4, patron `zoneStates`/Bastion) MAIS cette fois avec les FORMES de site (le
      temoin spatial 12 m devient mesurable, ce qui manquait au lot A). Etat de la zone :
      neutre / en cours d'armement / armee / desamorcage. GATE (repris de A3.2) : accord
      canal <-> explosions datees >= 90 % sur >= 2 films, temoin decale <= 20 %. SI ti=13
      reste illisible (contamination arene documentee A3) : `[!]`, la zone attend le chantier
      decodage ti=13. Log `R3_site_zone.log`.
- [ ] R3.4 POSEUR de bombe (deja acquis A4 : comp 0 A = explosions par joueur) : publier
      l'attribution du poseur a chaque armement OU documenter l'acquis. PORTEUR de bombe = voie
      drapeau (R1) SI R1 reussit : transposer skull_carries au bomb-carry (bombe = objet porte).

---

## R4 — Stockpile & Extraction (les deux confirmes par le user : Extraction = ZONES)

**Extraction = mode a ZONES** (points de conversion, precision user) — PAS une sonde a
l'aveugle : appliquer la machinerie `zoneStates`/Bastion aux points d'extraction.
- [ ] R4.1 Extraction (2 films `41722c72`, `be565134`) : le role `extraction_zone` est deja
      servi statiquement. Chercher l'etat vivant par canal ti=13 tag 4 aux points (patron
      Bastion), croiser `ExtractionStats` (initiations/conversions par joueur) + score de
      mode. GATE : accord proprietaire/conversion >= 90 % sur les 2 films, temoin decale
      <= 20 %. SINON `[!]` avec reserve de corpus.
- [ ] R4.2 Stockpile (2 films) : le noyau/graine = objet PORTE (meme recette que skull/flag) —
      `StockpileStats` a `PowerSeedsDeposited`/`PowerSeedsStolen`/`TimeAsPowerSeedCarrier` ;
      batir la table `ObjectiveTypeSeed` (seed_grabs/deposits) comme R1, transposer
      flag_carries. Le point de DEPOT = marqueur statique (pas une zone tenue, precision user).
      Corpus mince (2 films) : verdict avec reserve OU `[!]`.
- BONUS strict : ne pas retarder R1-R3.

---

## GATES DU LOT
Protocole commite avant mesure (un seul commit) ; logs figes ; si publication : go test
packages touches + contracttest, tsc -b (cache purge), vitest match-replay, lint web, parite
schema web/Go ; go vet + go build exit 0 ; arbre propre ; pas de push. CR : verdict chiffre
par mode, comptes de temoins re-cuits, commits, textes journal/registre, decouvertes.
