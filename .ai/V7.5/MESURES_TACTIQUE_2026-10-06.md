# Mesures — maquette « Tactique », avant / après (2026-10-06)

Base : COPIE `b915a0be…\scratchpad\db\shared_matches_v2.duckdb` (copiée le 2026-10-05 18:26), lue par
`diag_q.exe <db> "<sql>"` (TSV, lecture seule), une instruction par appel, vues `_latest` uniquement.
Joueur : JGtm `2533274823110022`. Périmètre : tous ses matchs, sans filtre. Fichiers bruts : `ex/`.
Aucun artefact de rejeu lu. Calages des fonds : `data/titles/halo_infinite/reference/map_backgrounds/*.json`
(copiés dans `ex/cal_*.json`, images dans `fonds/`).

## Q1 — Les 14 cartes les plus jouées (requête du brief)
```sql
SELECT r.map_id, r.map_name, count(*) n, sum(p.outcome=2) v, sum(p.outcome=3) d FROM match_registry r JOIN match_participants p ON p.match_id=r.match_id AND p.xuid='2533274823110022' GROUP BY 1,2 ORDER BY n DESC LIMIT 14
```
```
map_id	map_name	n	v	d
9e821f5e-042f-407c-97f3-de165b1cdb26	Illusion	54	30	24
3e1e4cec-4f2c-44c6-b8d2-96b85c66c702	Bazaar	50	22	26
9c7b0b0f-e933-4c2d-9d4a-3e4500d0de99	Streets	48	26	21
5324364b-39a8-4f93-96a6-b80a1f18ce8a	Cliffhanger	48	24	22
6c01f693-c968-4a71-b157-efc35ffcf71f	Live Fire	47	27	18
2b6d2baf-7645-4e16-8a80-c7006f595812	Recharge	45	19	26
2fdb8370-e5ac-4a1a-bdce-a08bc738b9ad	Prism	45	20	25
f7e8cde9-0c0a-487c-94a3-61bfa0f20465	Catalyst	44	20	23
c395f3ac-4614-45f9-a83a-56f69e8ae962	Aquarius	43	18	25
e9a5a982-6c4e-4db6-9383-7b03671460eb	Behemoth	43	26	16
87c03bfd-2db3-4a5b-bbf9-d5369c5894d1	Forbidden	41	15	22
e8d56863-9ad4-4efe-9059-81270884589c	Forest	40	22	16
a455572d-3141-48bc-ac55-dac78d9b52c9	Chasm	40	18	21
410f1c01-aca6-4567-9df5-9b16bd550cb2	Snowbound	23	9	12
(14 rows)
```
Fonds associés (champ `mapNames` des calages) : Illusion `ctf_illusion`, Bazaar `ctf_bazaar`, Streets `sgh_streets`,
Cliffhanger `ridgeline`, Live Fire `sgh_interlock`, Prism `sgh_crystalcaves`, Recharge `sgh_blueprint`, Catalyst
`catalyst`, Aquarius `ctf_aquarius`, Behemoth `va_behemoth`, Forbidden `ctf_forbidden`, Forest `forest`, Chasm
`chasm`, Snowbound `410f1c01-aca6-4567-9df5-9b16bd550cb2` (copié en `fonds/snowbound.webp`).

## Q2 — Noms de carte vides sur les deux témoins (écart au brief)
```sql
SELECT r.map_id, r.map_name, count(*) n, sum(p.outcome=2) v, sum(p.outcome=3) d, min(r.start_time_utc) a, max(r.start_time_utc) b FROM match_registry r JOIN match_participants p ON p.match_id=r.match_id AND p.xuid='2533274823110022' WHERE r.map_id IN ('9e821f5e-042f-407c-97f3-de165b1cdb26','3e1e4cec-4f2c-44c6-b8d2-96b85c66c702') GROUP BY 1,2
```
```
map_id	map_name	n	v	d	a	b
3e1e4cec-4f2c-44c6-b8d2-96b85c66c702	Bazaar	50	22	26	2025-10-23 20:17:49.143 +0000 UTC	2026-07-29 19:38:27.684 +0000 UTC
9e821f5e-042f-407c-97f3-de165b1cdb26	Illusion	54	30	24	2025-10-20 19:59:16.016 +0000 UTC	2026-09-07 19:42:15.92 +0000 UTC
9e821f5e-042f-407c-97f3-de165b1cdb26	NULL	2	1	1	2026-04-27 19:24:36.633 +0000 UTC	2026-04-27 20:52:37.413 +0000 UTC
3e1e4cec-4f2c-44c6-b8d2-96b85c66c702	NULL	1	0	1	2026-04-27 19:35:55.483 +0000 UTC	2026-04-27 19:35:55.483 +0000 UTC
(4 rows)
```
La maquette garde le périmètre du brief (`map_name` renseigné : 54 et 50 matchs) ; les 3 matchs du 27/04/2026 sont hors
plans et hors liste.

## Q3 / Q4 — Pied de la grille actuelle (« N cartes jouées … M matchs ») et Firefight
```sql
SELECT count(*) AS cartes, sum(n) AS matchs, sum(CASE WHEN n < 10 THEN 1 ELSE 0 END) AS sous_plancher FROM (SELECT r.map_id, r.map_name, count(*) n FROM match_registry r JOIN match_participants p ON p.match_id=r.match_id AND p.xuid='2533274823110022' WHERE r.map_id IS NOT NULL AND r.map_id <> '' AND COALESCE(r.is_firefight, FALSE) = FALSE GROUP BY 1,2)
```
```
cartes	matchs	sous_plancher
103	1158	61
(1 rows)
```
```sql
SELECT count(*) AS matchs_firefight FROM match_registry r JOIN match_participants p ON p.match_id=r.match_id AND p.xuid='2533274823110022' WHERE COALESCE(r.is_firefight, FALSE) = TRUE
```
```
matchs_firefight
2
(1 rows)
```

