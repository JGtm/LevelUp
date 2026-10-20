# PROTOCOLE D'EXECUTION — Lot RESOLUTION VIP + ASSAUT (v7.5)

> Executeur SEUL, branche `wt/resolution`, base `f350eb01d` (tete du lot C : sites d'Assaut
> candidats figes, corpus Oddball 5). Contrat `plan-execution`. Ce protocole est COMMITE
> AVANT toute mesure (commit `resolution(protocole):`). Les seuils ci-dessous sont RECOPIES
> des plans (`PLAN_RESOLUTION_MODES.md` R2/R3, `PROTOCOLE_REMESURE_ODDBALL_VIP.md` §3) et ne
> se rebaissent JAMAIS apres coup. Un seul build Go a la fois. Decodage de film sous
> `internal/filmproc` (un film = un processus, plafond 2 Gio). DuckDB en LECTURE via
> `OpenReadForQuery` / payloads bruts figes. thought_log et REGISTRE_REPORTS : textes au CR,
> le superviseur consigne. R1 Oddball est DIFFERE (hors de ce lot).

## Perimetre

- **R2 — VIP** : nommer le composant statborg qui replique `TimesSelectedAsVip` par joueur,
  puis (si le gate tient) publier le marqueur VIP (couronne) par periode.
- **R3 — ASSAUT** : entrer les sites candidats d'armement au catalogue, rejouer le gate A1
  d'identite de la bombe (inchange), puis (si le gate tient) publier les vies libres de la
  bombe et documenter le poseur (A4, deja acquis).

Ce qui rate son gate reste `[!]` CHIFFRE. Un gate rate est un resultat valide.

---

## R2 — VIP : corpus, oracle, gates (FIGES)

### Corpus VIP — FIGE (rien ne s'ajoute apres ce commit)

Trois films VIP en cache (`data/cache/film_chunks/{...}`), mode `GameVariantCategory=23`,
8 joueurs `PlayerType=1`, 2 equipes, une seule playlist
`AssetId=1b1691dc-d8b9-4b1f-825d-cb1c065184c1` (verifie sur les payloads bruts).

| film (id court) | match_id | carte | map_id | bornes quant. |
|---|---|---|---|---|
| `00761d27` | 00761d27-487c-4d7d-ac4c-bf7584de652c | Bazaar   | 3e1e4cec-4f2c-44c6-b8d2-96b85c66c702 | PRESENTES |
| `9903b1c5` | 9903b1c5-b5e5-40d2-bd81-83d540b233cf | Bazaar   | 3e1e4cec-4f2c-44c6-b8d2-96b85c66c702 | PRESENTES |
| `99553e4a` | 99553e4a-905a-42b8-a2af-e8e5073fcb92 | Catalyst | f7e8cde9-0c0a-487c-94a3-61bfa0f20465 | PRESENTES |

Les deux cartes (Bazaar, Catalyst) sont au catalogue `map_quant_bounds.json` — la
qualification par bornes est donc franchissable ; la precondition de pont bipede >= 50 %
(`d8PontMinimum`) reste a MESURER (phase V0, instrument dedie).

### Oracle VIP — FIGE (`registre_film/V_oracle_vipstats.json`)

Releve des 7 colonnes `VipStats` par xuid (decimal, la forme du pont statborg), depuis les
payloads bruts `GetMatchStats` VERIFIES. Validation de la lecture : la somme par-joueur de
`TimesSelectedAsVip` egale l'agregat d'equipe `Teams[].VipStats` sur les 3 films, par equipe
(24/24 lignes) — la lecture est saine.

`TimesSelectedAsVip` par film (multiset, ordre indifferent) et total-film :
- `00761d27` (Bazaar)   : {4,3,2,1,2,1,1,1}, total = 15 (equipe 0 = 10, equipe 1 = 5)
- `9903b1c5` (Bazaar)   : {3,2,3,2,2,1,2,2}, total = 17 (equipe 0 = 10, equipe 1 = 7)
- `99553e4a` (Catalyst) : {2,2,2,2,2,3,2,3}, total = 18 (equipe 0 = 8,  equipe 1 = 10)