## Q5 — Matchs de chaque témoin (date locale Europe/Paris, issue et camp de JGtm, variante)
```sql
SELECT r.match_id, strftime(timezone('Europe/Paris', r.start_time_utc), '%Y-%m-%d %H:%M') AS loc, p.outcome, p.team_id, COALESCE(r.game_variant_name,'') AS gv, COALESCE(r.is_firefight,false) AS ff, COALESCE(r.map_name,'(NULL)') AS map_name FROM match_registry r JOIN match_participants p ON p.match_id=r.match_id AND p.xuid='<JGtm>' WHERE r.map_id='<map_id>' ORDER BY r.start_time_utc
```
Illusion (`ex/illu_matches.tsv`) :
```
match_id	loc	outcome	team_id	gv	ff	map_name
05fffb2a-50db-4fb2-b0c6-0dda57e3d44f	2025-10-20 21:59	2	0	Tactical:Slayer	false	Illusion
6d49207d-e39a-4cdc-99b3-605c5c6eebca	2025-10-23 22:07	2	0	Arena:Strongholds	false	Illusion
d2b74083-5dcd-4d41-871f-6096e831040e	2025-10-31 16:46	2	0	KOTH:Arena	false	Illusion
18ad5c88-6c49-4882-8c99-76e836e80b60	2025-11-03 20:48	2	1	Slayer:Arena	false	Illusion
415f2c6c-e882-428a-baf2-56570df3effb	2025-11-13 22:21	3	0	Strongholds:Arena	false	Illusion
a1f20336-40f8-4ac0-bc49-c6241821a207	2025-11-29 20:37	2	1	Slayer:Arena	false	Illusion
652907bb-7bf2-4539-9a03-b610d73ab265	2025-11-29 20:59	2	1	Team Slayer:Arena	false	Illusion
60ec0dfb-164c-4833-b2b3-19bf999fd247	2025-12-01 21:03	3	1	Team Slayer:Arena	false	Illusion
795c7f1e-e24f-4eae-ad13-dc5771fd0b1d	2025-12-20 11:37	3	0	Slayer:Arena Super Fiesta	false	Illusion
e13b44cc-494e-45a5-a3cd-f73f6b10a325	2025-12-22 17:26	2	0	Slayer:Arena Super Fiesta	false	Illusion
ea24327b-f507-49d1-9a7b-b0ba5f5dedc4	2026-01-04 20:04	2	1	Team Slayer:Arena	false	Illusion
a92bab93-6015-40d8-a42a-7039c16a52d1	2026-01-18 16:23	2	1	KOTH:Arena	false	Illusion
f6a9d127-0114-4fd5-bb09-d51412f5e4de	2026-01-21 21:35	3	1	Strongholds:Arena	false	Illusion
9126ddc0-5a18-461b-bc11-3eef34fa8a5d	2026-01-24 19:11	2	1	Slayer:Arena	false	Illusion
153a82d6-d26d-4ed8-9bf2-d5af28d79013	2026-01-24 19:24	2	0	Slayer:Arena Super Fiesta	false	Illusion
d39a5435-621b-48fe-9432-949bdd0963b0	2026-01-28 18:19	3	0	Slayer:Arena Super Fiesta	false	Illusion
e99e9ee1-502b-4e02-b1b4-31563a771bfe	2026-02-05 16:16	3	1	CTF:Arena	false	Illusion
cf3a088b-a570-4f12-9ec4-13b0f930ae2f	2026-02-06 19:05	2	1	Team Slayer:Arena	false	Illusion
7ff4271a-6f9f-4554-9c3e-4726b01c52fe	2026-02-07 15:30	3	1	CTF:Arena	false	Illusion
a2a023cf-9be4-4938-bd97-eab856bfd59f	2026-02-08 21:09	2	0	Slayer:Arena Super Fiesta	false	Illusion
1b6f7a27-7fd2-4813-9167-70cba83d217b	2026-02-10 21:35	2	0	CTF:Arena	false	Illusion
ae9c32df-7f03-49eb-bec2-fdd717dc1ab3	2026-02-11 16:50	2	0	Slayer:Arena Super Fiesta	false	Illusion
eb109229-456d-4153-a47a-7fa9dc676855	2026-02-11 17:46	2	0	Slayer:Arena Super Fiesta	false	Illusion
65bab728-c1e7-405b-a7d8-6de6f506fd03	2026-02-11 18:11	3	0	Slayer:Arena Super Fiesta	false	Illusion
07aa428d-ea13-4862-bcc0-58471610df4e	2026-02-11 20:27	3	1	Slayer:Arena Super Fiesta	false	Illusion
fb71123d-99f6-4bf0-978f-1129c8691175	2026-02-12 17:15	2	1	Slayer:Arena Super Fiesta	false	Illusion
3523df2b-1e7a-4269-a8dd-da2408515949	2026-02-15 18:16	3	0	Slayer:Arena Super Fiesta	false	Illusion
ec62cb21-3307-4e4e-a9bf-8976d751654f	2026-02-15 20:30	2	1	Slayer:Arena Super Fiesta	false	Illusion
ac041ffa-0b91-404b-af41-fc75f6e0f898	2026-02-16 22:01	2	0	Slayer:Arena Super Fiesta	false	Illusion
e869bcdf-a3db-4bff-b1b0-1eabe00befff	2026-02-17 22:19	3	1	Slayer:Arena Super Fiesta	false	Illusion
f7270782-01c2-47c8-8f13-7dfad815b5fb	2026-02-18 17:34	2	1	Slayer:Arena Super Fiesta	false	Illusion
963e8c72-cffa-40fe-90d3-a643c16163c3	2026-02-27 19:34	3	1	Slayer:Arena Super Fiesta	false	Illusion
4f1639b2-a433-4f86-b5b8-0d9304d9e13b	2026-03-05 20:01	2	1	Slayer:Arena Super Fiesta	false	Illusion
05ebe184-07af-461e-976a-50062fe495e2	2026-03-05 20:50	2	1	Slayer:Arena Super Fiesta	false	Illusion
549d808e-d735-4443-819e-35fd772e849c	2026-03-15 21:34	2	0	Slayer:Arena Super Fiesta	false	Illusion
2acc306f-da41-41e3-8dfd-9e12ab1c9926	2026-03-18 21:58	3	0	CTF:Arena	false	Illusion
386c917b-aa20-479f-a7a6-6296977355ec	2026-03-21 20:24	3	1	Slayer:Arena Super Fiesta	false	Illusion
49e6248f-f5ab-4e29-8a1b-74f3c4b64d10	2026-03-22 19:41	2	1	Slayer:Arena Super Fiesta	false	Illusion
c139818f-9378-4b24-8d9a-43f767ca8656	2026-03-31 21:24	2	1	Strongholds:Arena	false	Illusion
0727867d-ca7a-43dc-9845-bef7093a88e5	2026-03-31 22:20	3	0	Strongholds:Arena	false	Illusion
8faf5c41-0af2-4102-b687-60b297afc1c7	2026-03-31 22:45	2	1	CTF:Arena	false	Illusion
ac7ec523-1aab-4cb4-9aef-395f7e725ae6	2026-04-06 21:43	3	1	CTF:Arena	false	Illusion
6dd234b2-10ff-4492-b7e0-26e53f7c9388	2026-04-27 21:24	2	0	Team Slayer:Arena	false	(NULL)
e5b45563-0b6e-491e-8991-6516e01bba77	2026-04-27 22:52	3	1	Strongholds:Arena	false	(NULL)
a26d3c4d-119f-42b1-879d-4e55d3412ee6	2026-05-21 22:05	3	0	CTF:Arena	false	Illusion
cf6465d1-d0cf-4e9e-8620-cbb5c6620fed	2026-05-23 20:09	3	0	Slayer:Arena Super Fiesta	false	Illusion
09922983-e652-4824-99fc-f2020b28b42b	2026-05-25 14:50	2	0	Slayer:Arena Super Fiesta	false	Illusion
c7b37f05-cf78-4039-9cd5-e207c37f5760	2026-05-26 22:12	3	1	Slayer:Arena	false	Illusion
395251d2-4733-40c1-b6ff-23580fd0a17d	2026-06-09 20:52	2	1	Slayer:Arena Super Fiesta	false	Illusion
e6b8a4e6-24c4-4c5d-9d6c-fa9f8dd040dc	2026-06-28 20:09	3	1	Slayer:Arena Super Fiesta	false	Illusion
350f915f-78aa-4deb-ae40-2947a6643b5d	2026-07-16 21:49	3	1	Slayer:Arena Super Fiesta	false	Illusion
fbe4ebeb-c97a-4aff-87de-919012540979	2026-07-17 20:50	2	0	Slayer:Arena Super Fiesta	false	Illusion
fbef8f77-a290-4e69-8d15-0416475e73c4	2026-07-19 14:46	3	1	Slayer:Arena Super Fiesta	false	Illusion
bf5ced1b-4efc-47d9-b0ec-050116df71f4	2026-07-28 21:59	3	1	CTF:Arena	false	Illusion
bc60b4d9-40dc-4790-a533-3197aea9060a	2026-07-28 22:59	3	1	CTF:Arena	false	Illusion
396cfc92-6f9e-403b-a46f-20bc73b38c11	2026-09-07 21:42	2	0	Strongholds:Arena	false	Illusion
(56 rows)
```
Bazaar (`ex/baz_matches.tsv`) :
```
match_id	loc	outcome	team_id	gv	ff	map_name
c7d40d45-c7bb-497c-bf0e-19e5c4e46139	2025-10-23 22:17	2	1	Arena:Slayer	false	Bazaar
f65627bd-313c-41f0-8f25-111b88d3276a	2025-11-03 21:39	3	1	Slayer:Arena Tactical	false	Bazaar
e94163af-3785-46bd-9cef-8d85cc5687b3	2025-12-03 21:57	3	0	CTF:Arena Neutral Flag	false	Bazaar
323ec1cf-6cd1-4301-9312-170c1df45849	2025-12-23 18:38	2	1	CTF:Arena Neutral Flag	false	Bazaar
fb1c5d79-1789-4176-bd3f-063053a3a5e5	2026-01-09 20:37	2	0	Slayer:Arena	false	Bazaar
735b92ac-763d-4bb9-954f-2c98a629f024	2026-01-21 21:17	3	0	CTF:Arena	false	Bazaar
9903b1c5-b5e5-40d2-bd81-83d540b233cf	2026-01-22 20:40	2	1	Arena:VIP	false	Bazaar
0e64e07b-2cb4-452e-a153-c7c087d2523d	2026-01-26 19:01	3	1	Slayer:Arena Super Fiesta	false	Bazaar
7c554aa7-2a0a-431c-85e3-4106d3fb09a4	2026-01-26 19:12	3	1	Slayer:Arena	false	Bazaar
666c7cbc-d549-482a-bc41-8b11a45df4f6	2026-01-28 18:12	3	0	KOTH:Arena	false	Bazaar
189d1c23-b006-421a-9515-f978edc0dc45	2026-02-01 18:52	3	0	Slayer:Arena Super Fiesta	false	Bazaar
00761d27-487c-4d7d-ac4c-bf7584de652c	2026-02-03 15:44	3	0	Arena:VIP	false	Bazaar
340584f6-1fc5-4dce-afe2-1ba96c387615	2026-02-08 21:22	3	1	Slayer:Arena Super Fiesta	false	Bazaar
a7a2fce3-c9da-411c-bdb7-389a541f156b	2026-02-10 20:44	3	1	Slayer:Arena Super Fiesta	false	Bazaar
a3af2b23-a4d3-488c-ab46-9bca3008985a	2026-02-15 20:14	3	1	Slayer:Arena Super Fiesta	false	Bazaar
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2026-02-17 21:33	2	1	Slayer:Arena Super Fiesta	false	Bazaar
53a3768e-4619-45dc-8964-00bf968280dd	2026-02-17 22:07	3	1	Slayer:Arena Super Fiesta	false	Bazaar
e44bfaaa-877b-4893-99c0-aa94271bb8e9	2026-02-20 17:35	2	0	Slayer:Arena Super Fiesta	false	Bazaar
f1ae8345-e0a6-49b9-a0eb-3e861541bef9	2026-03-05 20:29	4	1	Slayer:Arena Super Fiesta	false	Bazaar
0e697aca-550b-4fc9-b6f2-df7989e65bc0	2026-03-05 21:09	2	0	Slayer:Arena Super Fiesta	false	Bazaar
147ffd4d-3d1d-4b90-a46d-5570009f8c36	2026-03-06 21:09	3	1	Slayer:Arena Super Fiesta	false	Bazaar
4e49d934-b030-414b-98de-8615551e25e1	2026-03-11 20:54	3	0	Slayer:Arena Super Fiesta	false	Bazaar
cc2035f9-e867-45dd-aa64-721ef8df32c6	2026-03-12 22:49	2	0	Team Slayer:Arena	false	Bazaar
c7ba5c8c-8e1d-405d-a6ab-2389f50af165	2026-03-15 16:45	3	1	Slayer:Arena Super Fiesta	false	Bazaar
ce46a96e-2429-4843-9836-fd96d6a6f78a	2026-03-16 19:10	2	1	Team Slayer:Arena	false	Bazaar
64f405f9-5959-4cc7-a5e1-afdb94d21fc8	2026-03-17 20:20	2	0	Slayer:Arena Super Fiesta	false	Bazaar
82f3af9f-c0fa-477b-be9b-df240d62305d	2026-03-18 19:17	3	1	Slayer:Arena Super Fiesta	false	Bazaar
de550699-ed03-4ad5-8eac-6c5a6ad8d7bb	2026-03-21 19:50	3	0	Slayer:Arena Super Fiesta	false	Bazaar
941f7f8c-87b6-4f7e-871e-5ac38bfb1202	2026-03-21 20:13	2	1	Slayer:Arena Super Fiesta	false	Bazaar
aa823fc2-17a7-4412-ba69-2fda05c8d73c	2026-03-28 18:18	2	1	Slayer:Arena Super Fiesta	false	Bazaar
9b191a7f-72ae-417b-9789-cd0d870a43cf	2026-04-06 23:40	3	0	Team Slayer:Arena	false	Bazaar
6bf3090a-cc5c-4a05-a304-b54d94eb1b9c	2026-04-08 20:22	2	0	Slayer:Arena Super Fiesta	false	Bazaar
67c5503d-264e-46fa-aa21-51dc5829aae6	2026-04-27 21:35	3	0	Team Slayer:Arena	false	(NULL)
48aa9494-1e04-4e9f-a5e2-9c75f85662ce	2026-05-11 22:54	2	0	CTF:Arena	false	Bazaar
4d8d1644-0ee3-4acc-a473-0d264086670b	2026-05-21 21:53	3	1	CTF:Arena	false	Bazaar
636782a0-6cd1-4b55-a769-73ed74e52dd0	2026-05-24 23:03	2	0	Slayer:Arena Super Fiesta	false	Bazaar
4510b5e0-37ce-4962-b7c0-554a6579a9d4	2026-06-09 20:42	2	1	Slayer:Arena Super Fiesta	false	Bazaar
a224dc60-8903-462c-bd35-a1c3230696fa	2026-06-09 22:23	2	0	CTF:Arena	false	Bazaar
789a0aa0-481d-4a17-81b4-e88e6772a9bb	2026-06-27 19:12	2	0	Slayer:Arena Super Fiesta	false	Bazaar
5ceef09f-18d5-4f92-b173-7cb33eb4e9e6	2026-06-28 20:44	3	1	Slayer:Arena Super Fiesta	false	Bazaar
30929866-f0f3-4d05-afb0-c2c7aeb57028	2026-07-03 21:30	3	1	Slayer:Arena	false	Bazaar
4d4a3961-7b69-447b-9912-5aa1242c6f09	2026-07-16 15:53	2	1	Slayer:Arena Super Fiesta	false	Bazaar
f0a0115a-26cf-4c74-ae69-54f2b708109d	2026-07-16 21:36	2	1	Slayer:Arena Super Fiesta	false	Bazaar
16bf0b50-c6af-4122-a1ba-e2ae7c1183d2	2026-07-18 22:19	3	1	Slayer:Arena Super Fiesta	false	Bazaar
ec422a43-993e-4451-aaf9-4145a39913f4	2026-07-19 15:36	2	0	Slayer:Arena Super Fiesta	false	Bazaar
ab4eef5a-e9f8-493a-a413-4c5afe2268a7	2026-07-22 17:56	3	0	Slayer:Arena Super Fiesta	false	Bazaar
c0dace44-7e12-42c9-97b4-1a9feb4c56ab	2026-07-23 17:17	3	0	Slayer:Arena Super Fiesta	false	Bazaar
cecbc053-3213-4e32-8698-02ded2a87560	2026-07-23 19:46	3	0	Slayer:Arena Super Fiesta	false	Bazaar
c7f94693-29c4-4c41-b6b2-532e47f7e920	2026-07-28 11:58	4	1	Slayer:Arena Super Fiesta	false	Bazaar
bf2a9f05-2818-4209-abb8-9e57d74ee78e	2026-07-28 20:00	2	1	Slayer:Arena Super Fiesta	false	Bazaar
9ffce8ef-a755-4b7c-a558-9e1e99664dae	2026-07-29 21:38	3	1	Slayer:Arena Super Fiesta	false	Bazaar
(51 rows)
```

## Q6 — Journal des morts par match (lignes, lignes publiables)
```sql
SELECT e.match_id, count(*) AS lignes, count(*) FILTER (WHERE e.publishable) AS pub FROM match_kill_events_latest e WHERE e.match_id IN (<liste des match_id de la carte : ex/{illu,baz}_ids.txt>) GROUP BY 1 ORDER BY 1
```
Illusion (`ex/illu_journal.tsv`) :
```
match_id	lignes	pub
05ebe184-07af-461e-976a-50062fe495e2	77	77
05fffb2a-50db-4fb2-b0c6-0dda57e3d44f	92	92
0727867d-ca7a-43dc-9845-bef7093a88e5	96	96
07aa428d-ea13-4862-bcc0-58471610df4e	91	91
09922983-e652-4824-99fc-f2020b28b42b	94	94
153a82d6-d26d-4ed8-9bf2-d5af28d79013	87	87
18ad5c88-6c49-4882-8c99-76e836e80b60	89	89
1b6f7a27-7fd2-4813-9167-70cba83d217b	41	41
2acc306f-da41-41e3-8dfd-9e12ab1c9926	99	99
350f915f-78aa-4deb-ae40-2947a6643b5d	81	81
3523df2b-1e7a-4269-a8dd-da2408515949	86	86
386c917b-aa20-479f-a7a6-6296977355ec	97	97
395251d2-4733-40c1-b6ff-23580fd0a17d	96	96
396cfc92-6f9e-403b-a46f-20bc73b38c11	67	67
415f2c6c-e882-428a-baf2-56570df3effb	118	118
49e6248f-f5ab-4e29-8a1b-74f3c4b64d10	99	99
4f1639b2-a433-4f86-b5b8-0d9304d9e13b	99	99
549d808e-d735-4443-819e-35fd772e849c	82	82
60ec0dfb-164c-4833-b2b3-19bf999fd247	98	98
652907bb-7bf2-4539-9a03-b610d73ab265	89	89
65bab728-c1e7-405b-a7d8-6de6f506fd03	95	95
6d49207d-e39a-4cdc-99b3-605c5c6eebca	82	82
6dd234b2-10ff-4492-b7e0-26e53f7c9388	97	97
795c7f1e-e24f-4eae-ad13-dc5771fd0b1d	98	98
7ff4271a-6f9f-4554-9c3e-4726b01c52fe	132	132
8faf5c41-0af2-4102-b687-60b297afc1c7	90	90
9126ddc0-5a18-461b-bc11-3eef34fa8a5d	88	88
963e8c72-cffa-40fe-90d3-a643c16163c3	86	86
a1f20336-40f8-4ac0-bc49-c6241821a207	85	85
a26d3c4d-119f-42b1-879d-4e55d3412ee6	96	96
a2a023cf-9be4-4938-bd97-eab856bfd59f	95	95
a92bab93-6015-40d8-a42a-7039c16a52d1	78	0
ac041ffa-0b91-404b-af41-fc75f6e0f898	96	96
ac7ec523-1aab-4cb4-9aef-395f7e725ae6	81	81
ae9c32df-7f03-49eb-bec2-fdd717dc1ab3	97	97
bc60b4d9-40dc-4790-a533-3197aea9060a	81	81
bf5ced1b-4efc-47d9-b0ec-050116df71f4	26	26
c139818f-9378-4b24-8d9a-43f767ca8656	103	103
c7b37f05-cf78-4039-9cd5-e207c37f5760	95	95
cf3a088b-a570-4f12-9ec4-13b0f930ae2f	94	94
cf6465d1-d0cf-4e9e-8620-cbb5c6620fed	82	82
d2b74083-5dcd-4d41-871f-6096e831040e	49	49
d39a5435-621b-48fe-9432-949bdd0963b0	90	90
e13b44cc-494e-45a5-a3cd-f73f6b10a325	95	95
e5b45563-0b6e-491e-8991-6516e01bba77	81	81
e6b8a4e6-24c4-4c5d-9d6c-fa9f8dd040dc	95	95
e869bcdf-a3db-4bff-b1b0-1eabe00befff	92	92
e99e9ee1-502b-4e02-b1b4-31563a771bfe	132	132
ea24327b-f507-49d1-9a7b-b0ba5f5dedc4	96	96
eb109229-456d-4153-a47a-7fa9dc676855	97	97
ec62cb21-3307-4e4e-a9bf-8976d751654f	90	90
f6a9d127-0114-4fd5-bb09-d51412f5e4de	91	91
f7270782-01c2-47c8-8f13-7dfad815b5fb	97	97
fb71123d-99f6-4bf0-978f-1129c8691175	83	83
fbe4ebeb-c97a-4aff-87de-919012540979	90	90
fbef8f77-a290-4e69-8d15-0416475e73c4	85	85
(56 rows)
```
Bazaar (`ex/baz_journal.tsv`) :
```
match_id	lignes	pub
00502e52-50cf-43dc-9c29-38e40ec3ab5a	95	95
00761d27-487c-4d7d-ac4c-bf7584de652c	67	67
0e64e07b-2cb4-452e-a153-c7c087d2523d	92	0
0e697aca-550b-4fc9-b6f2-df7989e65bc0	91	91
147ffd4d-3d1d-4b90-a46d-5570009f8c36	86	0
16bf0b50-c6af-4122-a1ba-e2ae7c1183d2	84	84
189d1c23-b006-421a-9515-f978edc0dc45	99	99
30929866-f0f3-4d05-afb0-c2c7aeb57028	84	84
323ec1cf-6cd1-4301-9312-170c1df45849	97	97
340584f6-1fc5-4dce-afe2-1ba96c387615	92	92
4510b5e0-37ce-4962-b7c0-554a6579a9d4	88	88
48aa9494-1e04-4e9f-a5e2-9c75f85662ce	95	95
4d4a3961-7b69-447b-9912-5aa1242c6f09	78	78
4d8d1644-0ee3-4acc-a473-0d264086670b	90	90
4e49d934-b030-414b-98de-8615551e25e1	99	99
53a3768e-4619-45dc-8964-00bf968280dd	97	97
5ceef09f-18d5-4f92-b173-7cb33eb4e9e6	98	98
636782a0-6cd1-4b55-a769-73ed74e52dd0	95	95
64f405f9-5959-4cc7-a5e1-afdb94d21fc8	94	94
666c7cbc-d549-482a-bc41-8b11a45df4f6	36	36
67c5503d-264e-46fa-aa21-51dc5829aae6	99	99
6bf3090a-cc5c-4a05-a304-b54d94eb1b9c	94	94
735b92ac-763d-4bb9-954f-2c98a629f024	134	134
789a0aa0-481d-4a17-81b4-e88e6772a9bb	94	94
7c554aa7-2a0a-431c-85e3-4106d3fb09a4	87	87
82f3af9f-c0fa-477b-be9b-df240d62305d	94	94
941f7f8c-87b6-4f7e-871e-5ac38bfb1202	91	91
9903b1c5-b5e5-40d2-bd81-83d540b233cf	56	56
9b191a7f-72ae-417b-9789-cd0d870a43cf	90	90
9ffce8ef-a755-4b7c-a558-9e1e99664dae	95	95
a224dc60-8903-462c-bd35-a1c3230696fa	130	130
a3af2b23-a4d3-488c-ab46-9bca3008985a	94	94
a7a2fce3-c9da-411c-bdb7-389a541f156b	87	87
aa823fc2-17a7-4412-ba69-2fda05c8d73c	91	91
ab4eef5a-e9f8-493a-a413-4c5afe2268a7	94	94
bf2a9f05-2818-4209-abb8-9e57d74ee78e	96	96
c0dace44-7e12-42c9-97b4-1a9feb4c56ab	91	91
c7ba5c8c-8e1d-405d-a6ab-2389f50af165	92	92
c7d40d45-c7bb-497c-bf0e-19e5c4e46139	95	95
c7f94693-29c4-4c41-b6b2-532e47f7e920	85	85
cc2035f9-e867-45dd-aa64-721ef8df32c6	92	92
ce46a96e-2429-4843-9836-fd96d6a6f78a	89	89
cecbc053-3213-4e32-8698-02ded2a87560	97	97
de550699-ed03-4ad5-8eac-6c5a6ad8d7bb	92	92
e44bfaaa-877b-4893-99c0-aa94271bb8e9	96	96
e94163af-3785-46bd-9cef-8d85cc5687b3	33	33
ec422a43-993e-4451-aaf9-4145a39913f4	98	98
f0a0115a-26cf-4c74-ae69-54f2b708109d	99	99
f1ae8345-e0a6-49b9-a0eb-3e861541bef9	99	99
f65627bd-313c-41f0-8f25-111b88d3276a	89	89
fb1c5d79-1789-4176-bd3f-063053a3a5e5	82	82
(51 rows)
```

## Q7 — Morts localisées (`QTacticalPositions` de platform/duckdb/tactical_repo.go, plus `killer_z` / `victim_z` pour le nom de zone (v2), gamertags, source et catégorie de dégât pour les tuiles « Rejeu » (v4))
```sql
SELECT kp.match_id, COALESCE(kp.killer_xuid, '') AS killer_xuid, COALESCE(min(e.victim_xuid), '') AS victim_xuid, min(kp.killer_x) AS killer_x, min(kp.killer_y) AS killer_y, min(kp.victim_x) AS victim_x, min(kp.victim_y) AS victim_y, kp.time_ms AS time_ms, min(kp.killer_z) AS killer_z, min(kp.victim_z) AS victim_z, COALESCE(min(e.feed_killer_gamertag), '') AS kgt, COALESCE(min(e.victim_gamertag), '') AS vgt, COALESCE(printf('%08x', min(e.source_tag)), '') AS tag, COALESCE(min(e.source_category), '') AS cat FROM kill_positions_latest kp JOIN match_kill_events_latest e ON e.match_id = kp.match_id AND e.feed_killer_xuid = kp.killer_xuid AND e.time_ms = kp.time_ms WHERE kp.match_id IN (<liste des match_id de la carte : ex/{illu,baz}_ids.txt>) AND e.match_id IN (<liste des match_id de la carte : ex/{illu,baz}_ids.txt>) AND e.publishable AND kp.killer_x IS NOT NULL AND kp.killer_y IS NOT NULL AND kp.victim_x IS NOT NULL AND kp.victim_y IS NOT NULL GROUP BY kp.match_id, kp.killer_xuid, kp.time_ms HAVING count(*) = 1 ORDER BY kp.match_id, kp.time_ms
```
Illusion (`ex/illu_pos.tsv`, 4 540 lignes sur les 56 matchs de la carte, 55 matchs) :
```
match_id	killer_xuid	victim_xuid	killer_x	killer_y	victim_x	victim_y	time_ms	killer_z	victim_z	kgt	vgt	tag	cat
05ebe184-07af-461e-976a-50062fe495e2	2535422821674619	2535427308376866	0.19298546016216278	11.730963706970215	0.4377475082874298	-12.851893424987793	41145	-0.0014723148196935654	-0.0014723148196935654	Elcheke3141	jkrebornX	64ab85c4	None
05ebe184-07af-461e-976a-50062fe495e2	2535410609299646	2533274823110022	-0.7201652526855469	12.774290084838867	-0.28712472319602966	12.981231689453125	50504	-0.0014723148196935654	-0.0014723148196935654	Crashbanana0	JGtm	daa03c35	None
05ebe184-07af-461e-976a-50062fe495e2	2535456538207904	2535439072500441	3.054818630218506	-4.013791561126709	3.6008262634277344	-4.513898849487305	53606	7.210720062255859	6.3224711418151855	HopingGamerXBOX	TUG103	1fc1a4e6	None
05ebe184-07af-461e-976a-50062fe495e2	2535422821674619	2535427308376866	8.05361270904541	-0.9096778035163879	4.372767925262451	2.5910727977752686	64750	3.47428297996521	0.423342227935791	Elcheke3141	jkrebornX	ba5ac78e	None
05ebe184-07af-461e-976a-50062fe495e2	2535410609299646	2535456538207904	-2.74415922164917	-0.6510016322135925	-3.9114859104156494	-3.608532190322876	72792	-0.0014723148196935654	1.533652901649475	Crashbanana0	HopingGamerXBOX	119861b4	AttachedDamage
05ebe184-07af-461e-976a-50062fe495e2	2535422821674619	2535439072500441	7.695883750915527	14.964415550231934	7.573502540588379	19.663698196411133	78931	2.489485740661621	3.0687782764434814	Elcheke3141	TUG103	00404748	None
05ebe184-07af-461e-976a-50062fe495e2	2533274823110022	2535427308376866	-4.419837951660156	-5.324417591094971	-3.4125478267669678	3.625777244567871	82802	-0.802827000617981	2.9046452045440674	JGtm	jkrebornX	fc1ca9c2	Headshot
05ebe184-07af-461e-976a-50062fe495e2	2535410609299646	2533274823110022	-12.873542785644531	2.409999370574951	-10.463577270507812	-2.634185552597046	92061	2.0743260383605957	1.0026347637176514	Crashbanana0	JGtm	4a555df8	AttachedDamage
05ebe184-07af-461e-976a-50062fe495e2	2535439072500441	2535456538207904	14.44566822052002	-12.446634292602539	11.913322448730469	-12.843271255493164	103323	1.1957322359085083	1.6302016973495483	TUG103	HopingGamerXBOX	bebe70df	None
05ebe184-07af-461e-976a-50062fe495e2	2535418835029748	2535439072500441	12.449915885925293	-12.489747047424316	12.223981857299805	-12.541481971740723	105475	1.215041995048523	1.3405554294586182	makus1me2443	TUG103	166ba62b	None
05ebe184-07af-461e-976a-50062fe495e2	2535410318851859	2535422821674619	4.598702430725098	3.444704055786133	6.829802513122559	10.187528610229492	106743	-0.802827000617981	2.1998393535614014	NotTauro3005	Elcheke3141	4a555df8	AttachedDamage
…
(4540 rows)
```
Bazaar (`ex/baz_pos.tsv`, 4 016 lignes, 49 matchs) :
```
match_id	killer_xuid	victim_xuid	killer_x	killer_y	victim_x	victim_y	time_ms	killer_z	victim_z	kgt	vgt	tag	cat
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2535441202785466	2535420181710097	-5.467294216156006	-4.93788480758667	-2.0270626544952393	-7.151477813720703	57427	0.04174726456403732	2.20251727104187	jalejandropb25	AvoidinNorml	4a555df8	AttachedDamage
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2535425717197133	2535441202785466	-3.2324633598327637	4.518240451812744	-5.440898418426514	-6.345558166503906	58128	0.13836707174777985	0.05053088441491127	superjp12345	jalejandropb25	4fbad006	None
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2533274806463389	2533274793912226	-12.356554985046387	-5.453673362731934	-12.321361541748047	-3.4335014820098877	65606	3.2038497924804688	3.581545352935791	OH OH Digo Yo	BadWolf	fc1ca9c2	None
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2535425717197133	2535453521991010	-4.270691871643066	7.333586692810059	-3.3908371925354004	-0.7148653864860535	68358	2.4133241176605225	2.1585991382598877	superjp12345	KindaFlounder21	4a555df8	AttachedDamage
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2533274806463389	2535441202785466	-14.441810607910156	-5.1635422706604	-17.82925033569336	7.172402858734131	69226	3.0720953941345215	2.949124813079834	OH OH Digo Yo	jalejandropb25	fc1ca9c2	None
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2533274806463389	2533274823110022	-13.949091911315918	7.086437702178955	-9.998544692993164	6.259027004241943	78538	2.817370653152466	2.817370653152466	OH OH Digo Yo	JGtm	fc1ca9c2	None
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2533274806463389	2533274793912226	-14.916932106018066	7.312095642089844	-13.861106872558594	3.733811616897583	81641	2.808587074279785	2.9930429458618164	OH OH Digo Yo	BadWolf	fc1ca9c2	None
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2535453521991010	2535425717197133	3.102489471435547	5.14148473739624	2.88252592086792	8.913188934326172	86029	0.13836707174777985	0.4018756151199341	KindaFlounder21	superjp12345	69fd30b9	None
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2535420181710097	2535453521991010	5.249334812164307	-2.06881046295166	4.413472652435303	6.506175518035889	87346	0.05053088441491127	0.2789049446582794	AvoidinNorml	KindaFlounder21	13ff7bdf	None
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2533274806463389	2535441202785466	-9.998544692993164	3.8627588748931885	14.179858207702637	4.453766822814941	95048	2.817370653152466	2.7998032569885254	OH OH Digo Yo	jalejandropb25	7d012346	Headshot
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2533274823110022	2535435551873274	12.3761568069458	-5.582620620727539	16.12433624267578	-5.378454208374023	106728	3.2038497924804688	3.0984463691711426	JGtm	LordFlacco9189	17d6bd23	None
…
(4016 rows)
```