**CORRECTION DE FAIT.** Les arrays illustratifs du protocole source (`PROTOCOLE_REMESURE
_ODDBALL_VIP.md` §3.2 : `[2,2,1,2,2,5,2,0]`, `[7,1,1,1,4,1,0,2]`, `[4,2,2,1,1,5,0,0]`)
NE correspondent PAS aux payloads bruts (multisets differents, totaux 16/17/15 contre
15/17/18). Le payload brut fait foi (confirme par l'agregat d'equipe) ; les valeurs
ci-dessus sont l'oracle FIGE. La premisse « la variation par-joueur fait la force de
l'empreinte » reste valable : `TimesSelectedAsVip` varie 1..4, decouple de la duree.

### Gate VIP (RECOPIE de §3.5/§3.6, NON NEGOCIABLE)

Instrument : `cmd/statnames-sweep -films 00761d27,9903b1c5,99553e4a` (pont
`SlotIdentityByDeaths`, meme TSV qu'A4), puis confrontation VIP dediee (meme package,
reutilise `loadSweep`/`loadOracle`/`encode`).

- **Test par-joueur** (`TimesSelectedAsVip`, encodage entier `[n]`) : pour chaque
  emplacement (comp 0..27 x cote A/B), accord par-joueur (egalite entiere) par film.
  - CANDIDAT : le comp au MEILLEUR accord moyen sur les 3 films.
  - **GATE VERDICT** : le comp replique `TimesSelectedAsVip` si accord >= **90 %** sur
    >= **2 des 3 films**, avec >= **3 paires non nulles** par film compte.
  - **STABILITE** (garde-fou corpus mince) : le comp retenu doit etre le MEILLEUR sur
    **3/3** films. Meilleur sur 2 mais supplante sur le 3e = REJETE.
  - **TEMOIN** : permutation cyclique de l'affectation xuid -> oracle dans CHAQUE film
    (attribution aleatoire des comps). Exigence : accord <= **20 %** sur CHACUN des 3 films.
- **Test somme-film** (confirmation, immune au pont) : `S(film) = somme sur slots JOUEURS de
  la valeur finale du comp` ; `O(film) = total-film de TimesSelectedAsVip`. CANDIDAT si
  `S == O` sur >= 2/3 films (`O >= 1` requis) ; temoin decale (re-appariement cyclique
  film -> total), **0 faux candidat** exige. Desaccord test par-joueur (positif) vs
  somme-film (negatif) = pont VIP trop etroit, dit au log.
- **Cibles secondaires** (`KillsAsVip`, `VipKills`) : memes seuils PLUS anti-aliasing — le
  comp candidat doit etre DISTINCT de comp 2 A / 3 A et sa valeur par slot <= la valeur du
  comp generique correspondant (un sous-compte ne depasse pas son total). Hors gate
  principal ; publiees pour completude.

Log fige : `registre_film/V_statborg_vip.log` (qualification V0, `slots_nommes` par film,
accord par comp/film, candidat, gate 2/3, stabilite 3/3, temoin permute, somme-film, verdict
nomme chiffre).

### Publication VIP (SI et SEULEMENT SI le gate tient)

Les PERIODES VIP = entre deux selections, bornees par les morts du VIP (kill feed deja
decode). Publier le MARQUEUR VIP (couronne) sur le joueur VIP courant par periode. GATE de
periode : coherence des periodes reconstituees vs `TimeAsVip` API par joueur >= **90 %**.
Triplet schema (Go/contrat/`EXPECTED_REPLAY_SCHEMA_VERSION`) + chronique, i18n FR+EN, calque
au patron existant, re-cuisson temoins avec verification de CONTENU. Sinon `[!]` chiffre.

### Corroboration kill feed (DIAGNOSTIC, non gating)

Histogramme des ecarts entre chaque increment du comp candidat et la mort la plus proche du
kill feed ; accord = part a <= 1000 ms (`d4EcartEvenementMS`, deja commite). Temoin a deux
classes : morts du VIP vs instants aleatoires. Verdict « les selections suivent les morts »
seulement si accord morts >= 80 % ET nettement > la classe aleatoire. N'elit rien.

---

## R3 — ASSAUT : entree des sites, gate A1 (INCHANGE), publication

### Sites candidats — FIGES (`registre_film/C2_sites_candidats.json`, lot C)