## Q8 — Contexte de chaque mort (Morts seul, isolement de l'avant)
```sql
SELECT match_id, victim_xuid, time_ms, nearest_teammate_m, teammates_visible, teammates_out_of_sight, teammates_waiting, teammates_left, teammates_total FROM match_death_context_latest WHERE match_id IN (<liste des match_id de la carte : ex/{illu,baz}_ids.txt>) ORDER BY match_id, time_ms
```
Illusion (`ex/illu_ctx.tsv`) :
```
match_id	victim_xuid	time_ms	nearest_teammate_m	teammates_visible	teammates_out_of_sight	teammates_waiting	teammates_left	teammates_total
05ebe184-07af-461e-976a-50062fe495e2	2535427308376866	41145	2.13	2	2	0	0	4
05ebe184-07af-461e-976a-50062fe495e2	2533274823110022	50504	14.09	3	0	0	0	3
05ebe184-07af-461e-976a-50062fe495e2	2535439072500441	53606	11.94	2	2	0	0	4
05ebe184-07af-461e-976a-50062fe495e2	2535427308376866	64750	11.22	2	2	0	0	4
05ebe184-07af-461e-976a-50062fe495e2	2535456538207904	72792	11.41	3	0	0	0	3
05ebe184-07af-461e-976a-50062fe495e2	2535410609299646	72992	22.34	1	3	0	0	4
05ebe184-07af-461e-976a-50062fe495e2	2535439072500441	78931	15.54	1	3	0	0	4
05ebe184-07af-461e-976a-50062fe495e2	2535427308376866	82802	13.99	1	3	0	0	4
05ebe184-07af-461e-976a-50062fe495e2	2533274823110022	92061	17.83	3	0	0	0	3
05ebe184-07af-461e-976a-50062fe495e2	2535456538207904	103323	5.66	3	0	0	0	3
05ebe184-07af-461e-976a-50062fe495e2	2535439072500441	105475	3.43	3	1	0	0	4
…
(4922 rows)
```
Bazaar (`ex/baz_ctx.tsv`) :
```
match_id	victim_xuid	time_ms	nearest_teammate_m	teammates_visible	teammates_out_of_sight	teammates_waiting	teammates_left	teammates_total
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2535420181710097	57427	10.46	2	1	0	0	3
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2535441202785466	58128	11.84	3	0	0	0	3
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2533274793912226	65606	9.76	2	1	0	0	3
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2535453521991010	68358	8.73	2	1	0	0	3
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2535425717197133	68825	17.1	3	0	0	0	3
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2535441202785466	69226	8.64	1	2	0	0	3
00502e52-50cf-43dc-9c29-38e40ec3ab5a	2533274823110022	78538	3.16	3	0	0	0	3
…
(4523 rows)
```
Jointure aux positions sur (match_id, victim_xuid, time_ms) exact. Seul (brief) = `teammates_visible = 0` ou
`nearest_teammate_m` ≥ portée du radar de la variante (`regulation.toml [radar_range_m]` : 18 m pour toutes les variantes
d'arène des deux cartes).

## Q9 — Camps (axe « Adversaires », riposte)
```sql
SELECT match_id, xuid, team_id, COALESCE(gamertag,'') AS gamertag, outcome FROM match_participants WHERE match_id IN (<liste des match_id de la carte : ex/{illu,baz}_ids.txt>) ORDER BY match_id, team_id
```
```
match_id	xuid	team_id	gamertag	outcome
05ebe184-07af-461e-976a-50062fe495e2	2535410609299646	0		3
05ebe184-07af-461e-976a-50062fe495e2	2535410318851859	0		4
05ebe184-07af-461e-976a-50062fe495e2	2535439072500441	0		3
05ebe184-07af-461e-976a-50062fe495e2	2535427308376866	0		3
05ebe184-07af-461e-976a-50062fe495e2	bid(12.0)	0		3
05ebe184-07af-461e-976a-50062fe495e2	bid(50.0)	0		4
05ebe184-07af-461e-976a-50062fe495e2	2535438446338716	0		4
05ebe184-07af-461e-976a-50062fe495e2	2535418835029748	1		2
05ebe184-07af-461e-976a-50062fe495e2	2535456538207904	1		2
…
(503 rows)
```
(Bazaar : `ex/baz_parts.tsv`, 482 lignes.) Les bots (`bid(…)`) ont un camp : ils comptent dans « Adversaires ».

## Q10 — Journal publiable complet (riposte de l'avant)
```sql
SELECT match_id, time_ms, COALESCE(feed_killer_xuid,'') AS killer, COALESCE(victim_xuid,'') AS victim FROM match_kill_events_latest WHERE match_id IN (<liste des match_id de la carte : ex/{illu,baz}_ids.txt>) AND publishable ORDER BY match_id, time_ms
```
```
match_id	time_ms	killer	victim
05ebe184-07af-461e-976a-50062fe495e2	41145	2535422821674619	2535427308376866
05ebe184-07af-461e-976a-50062fe495e2	50504	2535410609299646	2533274823110022
05ebe184-07af-461e-976a-50062fe495e2	53606	2535456538207904	2535439072500441
05ebe184-07af-461e-976a-50062fe495e2	64750	2535422821674619	2535427308376866
05ebe184-07af-461e-976a-50062fe495e2	72792	2535410609299646	2535456538207904
05ebe184-07af-461e-976a-50062fe495e2	72992	2535456538207904	2535410609299646
05ebe184-07af-461e-976a-50062fe495e2	78931	2535422821674619	2535439072500441
…
(4940 rows)
```
(Bazaar : `ex/baz_ev.tsv`, 4 434 lignes.) Riposte = règle de `analysis/coordination/trade.go` : mort de JGtm vengeable
(tueur connu, deux camps connus et différents), vengée si un coéquipier abat le tueur dans les 5 000 ms (bornes comprises).

## Q11 — Matchs d'une même carte dans une même soirée (avis sur le filtre « Sessions »)
```sql
WITH s AS (SELECT strftime(timezone('Europe/Paris', r.start_time_utc - INTERVAL 6 HOUR), '%Y-%m-%d') AS soir, r.map_id, count(*) n FROM match_registry r JOIN match_participants p ON p.match_id=r.match_id AND p.xuid='2533274823110022' WHERE COALESCE(r.is_firefight,false)=false GROUP BY 1,2) SELECT n AS matchs_meme_carte_meme_soiree, count(*) AS couples_soiree_carte, round(100.0*count(*)/sum(count(*)) OVER (), 1) AS pct FROM s GROUP BY 1 ORDER BY 1
```
```
matchs_meme_carte_meme_soiree	couples_soiree_carte	pct
1	865	86.1
2	130	12.9
3	8	0.8
4	1	0.1
5	1	0.1
(5 rows)
```
99 % des couples (soirée, carte) comptent au plus 2 matchs : une session seule ne peut atteindre le plancher de 3 matchs
distincts par zone.

## Q12 — Toutes les cartes jouées (repli « cartes sous le plancher », v2)
```sql
SELECT r.map_id, COALESCE(r.map_name,'(nom vide)') AS map_name, count(*) n, sum(p.outcome=2) v, sum(p.outcome=3) d FROM match_registry r JOIN match_participants p ON p.match_id=r.match_id AND p.xuid='2533274823110022' WHERE r.map_id IS NOT NULL AND r.map_id <> '' AND COALESCE(r.is_firefight, FALSE) = FALSE GROUP BY r.map_id, r.map_name ORDER BY n DESC, r.map_id
```
```
map_id	map_name	n	v	d
9e821f5e-042f-407c-97f3-de165b1cdb26	Illusion	54	30	24
3e1e4cec-4f2c-44c6-b8d2-96b85c66c702	Bazaar	50	22	26
5324364b-39a8-4f93-96a6-b80a1f18ce8a	Cliffhanger	48	24	22
9c7b0b0f-e933-4c2d-9d4a-3e4500d0de99	Streets	48	26	21
6c01f693-c968-4a71-b157-efc35ffcf71f	Live Fire	47	27	18
2b6d2baf-7645-4e16-8a80-c7006f595812	Recharge	45	19	26
2fdb8370-e5ac-4a1a-bdce-a08bc738b9ad	Prism	45	20	25
f7e8cde9-0c0a-487c-94a3-61bfa0f20465	Catalyst	44	20	23
c395f3ac-4614-45f9-a83a-56f69e8ae962	Aquarius	43	18	25
e9a5a982-6c4e-4db6-9383-7b03671460eb	Behemoth	43	26	16
87c03bfd-2db3-4a5b-bbf9-d5369c5894d1	Forbidden	41	15	22
a455572d-3141-48bc-ac55-dac78d9b52c9	Chasm	40	18	21
e8d56863-9ad4-4efe-9059-81270884589c	Forest	40	22	16
410f1c01-aca6-4567-9df5-9b16bd550cb2	Snowbound	23	9	12
5646ce03-a86e-40d7-9cac-08d442d32606	Launch Site	23	8	15
b302eb62-da9a-480b-a409-3c89df8c1a04	Origin	22	13	8
d39600e2-3c35-4a3a-bdf5-7b3cbdde98e1	Detachment	22	12	10
648ae7aa-c5d0-4f80-861a-79eb30440fcb	The Pit	18	9	9
78da545f-a168-4a5e-9c8d-dd379067c352	Absolution	18	7	11
cfd90b63-62fd-441a-8015-8d7804b9c3c3	Dynasty	18	11	6
2be34415-bc96-4d02-875c-c4f2aa135f89	Nemesis	17	14	2
7a9265af-a880-487b-8829-68d88fcfb145	Starboard	17	9	8
921aebb1-783d-45e4-bacd-7ad869fa8dae	Domicile	17	7	10
0d1c9255-d912-416c-befc-5f3e5e176df2	Fortress	16	8	8
4bffd021-92c0-422b-8b6e-8f595511458c	Cliffside	16	7	6
504ebf22-12b6-46c3-a9c1-ea20ca5bf03c	Goliath	16	8	8
01af558d-53ab-4f05-ba68-92d805fc6260	Isolation	15	5	7
2890782c-0a33-4f2c-a468-e3a7d6cd6db4	Shiro	15	5	10
63d634be-0319-489d-8c21-9c4e012f664f	Curfew	15	6	8
105f5d84-8de1-4908-af3a-1c4f3bf9d642	Vagabond	13	7	6
9ad226d8-8947-4c5b-95bc-d220187698c1	Banished Narrows	13	7	6
cf034ec8-ee47-43c2-b2e8-4751c22b3d4d	Houseki	13	6	7
d035fc3e-f298-4c14-9487-465be2e1dc1f	Empyrean	12	8	4
dd600260-d91c-4d77-9990-3f35873c90a1	Argyle	12	6	5
edcd4467-6846-455f-ac44-f1034476f774	Takamanohara	12	6	5
255bbe78-b191-476e-b0ae-0763c3bc2f44	Opulence	11	1	9
bb7b78ae-3468-46ce-b5ba-cca61c3a338a	High Ground	11	7	4
cd08bc7a-7ba5-4502-be87-c58b641fc94d	Salvation	11	6	3
e4bb06db-065f-4902-b93b-d8dac315eac4	Dredge	11	5	6
98a83f87-2420-48c6-93f8-cc6b62d73235	Kaiketsu	10	7	3
c5ac9f12-660e-4f1a-83e7-2e7536bbcb04	Perilous	10	6	4
df7dbf08-b8de-4ade-9d7f-1947128c9ae4	Kiken'na	10	6	3
33075df7-01c8-40e1-8b3e-1baee0054c76	Shogun	9	4	5
76043dc6-2724-45e2-9b5a-6fe2e75da588	Elevation	9	2	7
bae4df14-4f4a-424c-aac1-2f795c807146	Critical Dewpoint	9	5	3
f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42	Solitude	9	4	5
8816f240-9038-404b-bbd5-ef4f2b00f482	Ecotone	8	5	3
95b69e4b-485f-4c6c-9b00-4bd68c94c1e9	Sylvanus	8	5	3
ee43d273-8677-45c2-a8cd-aedd2c463dc9	Solution	8	7	1
98783453-ce40-4020-9e87-62099a290b62	Smallhalla	5	3	2
4a5e5612-2b2e-4375-a0b3-9335a68815f3	Solitude - Ranked	4	3	1
33c0766c-ef15-48f8-b298-34aba5bff3b4	Aquarius	2	2	0
6f82df05-3b24-4746-8579-e0c5ef9e9d64	Deadlock	2	1	1
76043dc6-2724-45e2-9b5a-6fe2e75da588	(nom vide)	2	1	1
87c03bfd-2db3-4a5b-bbf9-d5369c5894d1	(nom vide)	2	0	2
88d45250-97dd-4a28-8fb2-b52baaeebb39	Pharaoh	2	0	0
8be179f7-8940-4868-b881-44cad1ca8711	Corpo	2	0	1
9e821f5e-042f-407c-97f3-de165b1cdb26	(nom vide)	2	1	1
b6aca0c7-8ba7-4066-bf91-693571374c3c	Live Fire	2	0	2
d035fc3e-f298-4c14-9487-465be2e1dc1f	(nom vide)	2	1	1
d5c5eb4f-0dcb-4677-a866-eae0dcbfde9b	Insolence	2	1	0
e8d56863-9ad4-4efe-9059-81270884589c	(nom vide)	2	1	1
f0a1760f-0d4a-4bcc-ac7a-e8f9aee331dc	Streets	2	0	0
01af558d-53ab-4f05-ba68-92d805fc6260	01af558d-53ab-4f05-ba68-92d805fc6260	1	0	1
01af558d-53ab-4f05-ba68-92d805fc6260	(nom vide)	1	0	1
0d849a52-fedb-4aea-b5a3-caee268f1f49	Fragmentation Heavies	1	1	0
105f5d84-8de1-4908-af3a-1c4f3bf9d642	(nom vide)	1	0	1
1a6cfc2e-ec86-48e1-9464-1ce1bff6ed48	Lattice - Ranked	1	1	0
1ede38fa-4d30-4dfa-a8b7-5d08bf4e46e3	Fortitude	1	1	0
2890782c-0a33-4f2c-a468-e3a7d6cd6db4	(nom vide)	1	1	0
298d5036-cd43-47b3-a4bd-31e127566593	Bazaar	1	0	1
2c9f3490-6be2-4d90-9015-02095651e91e	Command	1	0	0
2fdb8370-e5ac-4a1a-bdce-a08bc738b9ad	(nom vide)	1	0	1
336b5174-3579-4fd8-b2f0-922e4a5f7628	Recharge - Ranked	1	0	1
3e1e4cec-4f2c-44c6-b8d2-96b85c66c702	(nom vide)	1	0	1
41217472-3020-4bd8-bce9-b2a2b0d50896	Refuge	1	1	0
46a8319c-2c63-46ee-9382-788906dcb049	Origin - Ranked	1	0	1
525451ca-0bfa-4b5c-8a0f-29524e0f2834	Disciple	1	0	0
5324364b-39a8-4f93-96a6-b80a1f18ce8a	(nom vide)	1	0	1
5646ce03-a86e-40d7-9cac-08d442d32606	5646ce03-a86e-40d7-9cac-08d442d32606	1	0	0
63d634be-0319-489d-8c21-9c4e012f664f	(nom vide)	1	1	0
6439625e-277b-4da9-9502-eefedb186ba8	Houseki	1	0	0
648ae7aa-c5d0-4f80-861a-79eb30440fcb	(nom vide)	1	0	1
6c01f693-c968-4a71-b157-efc35ffcf71f	(nom vide)	1	1	0
6dbd1c0d-a6c2-4697-8453-f0799d941741	Nadair	1	0	1
7097bc4f-efcf-4c5a-a96e-4ddb03e84d2a	Flood Gulch	1	0	0
7a9265af-a880-487b-8829-68d88fcfb145	(nom vide)	1	0	1
8420410b-044d-44d7-80b6-98a766c8c39f	Recharge	1	1	0
8f51ccb9-7dc8-4bfb-8fca-6d84d0101ac0	Shogun	1	0	1
944396dd-5661-4a16-b1d8-a6053f762c55	944396dd-5661-4a16-b1d8-a6053f762c55	1	1	0
9c7b0b0f-e933-4c2d-9d4a-3e4500d0de99	(nom vide)	1	0	1
b302eb62-da9a-480b-a409-3c89df8c1a04	b302eb62-da9a-480b-a409-3c89df8c1a04	1	1	0
bae4df14-4f4a-424c-aac1-2f795c807146	bae4df14-4f4a-424c-aac1-2f795c807146	1	1	0
be848f91-3d87-4b80-8eb9-df3b52cb8d10	Urban Raid	1	1	0
c494ef7c-d203-42a9-9c0f-b3f576334501	Highpower	1	0	0
c5ac9f12-660e-4f1a-83e7-2e7536bbcb04	(nom vide)	1	0	1
df7dbf08-b8de-4ade-9d7f-1947128c9ae4	(nom vide)	1	1	0
df7dbf08-b8de-4ade-9d7f-1947128c9ae4	df7dbf08-b8de-4ade-9d7f-1947128c9ae4	1	0	1
e23ea388-9bcb-4180-a0dc-fbe987751b9e	Streets - Ranked	1	1	0
e8d56863-9ad4-4efe-9059-81270884589c	e8d56863-9ad4-4efe-9059-81270884589c	1	0	1
e9a5a982-6c4e-4db6-9383-7b03671460eb	(nom vide)	1	1	0
f459867d-7457-4397-a332-dbbb6812792a	Ronin	1	1	0
f7e8cde9-0c0a-487c-94a3-61bfa0f20465	(nom vide)	1	1	0
(103 rows)
```
103 rangées pour 77 map_id distincts : la grille groupe par (map_id, map_name), 23 map_id sont coupés en plusieurs rangées et
20 des 61 rangées sous le plancher ont un nom vide (l'app y affiche le map_id ; la maquette ajoute le nom porté par une autre
rangée, « nom vide au registre »). Rangées ouvrables hors des 14 : 28.

## Zones nommées (v2) — `data/titles/halo_infinite/reference/map_callouts.json`

`maps.ctf_illusion.zones` : 63 zones, 58 avec polygone (5 sans polygone ignorées) ; `maps.ctf_bazaar.zones` : 29 zones,
toutes avec polygone. Polygones en mètres monde, tranche `[z_bottom ; z_top]`.
Règle (`raster_lib.zoneOf`) : point = centre de la cellule ; zones dont le polygone contient le point ET dont la tranche
contient le z médian des événements de la cellule (z de la victime pour une face « mort », du tueur pour une face « frag ») ;
à plusieurs, la plus fréquente par événement (z de chaque événement dans la tranche), puis la tranche la plus étroite ; à
aucune, le bord de polygone le plus proche à moins de 2 m parmi les tranches compatibles, puis parmi toutes ; sinon
« Zone sans nom ». Résolutions des zones les plus chaudes : lignes « ZONE » des calculs ci-dessous.
Comparaison avec la règle du rejeu 2D (`calloutsLayer.zoneAt`, centre 3D le plus proche) : Illusion (−13, 5, z 2,9) donne
« Nid blindé » à 2,99 m (même nom) ; Bazaar (−7, −1, z 3,18) donne « Porte ouest » à 2,36 m (autre nom).

## Q13 — Mode, score et date de chaque match (mini-tuiles « Rejeu », v4)
```sql
SELECT r.match_id, COALESCE(r.pair_name,'') AS pair_name, COALESCE(r.pair_name_fr,'') AS pair_name_fr, COALESCE(r.playlist_name,'') AS playlist_name, COALESCE(r.playlist_name_fr,'') AS playlist_name_fr, COALESCE(r.game_variant_name,'') AS gv, r.mode_category, r.team_0_score, r.team_1_score, r.team_0_rounds_won, r.team_1_rounds_won, r.rounds_total, strftime(timezone('Europe/Paris', r.start_time_utc), '%d/%m/%Y · %H:%M') AS quand FROM match_registry r WHERE r.match_id IN (<liste des match_id de la carte : ex/{illu,baz}_ids.txt>) ORDER BY r.start_time_utc
```
Illusion (`ex/illu_matchinfo.tsv`) :
```
match_id	pair_name	pair_name_fr	playlist_name	playlist_name_fr	gv	mode_category	team_0_score	team_1_score	team_0_rounds_won	team_1_rounds_won	rounds_total	quand
05fffb2a-50db-4fb2-b0c6-0dda57e3d44f	Tactical:Slayer on Illusion - Forge		Quick Play		Tactical:Slayer	Assassin	50	42	NULL	NULL	NULL	20/10/2025 · 21:59
6d49207d-e39a-4cdc-99b3-605c5c6eebca	Arena:Strongholds on Illusion		Quick Play		Arena:Strongholds	Assassin	200	158	NULL	NULL	NULL	23/10/2025 · 22:07
d2b74083-5dcd-4d41-871f-6096e831040e	Arena:King of the Hill on Illusion		Quick Play		KOTH:Arena	Assassin	3	0	NULL	NULL	NULL	31/10/2025 · 16:46
18ad5c88-6c49-4882-8c99-76e836e80b60	Arena:Slayer on Illusion - Forge		Quick Play		Slayer:Arena	Assassin	45	50	NULL	NULL	NULL	03/11/2025 · 20:48
415f2c6c-e882-428a-baf2-56570df3effb	Arena:Strongholds on Illusion		Quick Play		Strongholds:Arena	Assassin	166	200	NULL	NULL	NULL	13/11/2025 · 22:21
a1f20336-40f8-4ac0-bc49-c6241821a207	Arena:Slayer on Illusion - Forge		Quick Play		Slayer:Arena	Assassin	37	50	NULL	NULL	NULL	29/11/2025 · 20:37
652907bb-7bf2-4539-9a03-b610d73ab265	Arena:Team Slayer on Illusion - Forge		Quick Play		Team Slayer:Arena	Assassin	39	50	NULL	NULL	NULL	29/11/2025 · 20:59
60ec0dfb-164c-4833-b2b3-19bf999fd247	Arena:Team Slayer on Illusion - Forge		Quick Play		Team Slayer:Arena	Assassin	50	48	NULL	NULL	NULL	01/12/2025 · 21:03
795c7f1e-e24f-4eae-ad13-dc5771fd0b1d	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	48	50	NULL	NULL	NULL	20/12/2025 · 11:37
e13b44cc-494e-45a5-a3cd-f73f6b10a325	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	45	NULL	NULL	NULL	22/12/2025 · 17:26
ea24327b-f507-49d1-9a7b-b0ba5f5dedc4	Arena:Team Slayer on Illusion - Forge		Quick Play		Team Slayer:Arena	Assassin	46	50	NULL	NULL	NULL	04/01/2026 · 20:04
a92bab93-6015-40d8-a42a-7039c16a52d1	Arena:King of the Hill on Illusion		Quick Play		KOTH:Arena	Assassin	0	3	NULL	NULL	NULL	18/01/2026 · 16:23
f6a9d127-0114-4fd5-bb09-d51412f5e4de	Arena:Strongholds on Illusion		Quick Play		Strongholds:Arena	Assassin	200	189	NULL	NULL	NULL	21/01/2026 · 21:35
9126ddc0-5a18-461b-bc11-3eef34fa8a5d	Arena:Slayer on Illusion - Forge		Quick Play		Slayer:Arena	Assassin	38	50	NULL	NULL	NULL	24/01/2026 · 19:11
153a82d6-d26d-4ed8-9bf2-d5af28d79013	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	37	NULL	NULL	NULL	24/01/2026 · 19:24
d39a5435-621b-48fe-9432-949bdd0963b0	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	40	50	NULL	NULL	NULL	28/01/2026 · 18:19
e99e9ee1-502b-4e02-b1b4-31563a771bfe	Arena:CTF on Illusion		Quick Play		CTF:Arena	Other	3	2	NULL	NULL	NULL	05/02/2026 · 16:16
cf3a088b-a570-4f12-9ec4-13b0f930ae2f	Arena:Team Slayer on Illusion - Forge		Quick Play		Team Slayer:Arena	Assassin	44	50	NULL	NULL	NULL	06/02/2026 · 19:05
7ff4271a-6f9f-4554-9c3e-4726b01c52fe	Arena:CTF on Illusion		Quick Play		CTF:Arena	Assassin	3	2	NULL	NULL	NULL	07/02/2026 · 15:30
a2a023cf-9be4-4938-bd97-eab856bfd59f	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	45	NULL	NULL	NULL	08/02/2026 · 21:09
1b6f7a27-7fd2-4813-9167-70cba83d217b	Arena:CTF on Illusion		Quick Play		CTF:Arena	Assassin	3	0	NULL	NULL	NULL	10/02/2026 · 21:35
ae9c32df-7f03-49eb-bec2-fdd717dc1ab3	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	47	NULL	NULL	NULL	11/02/2026 · 16:50
eb109229-456d-4153-a47a-7fa9dc676855	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	47	NULL	NULL	NULL	11/02/2026 · 17:46
65bab728-c1e7-405b-a7d8-6de6f506fd03	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	46	50	NULL	NULL	NULL	11/02/2026 · 18:11
07aa428d-ea13-4862-bcc0-58471610df4e	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	41	NULL	NULL	NULL	11/02/2026 · 20:27
fb71123d-99f6-4bf0-978f-1129c8691175	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	33	50	NULL	NULL	NULL	12/02/2026 · 17:15
3523df2b-1e7a-4269-a8dd-da2408515949	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	36	50	NULL	NULL	NULL	15/02/2026 · 18:16
ec62cb21-3307-4e4e-a9bf-8976d751654f	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	40	50	NULL	NULL	NULL	15/02/2026 · 20:30
ac041ffa-0b91-404b-af41-fc75f6e0f898	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	46	NULL	NULL	NULL	16/02/2026 · 22:01
e869bcdf-a3db-4bff-b1b0-1eabe00befff	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	42	NULL	NULL	NULL	17/02/2026 · 22:19
f7270782-01c2-47c8-8f13-7dfad815b5fb	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	47	50	NULL	NULL	NULL	18/02/2026 · 17:34
963e8c72-cffa-40fe-90d3-a643c16163c3	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	35	NULL	NULL	NULL	27/02/2026 · 19:34
4f1639b2-a433-4f86-b5b8-0d9304d9e13b	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	49	50	NULL	NULL	NULL	05/03/2026 · 20:01
05ebe184-07af-461e-976a-50062fe495e2	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	27	50	NULL	NULL	NULL	05/03/2026 · 20:50
549d808e-d735-4443-819e-35fd772e849c	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	32	NULL	NULL	NULL	15/03/2026 · 21:34
2acc306f-da41-41e3-8dfd-9e12ab1c9926	Arena:CTF on Illusion		Quick Play		CTF:Arena	Assassin	0	3	NULL	NULL	NULL	18/03/2026 · 21:58
386c917b-aa20-479f-a7a6-6296977355ec	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	47	NULL	NULL	NULL	21/03/2026 · 20:24
49e6248f-f5ab-4e29-8a1b-74f3c4b64d10	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	49	50	NULL	NULL	NULL	22/03/2026 · 19:41
c139818f-9378-4b24-8d9a-43f767ca8656	Arena:Strongholds on Illusion		Quick Play		Strongholds:Arena	Assassin	190	200	NULL	NULL	NULL	31/03/2026 · 21:24
0727867d-ca7a-43dc-9845-bef7093a88e5	Arena:Strongholds on Illusion		Quick Play		Strongholds:Arena	Assassin	169	200	NULL	NULL	NULL	31/03/2026 · 22:20
8faf5c41-0af2-4102-b687-60b297afc1c7	Arena:CTF on Illusion		Quick Play		CTF:Arena	Assassin	1	3	NULL	NULL	NULL	31/03/2026 · 22:45
ac7ec523-1aab-4cb4-9aef-395f7e725ae6	Arena:CTF on Illusion		Quick Play		CTF:Arena	Assassin	3	2	NULL	NULL	NULL	06/04/2026 · 21:43
6dd234b2-10ff-4492-b7e0-26e53f7c9388			Quick Play		Team Slayer:Arena	Other	50	47	NULL	NULL	NULL	27/04/2026 · 21:24
e5b45563-0b6e-491e-8991-6516e01bba77			Quick Play		Strongholds:Arena	Other	200	146	NULL	NULL	NULL	27/04/2026 · 22:52
a26d3c4d-119f-42b1-879d-4e55d3412ee6	Arena:CTF on Illusion		Quick Play		CTF:Arena	Other	0	3	NULL	NULL	NULL	21/05/2026 · 22:05
cf6465d1-d0cf-4e9e-8620-cbb5c6620fed	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	32	50	NULL	NULL	NULL	23/05/2026 · 20:09
09922983-e652-4824-99fc-f2020b28b42b	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	44	NULL	NULL	NULL	25/05/2026 · 14:50
c7b37f05-cf78-4039-9cd5-e207c37f5760	Arena:Slayer on Illusion - Forge		Quick Play		Slayer:Arena	Other	50	45	NULL	NULL	NULL	26/05/2026 · 22:12
395251d2-4733-40c1-b6ff-23580fd0a17d	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	46	50	NULL	NULL	NULL	09/06/2026 · 20:52
e6b8a4e6-24c4-4c5d-9d6c-fa9f8dd040dc	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	46	NULL	NULL	NULL	28/06/2026 · 20:09
350f915f-78aa-4deb-ae40-2947a6643b5d	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	31	NULL	NULL	NULL	16/07/2026 · 21:49
fbe4ebeb-c97a-4aff-87de-919012540979	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	40	NULL	NULL	NULL	17/07/2026 · 20:50
fbef8f77-a290-4e69-8d15-0416475e73c4	Super Fiesta:Slayer on Illusion - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	37	NULL	NULL	NULL	19/07/2026 · 14:46
bf5ced1b-4efc-47d9-b0ec-050116df71f4	Arena:CTF on Illusion		Quick Play		CTF:Arena	Other	3	0	NULL	NULL	NULL	28/07/2026 · 21:59
bc60b4d9-40dc-4790-a533-3197aea9060a	Arena:CTF on Illusion		Quick Play		CTF:Arena	Other	3	0	NULL	NULL	NULL	28/07/2026 · 22:59
396cfc92-6f9e-403b-a46f-20bc73b38c11	Arena:Strongholds on Illusion		Quick Play		Strongholds:Arena	Other	200	183	1	0	1	07/09/2026 · 21:42
(56 rows)
```
Bazaar (`ex/baz_matchinfo.tsv`) :
```
match_id	pair_name	pair_name_fr	playlist_name	playlist_name_fr	gv	mode_category	team_0_score	team_1_score	team_0_rounds_won	team_1_rounds_won	rounds_total	quand
c7d40d45-c7bb-497c-bf0e-19e5c4e46139	Arena:Slayer on Bazaar - Forge		Quick Play		Arena:Slayer	Assassin	45	50	NULL	NULL	NULL	23/10/2025 · 22:17
f65627bd-313c-41f0-8f25-111b88d3276a	Tactical:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Tactical	Assassin	50	39	NULL	NULL	NULL	03/11/2025 · 21:39
e94163af-3785-46bd-9cef-8d85cc5687b3	Arena:Neutral Flag CTF on Bazaar		Quick Play		CTF:Arena Neutral Flag	Assassin	1	5	NULL	NULL	NULL	03/12/2025 · 21:57
323ec1cf-6cd1-4301-9312-170c1df45849	Arena:Neutral Flag CTF on Bazaar		Quick Play		CTF:Arena Neutral Flag	Assassin	4	5	NULL	NULL	NULL	23/12/2025 · 18:38
fb1c5d79-1789-4176-bd3f-063053a3a5e5	Arena:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena	Assassin	50	34	NULL	NULL	NULL	09/01/2026 · 20:37
735b92ac-763d-4bb9-954f-2c98a629f024	Arena:CTF on Bazaar		Quick Play		CTF:Arena	Assassin	0	3	NULL	NULL	NULL	21/01/2026 · 21:17
9903b1c5-b5e5-40d2-bd81-83d540b233cf	Arena:VIP on Bazaar		Quick Play		Arena:VIP	Assassin	6	10	NULL	NULL	NULL	22/01/2026 · 20:40
0e64e07b-2cb4-452e-a153-c7c087d2523d	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	46	NULL	NULL	NULL	26/01/2026 · 19:01
7c554aa7-2a0a-431c-85e3-4106d3fb09a4	Arena:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena	Assassin	50	37	NULL	NULL	NULL	26/01/2026 · 19:12
666c7cbc-d549-482a-bc41-8b11a45df4f6	Arena:King of the Hill on Bazaar		Quick Play		KOTH:Arena	Assassin	0	3	NULL	NULL	NULL	28/01/2026 · 18:12
189d1c23-b006-421a-9515-f978edc0dc45	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	49	50	NULL	NULL	NULL	01/02/2026 · 18:52
00761d27-487c-4d7d-ac4c-bf7584de652c	Arena:VIP on Bazaar		Quick Play		Arena:VIP	Assassin	4	10	NULL	NULL	NULL	03/02/2026 · 15:44
340584f6-1fc5-4dce-afe2-1ba96c387615	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	42	NULL	NULL	NULL	08/02/2026 · 21:22
a7a2fce3-c9da-411c-bdb7-389a541f156b	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	37	NULL	NULL	NULL	10/02/2026 · 20:44
a3af2b23-a4d3-488c-ab46-9bca3008985a	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	44	NULL	NULL	NULL	15/02/2026 · 20:14
00502e52-50cf-43dc-9c29-38e40ec3ab5a	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	45	50	NULL	NULL	NULL	17/02/2026 · 21:33
53a3768e-4619-45dc-8964-00bf968280dd	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	47	NULL	NULL	NULL	17/02/2026 · 22:07
e44bfaaa-877b-4893-99c0-aa94271bb8e9	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	47	NULL	NULL	NULL	20/02/2026 · 17:35
f1ae8345-e0a6-49b9-a0eb-3e861541bef9	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	49	NULL	NULL	NULL	05/03/2026 · 20:29
0e697aca-550b-4fc9-b6f2-df7989e65bc0	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	41	NULL	NULL	NULL	05/03/2026 · 21:09
147ffd4d-3d1d-4b90-a46d-5570009f8c36	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	36	NULL	NULL	NULL	06/03/2026 · 21:09
4e49d934-b030-414b-98de-8615551e25e1	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	49	50	NULL	NULL	NULL	11/03/2026 · 20:54
cc2035f9-e867-45dd-aa64-721ef8df32c6	Arena:Team Slayer on Bazaar - Forge		Quick Play		Team Slayer:Arena	Assassin	50	42	NULL	NULL	NULL	12/03/2026 · 22:49
c7ba5c8c-8e1d-405d-a6ab-2389f50af165	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	44	NULL	NULL	NULL	15/03/2026 · 16:45
ce46a96e-2429-4843-9836-fd96d6a6f78a	Arena:Team Slayer on Bazaar - Forge		Quick Play		Team Slayer:Arena	Assassin	39	50	NULL	NULL	NULL	16/03/2026 · 19:10
64f405f9-5959-4cc7-a5e1-afdb94d21fc8	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	44	NULL	NULL	NULL	17/03/2026 · 20:20
82f3af9f-c0fa-477b-be9b-df240d62305d	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	50	44	NULL	NULL	NULL	18/03/2026 · 19:17
de550699-ed03-4ad5-8eac-6c5a6ad8d7bb	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	42	50	NULL	NULL	NULL	21/03/2026 · 19:50
941f7f8c-87b6-4f7e-871e-5ac38bfb1202	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	41	50	NULL	NULL	NULL	21/03/2026 · 20:13
aa823fc2-17a7-4412-ba69-2fda05c8d73c	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Fiesta	41	50	NULL	NULL	NULL	28/03/2026 · 18:18
9b191a7f-72ae-417b-9789-cd0d870a43cf	Arena:Team Slayer on Bazaar - Forge		Quick Play		Team Slayer:Arena	Assassin	40	50	NULL	NULL	NULL	06/04/2026 · 23:40
6bf3090a-cc5c-4a05-a304-b54d94eb1b9c	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	44	NULL	NULL	NULL	08/04/2026 · 20:22
67c5503d-264e-46fa-aa21-51dc5829aae6			Quick Play		Team Slayer:Arena	Other	49	50	NULL	NULL	NULL	27/04/2026 · 21:35
48aa9494-1e04-4e9f-a5e2-9c75f85662ce	Arena:CTF on Bazaar		Quick Play		CTF:Arena	Other	1	0	NULL	NULL	NULL	11/05/2026 · 22:54
4d8d1644-0ee3-4acc-a473-0d264086670b	Arena:CTF on Bazaar		Quick Play		CTF:Arena	Other	3	0	NULL	NULL	NULL	21/05/2026 · 21:53
636782a0-6cd1-4b55-a769-73ed74e52dd0	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	45	NULL	NULL	NULL	24/05/2026 · 23:03
4510b5e0-37ce-4962-b7c0-554a6579a9d4	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	38	50	NULL	NULL	NULL	09/06/2026 · 20:42
a224dc60-8903-462c-bd35-a1c3230696fa	Arena:CTF on Bazaar		Quick Play		CTF:Arena	Other	2	1	NULL	NULL	NULL	09/06/2026 · 22:23
789a0aa0-481d-4a17-81b4-e88e6772a9bb	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	44	NULL	NULL	NULL	27/06/2026 · 19:12
5ceef09f-18d5-4f92-b173-7cb33eb4e9e6	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	48	NULL	NULL	NULL	28/06/2026 · 20:44
30929866-f0f3-4d05-afb0-c2c7aeb57028	Arena:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena	Other	50	34	NULL	NULL	NULL	03/07/2026 · 21:30
4d4a3961-7b69-447b-9912-5aa1242c6f09	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	28	50	NULL	NULL	NULL	16/07/2026 · 15:53
f0a0115a-26cf-4c74-ae69-54f2b708109d	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	49	50	NULL	NULL	NULL	16/07/2026 · 21:36
16bf0b50-c6af-4122-a1ba-e2ae7c1183d2	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	34	NULL	NULL	NULL	18/07/2026 · 22:19
ec422a43-993e-4451-aaf9-4145a39913f4	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	48	NULL	NULL	NULL	19/07/2026 · 15:36
ab4eef5a-e9f8-493a-a413-4c5afe2268a7	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	44	50	NULL	NULL	NULL	22/07/2026 · 17:56
c0dace44-7e12-42c9-97b4-1a9feb4c56ab	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	41	50	NULL	NULL	NULL	23/07/2026 · 17:17
cecbc053-3213-4e32-8698-02ded2a87560	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	47	50	NULL	NULL	NULL	23/07/2026 · 19:46
c7f94693-29c4-4c41-b6b2-532e47f7e920	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	35	50	NULL	NULL	NULL	28/07/2026 · 11:58
bf2a9f05-2818-4209-abb8-9e57d74ee78e	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	46	50	NULL	NULL	NULL	28/07/2026 · 20:00
9ffce8ef-a755-4b7c-a558-9e1e99664dae	Super Fiesta:Slayer on Bazaar - Forge		Quick Play		Slayer:Arena Super Fiesta	Other	50	45	NULL	NULL	NULL	29/07/2026 · 21:38
(51 rows)
```
`pair_name_fr` et `playlist_name_fr` sont vides sur les 107 matchs ; `playlist_name` vaut « Quick Play » partout. Le mode
vient de `pair_name` (partie après « : », « on <carte> » et « - Forge » retirés, comme `normalizeModeLabel` de l'app) puis
de la table des maquettes Sessions / Vue match : CTF → Drapeau, Strongholds → Bases, Team Slayer → Assassin en équipe,
Slayer → Assassin ; complétée ici : Neutral Flag CTF → Drapeau neutre, King of the Hill → Colline du roi, VIP → VIP. La
variante du préfixe (Super Fiesta : 29 matchs d'Illusion ; Tactical) n'est pas affichée, elle est dans l'infobulle. Score
= `team_0_score` / `team_1_score`, mon camp d'abord selon `match_participants.team_id`.

## Mini-tuiles « Rejeu » (v4) — sources et champs

Gamertags : `feed_killer_gamertag` (mort : « Tué par ») et `victim_gamertag` (frag : « A tué »), colonnes ajoutées à Q7.
Arme : `printf('%08x', source_tag)` traduit par `damagetag_labels.tsv` (copié de la maquette Vue match, avec son registre
d'armes, dans `armes_t.js` / `ex/damagetag_labels.tsv`) ; à défaut la catégorie `source_category` (Headshot → tir à la
tête, AttachedDamage → dégât collé, SilentMelee → assassinat, ChainedProjectile → projectile en chaîne ; None → rien).
Placement d'une mort : contexte de `match_death_context_latest` (Q8) le plus proche à ± 1 500 ms de l'instant ; « seul » si
`teammates_visible = 0`, sinon « seul · N m » à 18 m ou plus, « près · N m » en deçà (distance tronquée au mètre) ; aucun
badge sans contexte ni pour un frag.

Sur tous les événements localisés : Illusion 4 371 — tueur et victime toujours nommés, 17 sans arme ni catégorie, 2 nommés
par la seule catégorie, 4 morts sans contexte à ± 1,5 s ; Bazaar 3 920 — gamertags toujours présents, 17 sans arme ni
catégorie, 6 par la catégorie, 0 sans contexte. Aucun match sans mode, score ou date.
Champs par ligne des zones les plus chaudes (Morts, Moi) : lignes « Lignes « Rejeu » » des calculs ci-dessous — les 11 lignes
d'Illusion et les 9 de Bazaar ont tous leurs champs (mode, score, issue, date, instant, tueur, arme, placement).

## Calculs (compute_t.js + raster_lib.js, le même code que la page)

Règles : grille de 2 m ancrée sur l'origine du monde (`col = floor(x / 2)`, `lig = floor(y / 2)`) ; plancher de 3 matchs
distincts par cellule (par côté pour Victoires − défaites, sur l'union pour le Solde) ; valeur = événements / matchs mesurés
(53 Illusion, 48 Bazaar ; Victoires − défaites : chaque côté sur ses victoires / défaites mesurées) ; échelle p50 → p95 des
cellules peintes (interpolation linéaire, `analysis/tactical/quantile.go`), symétrique sur |valeur| pour les lectures
signées ; peinture = `heatRamp` / `heatRampDivergent` (64 paliers) et réindexation des cellules sur le cadre du fond
(`grilleDuPlan`). « points » = événements de la lecture ; « candidates » = cellules touchées ; « cellules » = au-dessus
du plancher ; « plages » = rectangles après fusion des voisines de même palier.
```

=== Illusion ===
{
 "perimetre": 54,
 "horsPerimetre": [
  "6dd234b2-10ff-4492-b7e0-26e53f7c9388 2026-04-27 21:24 Team Slayer:Arena",
  "e5b45563-0b6e-491e-8991-6516e01bba77 2026-04-27 22:52 Strongholds:Arena"
 ],
 "mesuresPositions": 53,
 "mesuresJournal": 53,
 "nbV": 29,
 "nbD": 24,
 "positions": 4371,
 "ctxJoint": "4367 / 4371",
 "mortsMoiJournal": 619,
 "mortsMoiLocalisees": 594,
 "fragsMoiJournal": 524,
 "fragsMoiLocalises": 469,
 "mortsSeulMoi": 107,
 "mortsAccompMoi": 487,
 "mortsSansCtxMoi": 0,
 "cotes": "469/594 1657/1651 2245/2126 0/0",
 "avant": {
  "filtres": 54,
  "retenus": 53,
  "isolement": {
   "taux": 0.18013468013468015,
   "brut": 107,
   "n": 594,
   "equipeATerre": 0
  },
  "riposte": {
   "taux": 0.16962843295638125,
   "brut": 105,
   "n": 619
  },
  "mediane": 9.96,
  "bins": [
   {
    "min": 0,
    "max": 10,
    "n": 292
   },
   {
    "min": 10,
    "max": 20,
    "n": 217
   },
   {
    "min": 20,
    "max": 30,
    "n": 60
   },
   {
    "min": 30,
    "max": 40,
    "n": 9
   },
   {
    "min": 40,
    "max": 50,
    "n": 0
   },
   {
    "min": 50,
    "max": null,
    "n": 0
   }
  ],
  "sansDistance": 16,
  "matchsMesures": 53
 }
}
morts/moi: points=594 candidates=208 cellules=88 peintes=88 hors_cadre=0 plages=77 p50=0.075 p95=0.163 max=0.208 chaude=(-7,2) v=0.208 brut=11 m=11
morts/adv: points=2126 candidates=273 cellules=225 peintes=225 hors_cadre=0 plages=162 p50=0.151 p95=0.358 max=0.528 chaude=(-3,0) v=0.528 brut=28 m=13
frags/moi: points=469 candidates=196 cellules=64 peintes=64 hors_cadre=0 plages=61 p50=0.075 p95=0.132 max=0.151 chaude=(0,8) v=0.151 brut=8 m=8
frags/adv: points=2245 candidates=269 cellules=223 peintes=223 hors_cadre=0 plages=160 p50=0.151 p95=0.413 max=0.604 chaude=(-1,-1) v=0.604 brut=32 m=23
solde/moi: points=1063 candidates=242 cellules=150 peintes=150 hors_cadre=0 plages=124 p50=0.038 p95=0.094 max=0.151 chaude=(-3,-6) v=-0.151 brut=-8 m=9
solde/adv: points=4371 candidates=282 cellules=253 peintes=253 hors_cadre=0 plages=215 p50=0.038 p95=0.17 max=0.245 chaude=(-1,0) v=0.245 brut=13 m=31
vd/moi: points=1063 candidates=242 cellules=37 peintes=37 hors_cadre=0 plages=37 p50=0.063 p95=0.186 max=0.21 chaude=(-1,-7) v=-0.21 brut=-4 m=9
vd/adv: points=4371 candidates=282 cellules=189 peintes=189 hors_cadre=0 plages=186 p50=0.114 p95=0.394 max=0.754 chaude=(-1,0) v=-0.754 brut=-15 m=31
seul/moi: points=107 candidates=80 cellules=3 peintes=3 hors_cadre=0 plages=3 p50=0.057 p95=0.074 max=0.075 chaude=(6,1) v=0.075 brut=4 m=3
seul/adv: points=356 candidates=187 cellules=41 peintes=41 hors_cadre=0 plages=39 p50=0.075 p95=0.094 max=0.151 chaude=(2,-11) v=0.151 brut=8 m=7
ZONE morts/moi (-7,2) x -14..-12 y 4..6 : Nid blindé — hors polygone, bord le plus proche à 0,81 m (tranche compatible) — z médian 2.90 m (11 événements, polygones contenant le centre : 0)
ZONE morts/adv (-3,0) x -6..-4 y 0..2 : Intersection blindée — polygone contenant le centre, z médian hors de sa tranche (0,1 à 3,6 m) — z médian 0.00 m (28 événements, polygones contenant le centre : 2)
ZONE frags/moi (0,8) x 0..2 y 16..18 : Pont blindé — polygone et tranche de z — z médian 1.00 m (8 événements, polygones contenant le centre : 1)
ZONE frags/adv (-1,-1) x -2..0 y -2..0 : Salle de test — polygone et tranche de z — z médian 0.00 m (32 événements, polygones contenant le centre : 1)
ZONE solde/moi (-3,-6) x -6..-4 y -12..-10 : Plateforme camouflée — polygone et tranche de z — z médian 2.20 m (10 événements, polygones contenant le centre : 2)
ZONE solde/adv (-1,0) x -2..0 y 0..2 : Salle de test — polygone et tranche de z — z médian 0.00 m (51 événements, polygones contenant le centre : 1)
ZONE vd/moi (-1,-7) x -2..0 y -14..-12 : Relais camouflé — polygone et tranche de z — z médian 0.55 m (16 événements, polygones contenant le centre : 1)
ZONE vd/adv (-1,0) x -2..0 y 0..2 : Salle de test — polygone et tranche de z — z médian 0.00 m (51 événements, polygones contenant le centre : 1)
ZONE seul/moi (6,1) x 12..14 y 2..4 : Canon blindé — polygone et tranche de z — z médian 1.00 m (4 événements, polygones contenant le centre : 1)
ZONE seul/adv (2,-11) x 4..6 y -22..-20 : Grenier camouflé — polygone et tranche de z — z médian 3.30 m (8 événements, polygones contenant le centre : 1)
Zone la plus chaude (Morts, Moi) : col -7 lig 2 (x -14..-12 m, y 4..6 m)
Lignes « Rejeu » (zone la plus chaude, Morts, Moi) — champs trouvés / manquants :
  Assassin 31 – 50 Défaite | 16/07/2026 · 21:49 | 05:07 | Tué par Jaeger Akagy · M41 SPNKr | près · 7 m | manquants : aucun | brut : gv Slayer:Arena Super Fiesta, tag 7
  Assassin 50 – 46 Victoire | 09/06/2026 · 20:52 | 06:22 | Tué par PBoysen · Rayon de Sentinelle | près · 14 m | manquants : aucun | brut : gv Slayer:Arena Super Fiesta, tag 11
  Assassin 50 – 27 Victoire | 05/03/2026 · 20:50 | 04:13 | Tué par NotTauro3005 · MK50 Sidekick | seul · 19 m | manquants : aucun | brut : gv Slayer:Arena Super Fiesta, tag 4
  Assassin 35 – 50 Défaite | 27/02/2026 · 19:34 | 06:53 | Tué par Frankgriga9445 · Marteau antigravité | près · 1 m | manquants : aucun | brut : gv Slayer:Arena Super Fiesta, tag 9
  Assassin 42 – 50 Défaite | 17/02/2026 · 22:19 | 05:58 | Tué par valsked · M41 SPNKr | seul · 20 m | manquants : aucun | brut : gv Slayer:Arena Super Fiesta, tag 7
  Assassin 46 – 50 Défaite | 11/02/2026 · 18:11 | 02:23 | Tué par Atomic Toast519 · Marteau antigravité | près · 12 m | manquants : aucun | brut : gv Slayer:Arena Super Fiesta, tag 9
  Assassin en équipe 50 – 44 Victoire | 06/02/2026 · 19:05 | 01:57 | Tué par TNTMaster17 · Mêlée | près · 9 m | manquants : aucun | brut : gv Team Slayer:Arena, tag 0
  Assassin 50 – 38 Victoire | 24/01/2026 · 19:11 | 09:05 | Tué par devildawg2312 · Mutilateur | près · 11 m | manquants : aucun | brut : gv Slayer:Arena, tag 19
  Assassin 50 – 45 Victoire | 22/12/2025 · 17:26 | 07:46 | Tué par Stay Classy · Marteau antigravité | près · 17 m | manquants : aucun | brut : gv Slayer:Arena Super Fiesta, tag 9
  Assassin en équipe 50 – 39 Victoire | 29/11/2025 · 20:59 | 04:10 | Tué par xACcommando9x · BR75 | près · 16 m | manquants : aucun | brut : gv Team Slayer:Arena, tag 18
  Assassin 50 – 45 Victoire | 03/11/2025 · 20:48 | 00:57 | Tué par MiniKumaKikai · Bandit EVO | près · 6 m | manquants : aucun | brut : gv Slayer:Arena, tag 27
  2025-11-03 20:48 · Victoire · 00:57 · 18ad5c88-6c49-4882-8c99-76e836e80b60 · Slayer:Arena · accompagné
  2025-11-29 20:59 · Victoire · 04:10 · 652907bb-7bf2-4539-9a03-b610d73ab265 · Team Slayer:Arena · accompagné
  2025-12-22 17:26 · Victoire · 07:46 · e13b44cc-494e-45a5-a3cd-f73f6b10a325 · Slayer:Arena Super Fiesta · accompagné
  2026-01-24 19:11 · Victoire · 09:05 · 9126ddc0-5a18-461b-bc11-3eef34fa8a5d · Slayer:Arena · accompagné
  2026-02-06 19:05 · Victoire · 01:57 · cf3a088b-a570-4f12-9ec4-13b0f930ae2f · Team Slayer:Arena · accompagné
  2026-02-11 18:11 · Défaite · 02:23 · 65bab728-c1e7-405b-a7d8-6de6f506fd03 · Slayer:Arena Super Fiesta · accompagné
  2026-02-17 22:19 · Défaite · 05:58 · e869bcdf-a3db-4bff-b1b0-1eabe00befff · Slayer:Arena Super Fiesta · seul
  2026-02-27 19:34 · Défaite · 06:53 · 963e8c72-cffa-40fe-90d3-a643c16163c3 · Slayer:Arena Super Fiesta · accompagné
  2026-03-05 20:50 · Victoire · 04:13 · 05ebe184-07af-461e-976a-50062fe495e2 · Slayer:Arena Super Fiesta · seul
  2026-06-09 20:52 · Victoire · 06:22 · 395251d2-4733-40c1-b6ff-23580fd0a17d · Slayer:Arena Super Fiesta · accompagné
  2026-07-16 21:49 · Défaite · 05:07 · 350f915f-78aa-4deb-ae40-2947a6643b5d · Slayer:Arena Super Fiesta · accompagné

=== Bazaar ===
{
 "perimetre": 50,
 "horsPerimetre": [
  "67c5503d-264e-46fa-aa21-51dc5829aae6 2026-04-27 21:35 Team Slayer:Arena"
 ],
 "mesuresPositions": 48,
 "mesuresJournal": 48,
 "nbV": 22,
 "nbD": 24,
 "positions": 3920,
 "ctxJoint": "3920 / 3920",
 "mortsMoiJournal": 483,
 "mortsMoiLocalisees": 440,
 "fragsMoiJournal": 482,
 "fragsMoiLocalises": 447,
 "mortsSeulMoi": 66,
 "mortsAccompMoi": 374,
 "mortsSansCtxMoi": 0,
 "cotes": "447/440 1500/1533 1973/1947 0/0",
 "avant": {
  "filtres": 50,
  "retenus": 48,
  "isolement": {
   "taux": 0.15,
   "brut": 66,
   "n": 440,
   "equipeATerre": 0
  },
  "riposte": {
   "taux": 0.16216216216216217,
   "brut": 78,
   "n": 481
  },
  "mediane": 9.05,
  "bins": [
   {
    "min": 0,
    "max": 10,
    "n": 239
   },
   {
    "min": 10,
    "max": 20,
    "n": 154
   },
   {
    "min": 20,
    "max": 30,
    "n": 30
   },
   {
    "min": 30,
    "max": 40,
    "n": 8
   },
   {
    "min": 40,
    "max": 50,
    "n": 0
   },
   {
    "min": 50,
    "max": null,
    "n": 0
   }
  ],
  "sansDistance": 9,
  "matchsMesures": 47
 }
}
morts/moi: points=440 candidates=205 cellules=57 peintes=57 hors_cadre=0 plages=52 p50=0.063 p95=0.146 max=0.188 chaude=(-4,-1) v=0.188 brut=9 m=6
morts/adv: points=1947 candidates=267 cellules=208 peintes=208 hors_cadre=0 plages=153 p50=0.167 p95=0.417 max=0.708 chaude=(-4,-1) v=0.708 brut=34 m=27
frags/moi: points=447 candidates=198 cellules=59 peintes=59 hors_cadre=0 plages=53 p50=0.083 p95=0.125 max=0.229 chaude=(-4,-1) v=0.229 brut=11 m=11
frags/adv: points=1973 candidates=257 cellules=204 peintes=204 hors_cadre=0 plages=150 p50=0.167 p95=0.396 max=0.875 chaude=(-4,-1) v=0.875 brut=42 m=26
solde/moi: points=887 candidates=246 cellules=131 peintes=131 hors_cadre=0 plages=104 p50=0.042 p95=0.083 max=0.125 chaude=(3,5) v=-0.125 brut=-6 m=8
solde/adv: points=3920 candidates=277 cellules=240 peintes=240 hors_cadre=0 plages=200 p50=0.042 p95=0.146 max=0.229 chaude=(3,-2) v=0.229 brut=11 m=25
vd/moi: points=887 candidates=246 cellules=20 peintes=20 hors_cadre=0 plages=20 p50=0.057 p95=0.182 max=0.193 chaude=(3,5) v=0.193 brut=4 m=8
vd/adv: points=3920 candidates=277 cellules=171 peintes=171 hors_cadre=0 plages=168 p50=0.117 p95=0.371 max=0.511 chaude=(7,0) v=0.511 brut=11 m=13
seul/moi: points=66 candidates=55 cellules=1 peintes=1 hors_cadre=0 plages=1 p50=0.063 p95=0.063 max=0.063 chaude=(8,-4) v=0.063 brut=3 m=3
seul/adv: points=305 candidates=171 cellules=33 peintes=33 hors_cadre=0 plages=32 p50=0.063 p95=0.092 max=0.125 chaude=(-3,5) v=0.125 brut=6 m=6
ZONE morts/moi (-4,-1) x -8..-6 y -2..0 : Grande cour ouest — 2 zones empilées, la plus fréquente par événement (Grande cour ouest 9, Pont du marché ouest 7) — z médian 3.18 m (9 événements, polygones contenant le centre : 3)
ZONE morts/adv (-4,-1) x -8..-6 y -2..0 : Grande cour ouest — 2 zones empilées, la plus fréquente par événement (Grande cour ouest 34, Pont du marché ouest 22) — z médian 3.18 m (34 événements, polygones contenant le centre : 3)
ZONE frags/moi (-4,-1) x -8..-6 y -2..0 : Grande cour ouest — 2 zones empilées, la plus fréquente par événement (Grande cour ouest 11, Pont du marché ouest 8) — z médian 3.18 m (11 événements, polygones contenant le centre : 3)
ZONE frags/adv (-4,-1) x -8..-6 y -2..0 : Grande cour ouest — 2 zones empilées, la plus fréquente par événement (Grande cour ouest 42, Pont du marché ouest 33) — z médian 3.18 m (42 événements, polygones contenant le centre : 3)
ZONE solde/moi (3,5) x 6..8 y 10..12 : Bibliothèque — polygone et tranche de z — z médian 2.80 m (10 événements, polygones contenant le centre : 1)
ZONE solde/adv (3,-2) x 6..8 y -4..-2 : Tanière — polygone et tranche de z — z médian 3.20 m (39 événements, polygones contenant le centre : 2)
ZONE vd/moi (3,5) x 6..8 y 10..12 : Bibliothèque — polygone et tranche de z — z médian 2.80 m (10 événements, polygones contenant le centre : 1)
ZONE vd/adv (7,0) x 14..16 y 0..2 : Grande cour est — polygone et tranche de z — z médian 0.89 m (17 événements, polygones contenant le centre : 1)
ZONE seul/moi (8,-4) x 16..18 y -8..-6 : Grande cour est — polygone et tranche de z — z médian 0.53 m (3 événements, polygones contenant le centre : 1)
ZONE seul/adv (-3,5) x -6..-4 y 10..12 : Casiers — hors polygone, bord le plus proche à 0,17 m (tranche compatible) — z médian 2.80 m (6 événements, polygones contenant le centre : 0)
Zone la plus chaude (Morts, Moi) : col -4 lig -1 (x -8..-6 m, y -2..0 m)
Lignes « Rejeu » (zone la plus chaude, Morts, Moi) — champs trouvés / manquants :
  Assassin 50 – 41 Victoire | 21/03/2026 · 20:13 | 07:40 | Tué par HighShooter4200 · S7 Sniper | près · 14 m | manquants : aucun | brut : gv Slayer:Arena Super Fiesta, tag 5
  Assassin 49 – 50 issue 4 | 05/03/2026 · 20:29 | 00:50 | Tué par Deathstare6510 · Rayon de Sentinelle | près · 1 m | manquants : aucun | brut : gv Slayer:Arena Super Fiesta, tag 6
  Assassin 50 – 47 Victoire | 20/02/2026 · 17:35 | 01:41 | Tué par SirAvlas · Mêlée | seul · 22 m | manquants : aucun | brut : gv Slayer:Arena Super Fiesta, tag 14
  Assassin 49 – 50 Défaite | 01/02/2026 · 18:52 | 07:11 | Tué par NerdAlert5125 · Rayon de Sentinelle | près · 8 m | manquants : aucun | brut : gv Slayer:Arena Super Fiesta, tag 6
  Drapeau 0 – 3 Défaite | 21/01/2026 · 21:17 | 08:10 | Tué par Sandalman242 · Grenade frag | près · 1 m | manquants : aucun | brut : gv CTF:Arena, tag 23
  Assassin 39 – 50 Défaite | 03/11/2025 · 21:39 | 00:36 | Tué par TheeFatherD · BR75 | près · 6 m | manquants : aucun | brut : gv Slayer:Arena Tactical, tag 27
  Assassin 39 – 50 Défaite | 03/11/2025 · 21:39 | 00:55 | Tué par TheeFatherD · BR75 | près · 10 m | manquants : aucun | brut : gv Slayer:Arena Tactical, tag 27
  Assassin 39 – 50 Défaite | 03/11/2025 · 21:39 | 02:50 | Tué par Midoriya090 · BR75 | près · 0 m | manquants : aucun | brut : gv Slayer:Arena Tactical, tag 27
  Assassin 39 – 50 Défaite | 03/11/2025 · 21:39 | 05:34 | Tué par CRANEOLEGEND · BR75 | près · 10 m | manquants : aucun | brut : gv Slayer:Arena Tactical, tag 27
  2025-11-03 21:39 · Défaite · 00:36 · f65627bd-313c-41f0-8f25-111b88d3276a · Slayer:Arena Tactical · accompagné
  2025-11-03 21:39 · Défaite · 00:55 · f65627bd-313c-41f0-8f25-111b88d3276a · Slayer:Arena Tactical · accompagné
  2025-11-03 21:39 · Défaite · 02:50 · f65627bd-313c-41f0-8f25-111b88d3276a · Slayer:Arena Tactical · accompagné
  2025-11-03 21:39 · Défaite · 05:34 · f65627bd-313c-41f0-8f25-111b88d3276a · Slayer:Arena Tactical · accompagné
  2026-01-21 21:17 · Défaite · 08:10 · 735b92ac-763d-4bb9-954f-2c98a629f024 · CTF:Arena · accompagné
  2026-02-01 18:52 · Défaite · 07:11 · 189d1c23-b006-421a-9515-f978edc0dc45 · Slayer:Arena Super Fiesta · accompagné
  2026-02-20 17:35 · Victoire · 01:41 · e44bfaaa-877b-4893-99c0-aa94271bb8e9 · Slayer:Arena Super Fiesta · seul
  2026-03-05 20:29 · issue 4 · 00:50 · f1ae8345-e0a6-49b9-a0eb-3e861541bef9 · Slayer:Arena Super Fiesta · accompagné
  2026-03-21 20:13 · Victoire · 07:40 · 941f7f8c-87b6-4f7e-871e-5ac38bfb1202 · Slayer:Arena Super Fiesta · accompagné
```

## Budget de hauteur (1920 × 1080)

Navigateur et barre des tâches 130 px → fenêtre 950 px. Chrome partagé : NavL1 48 + écart 16 (AppShell `gap-4`) + `py-6`
24 + en-tête 56 (h1 32 + 4 + sous-titre 20) + 24 + bandeau de conseils 56 (`min-h-[3.5rem]`) + 24 + onglets 39 (`py-2`
+ 20 + bordure 2 + bordure de la barre 1) + 24 = 311 px. Barre de filtres 40 px (`min-h-10`).
Avant : bascule 44 + 24 + H2 40 + 24 + barre d'outils 73 + 24 + `p-3` 12 + tuiles 82 + 12 + bordure 1 + en-tête 37 + 12
→ premier pixel du plan à 760 px, 190 px de plan visibles sur 720.
Après v2 : 950 − 311 − 40 − 24 − 24 = 551 px pour les trois colonnes (dont 15 px de lignes grises « remplace » propres à la
maquette) ; la ligne « Avis » (841 caractères, 4 lignes de 12,5 px + 4 px, environ 54 px) retirée rendait 497 → 551 px.
Plan dessiné en v2 (1920 × 1080, pleine largeur, colonnes 208 / 716 / 316) : boîte de 690 × 445 px → Illusion 342 × 445,
Bazaar 618 × 445 ; largeur actuelle (colonne centrale 564, bandeau sur deux rangées) : boîte 538 × 409 → Illusion 315 × 409,
Bazaar 538 × 387.

## Hauteur de page et défilement (v3 : la page défile ; v4b : colonne « Zone sélectionnée » à 360 px)

Modèle de la page Après (`modelApres` dans la page, mêmes constantes que la feuille de style) : colonnes 208 / reste / 360 px,
écarts 12 px, trois colonnes dès 1400 px de fenêtre ; boîte du plan = min(800 px, (colonne centrale − 2 − 24 − 90) / rapport du
fond) de haut ; rampe verticale 8 px + libellés (90 px avec l'écart) ; carte du plan = 2 + bandeau (45, ou 81 sur deux rangées)
+ 10 + plan + 12 ; « Cartes jouées » 551 px ; haut de la rangée à 311 + 40 + 24 = 375 px ; marge basse 24 px. Pied de page de
l'app non compté. Navigateur et barre des tâches : 130 px (fenêtre utile 950 px en 1080, 770 px en 900). En 1440 la largeur de
l'app est la même qu'en 1920 (plafond de 1320 px).

| Vue | Plan dessiné | Carte du plan | Page | Défilement 1920 × 1080 | Défilement 1440 × 900 |
|---|---|---|---|---|---|
| Illusion, pleine largeur | 564 × 733 (largeur) | 838 | 1 252 | 302 | 482 |
| Illusion, largeur actuelle | 412 × 536 (largeur) | 641 | 1 055 | 105 | 285 |
| Bazaar, pleine largeur | 564 × 406 (largeur) | 511 | 950 | 0 | 180 |
| Bazaar, largeur actuelle | 412 × 296 (largeur) | 401 | 950 | 0 | 180 |

v3 (colonne de zone à 300 px) : Illusion pleine 615 × 800, page 1 283 px, défilement 333 / 513 px. En v4b la colonne centrale
passe à 680 px : le bandeau du plan est estimé sur deux rangées (81 px). Bazaar : la rangée vaut la hauteur de « Cartes jouées »
(551 px), plus haute que sa carte du plan ; la page fait 950 px. La mesure en direct (scrollHeight de la fenêtre émulée)
s'affiche sous le cadre quand la page est ouverte dans un navigateur.

## Longueur des deux lignes des mini-tuiles « Rejeu » (v4b)

Calcul (`largeurs_t.js`) avec les chasses de Roboto (table d'avances de Roboto Regular approchée, +2 % en 500, +4 % en 700,
Roboto Mono 0,6 em) sur le texte rendu par la page ; ce n'est pas une mesure d'écran. Place disponible : colonne 360 → tuile
358 − 16 (marges) − 6 (écart) − 36 (bouton) = 300 px de texte, environ 289 px quand la liste défile (Bazaar).
Ligne 1 = mode (12,5 px, 500) + score (12,5 px, 700) + issue (12,5 px, 500) + date (10,5 px), écarts 4 px. Ligne 2 = pastille
(Roboto Mono 11 px + 10) + « Tué par joueur » (11,5 px) + « · arme » (11,5 px) + badge (10,5 px + 12), écarts 5 px.
Contrôle demandé : « Assassin en équipe · 50 – 44 · Victoire · 06/02/2026 · 19:05 » = 297 px (tient) ; « 05:07 · Tué par
Frankgriga9445 · Marteau antigravité · près · 1 m » = 343 px (dépasse de 43 px : l'arme se tronque).
```
{
 "#apres-illusion-pleine": {
  "max1": 297,
  "max2": 357,
  "rows": [
   {
    "l1": 236,
    "l2": 292,
    "l2fixe": 222,
    "txt1": "Assassin · 31 – 50 · Défaite · 16/07/2026 · 21:49",
    "txt2": "05:07 Tué par Jaeger Akagy · M41 SPNKr près · 7 m"
   },
   {
    "l1": 239,
    "l2": 314,
    "l2fixe": 203,
    "txt1": "Assassin · 50 – 46 · Victoire · 09/06/2026 · 20:52",
    "txt2": "06:22 Tué par PBoysen · Rayon de Sentinelle près · 14 m"
   },
   {
    "l1": 239,
    "l2": 318,
    "l2fixe": 232,
    "txt1": "Assassin · 50 – 27 · Victoire · 05/03/2026 · 20:50",
    "txt2": "04:13 Tué par NotTauro3005 · MK50 Sidekick seul · 19 m"
   },
   {
    "l1": 236,
    "l2": 343,
    "l2fixe": 233,
    "txt1": "Assassin · 35 – 50 · Défaite · 27/02/2026 · 19:34",
    "txt2": "06:53 Tué par Frankgriga9445 · Marteau antigravité près · 1 m"
   },
   {
    "l1": 236,
    "l2": 266,
    "l2fixe": 196,
    "txt1": "Assassin · 42 – 50 · Défaite · 17/02/2026 · 22:19",
    "txt2": "05:58 Tué par valsked · M41 SPNKr seul · 20 m"
   },
   {
    "l1": 236,
    "l2": 357,
    "l2fixe": 247,
    "txt1": "Assassin · 46 – 50 · Défaite · 11/02/2026 · 18:11",
    "txt2": "02:23 Tué par Atomic Toast519 · Marteau antigravité près · 12 m"
   },
   {
    "l1": 297,
    "l2": 265,
    "l2fixe": 223,
    "txt1": "Assassin en équipe · 50 – 44 · Victoire · 06/02/2026 · 19:05",
    "txt2": "01:57 Tué par TNTMaster17 · Mêlée près · 9 m"
   },
   {
    "l1": 239,
    "l2": 299,
    "l2fixe": 236,
    "txt1": "Assassin · 50 – 38 · Victoire · 24/01/2026 · 19:11",
    "txt2": "09:05 Tué par devildawg2312 · Mutilateur près · 11 m"
   },
   {
    "l1": 239,
    "l2": 328,
    "l2fixe": 218,
    "txt1": "Assassin · 50 – 45 · Victoire · 22/12/2025 · 17:26",
    "txt2": "07:46 Tué par Stay Classy · Marteau antigravité près · 17 m"
   },
   {
    "l1": 297,
    "l2": 288,
    "l2fixe": 250,
    "txt1": "Assassin en équipe · 50 – 39 · Victoire · 29/11/2025 · 20:59",
    "txt2": "04:10 Tué par xACcommando9x · BR75 près · 16 m"
   },
   {
    "l1": 239,
    "l2": 298,
    "l2fixe": 229,
    "txt1": "Assassin · 50 – 45 · Victoire · 03/11/2025 · 20:48",
    "txt2": "00:57 Tué par MiniKumaKikai · Bandit EVO près · 6 m"
   }
  ]
 },
 "#apres-bazaar-pleine": {
  "max1": 246,
  "max2": 345,
  "rows": [
   {
    "l1": 239,
    "l2": 308,
    "l2fixe": 248,
    "txt1": "Assassin · 50 – 41 · Victoire · 21/03/2026 · 20:13",
    "txt2": "07:40 Tué par HighShooter4200 · S7 Sniper près · 14 m"
   },
   {
    "l1": 246,
    "l2": 345,
    "l2fixe": 234,
    "txt1": "Assassin · 49 – 50 · Abandon · 05/03/2026 · 20:29",
    "txt2": "00:50 Tué par Deathstare6510 · Rayon de Sentinelle près · 1 m"
   },
   {
    "l1": 239,
    "l2": 241,
    "l2fixe": 199,
    "txt1": "Assassin · 50 – 47 · Victoire · 20/02/2026 · 17:35",
    "txt2": "01:41 Tué par SirAvlas · Mêlée seul · 22 m"
   },
   {
    "l1": 236,
    "l2": 338,
    "l2fixe": 227,
    "txt1": "Assassin · 49 – 50 · Défaite · 01/02/2026 · 18:52",
    "txt2": "07:11 Tué par NerdAlert5125 · Rayon de Sentinelle près · 8 m"
   },
   {
    "l1": 218,
    "l2": 307,
    "l2fixe": 230,
    "txt1": "Drapeau · 0 – 3 · Défaite · 21/01/2026 · 21:17",
    "txt2": "08:10 Tué par Sandalman242 · Grenade frag près · 1 m"
   },
   {
    "l1": 236,
    "l2": 256,
    "l2fixe": 218,
    "txt1": "Assassin · 39 – 50 · Défaite · 03/11/2025 · 21:39",
    "txt2": "00:36 Tué par TheeFatherD · BR75 près · 6 m"
   },
   {
    "l1": 236,
    "l2": 262,
    "l2fixe": 224,
    "txt1": "Assassin · 39 – 50 · Défaite · 03/11/2025 · 21:39",
    "txt2": "00:55 Tué par TheeFatherD · BR75 près · 10 m"
   },
   {
    "l1": 236,
    "l2": 262,
    "l2fixe": 224,
    "txt1": "Assassin · 39 – 50 · Défaite · 03/11/2025 · 21:39",
    "txt2": "02:50 Tué par Midoriya090 · BR75 près · < 1 m"
   },
   {
    "l1": 236,
    "l2": 284,
    "l2fixe": 246,
    "txt1": "Assassin · 39 – 50 · Défaite · 03/11/2025 · 21:39",
    "txt2": "05:34 Tué par CRANEOLEGEND · BR75 près · 10 m"
   }
  ]
 },
 "controle": {
  "Assassin en équipe · 50 – 44 · Victoire · 06/02/2026 · 19:05": 297,
  "05:07 · Tué par Frankgriga9445 · Marteau antigravité · près · 1 m": 343
 }
}
```
Règle de la page : sur la ligne 2, le joueur (« Tué par … ») et le badge ne se compriment jamais ; seule l'arme, en fin, se tronque
avec une ellipse (texte complet dans l'infobulle de la tuile). Sur la ligne 1, seul le mode porte l'ellipse de secours.