Les 5 cartes du corpus Assaut ne portent aucun objet au role historique `assault_bomb`
(hash -534119345, resolu mais absent de ces cartes). Le motif de site est porte par deux
hashs de label NON RESOLUS (chasse murmur3 a 2173 candidats sans nom, patron KOTH) :
`-1843278509` = positions de BASE par equipe, `-1537427652` = position CENTRALE neutre.
Motif stable sur les 5 cartes (origin, absolution, rat's nest, curfew, urban raid). Les
`.mvar` terrain (`*_map.mvar`) de ces 5 cartes sont disponibles (re_dump du depot).

### La chaine du lot C (entree au catalogue)

1. **mapvar** : attribuer `RoleAssaultBomb` par HASH pour `-1843278509` / `-1537427652`,
   EXACTEMENT le patron KOTH (`LabelHashHillRole`, role pose par hash quand le nom snake_case
   reste introuvable). Le hash est le seul critere ; aucune heuristique de type_id ou de
   dimensions.
2. **mapobj-build** `--from-file <carte>_map.mvar --map-id <uuid>` pour les 5 cartes
   (mergeExisting preserve le reste du catalogue) — regenere les marqueurs `assault_bomb` de
   ces cartes avec la classification courante. Schema du catalogue inchange (v2).
3. Verification : `attMarqueurs(..., "assault_bomb")` rend desormais N sites ponctuels par
   carte du corpus.

### ARBITRAGE (lot C) — pourquoi ce n'est PAS circulaire

Les sites ont ete choisis spatialement (25/25 explosions avec activite au site), mais le
temoin spatial 12 m est INSATURABLE (deux disques 10 m / 12 m se recouvrent). L'identite de
la bombe se valide donc par le gate A1 LUI-MEME, qui ajoute deux jambes INDEPENDANTES du
choix spatial : (a) jambe TEMPORELLE — la creation `ti=42` doit coincider avec un debut de
manche (<= 5 s) ou une remise post-explosion (<= 15 s) ; (b) SELECTIVITE INTERNE — parmi les
mots `ti=42` que le catalogue d'armes ECARTE, UN SEUL mot de 32 bits doit reunir site ET
coincidence, sur >= 2 films, quand le temoin (tous les autres mots ecartes) reste a 0. Un mot
constant a travers les films (la bombe est le meme objet sur toutes les cartes) qui satisfait
les deux jambes est un signal, pas une consequence du choix des sites.

### Gate A1.3 (ORIGINEL, INCHANGE)

`assaut_a1_identite_test.go` rejoue TEL QUEL (seuils inchanges : `a1DebutMancheMS=5000`,
`a1RemiseMaxMS=15000`, `attDrapeauRayonM=3.0`), un film par processus, sur le corpus Assaut
qualifie. GATE : UN mot elu (jambe site ET jambe coincidence, `lesDeux > 0`), sur >= **2
films**, TEMOIN = 0 (aucun autre mot ecarte ne reunit les deux jambes). Candidats connus
publies : `0x3FEE4FCF` (7/7 films au balayage precedent), `0xE9E7FF79` (4). Log fige :
`registre_film/R3_a1_rejeu.log`.

### Publication ASSAUT (SI et SEULEMENT SI A1.3 tient)

VIES LIBRES de la bombe : famille `bomb` au manifeste (EN+FR), garde d'exclusion des socles,
calque `objectiveObjects`, patron du crane libre (`flag_objects.go`). Documenter/publier le
POSEUR (A4 : comp 0 A des slots joueurs = explosions par joueur). Triplet schema + chronique,
i18n, re-cuisson temoins avec verification de CONTENU. Sinon `[!]` chiffre.

---

## GATES DU LOT & garde-fous communs

- Protocole commite avant mesure (historique git en temoigne) ; seuils geles ; logs figes.
- Temoin obligatoire dans CHAQUE test (permute/decale) — un test dont le temoin ne s'effondre
  pas n'a rien mesure.
- Si PUBLICATION : `go test` des packages touches + contracttest ; `tsc -b` (cache purge) ;
  vitest match-replay ; lint web ; parite schema web/Go. `go vet` + `go build` exit 0. Arbre
  propre, aucun push.
- Un seul build Go a la fois ; decouvertes hors perimetre NOTEES, jamais traitees.

## Sorties attendues (logs figes)

- `V_oracle_vipstats.json` (committe avec ce protocole).
- `V_statborg_vip.log` (mesure R2).
- `R3_a1_rejeu.log` (mesure R3).
