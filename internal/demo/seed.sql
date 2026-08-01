INSERT INTO countries (code, name)
VALUES ('SC', 'Seychelles')
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO leagues (id, name, slug, type, gender, country_code, metadata)
VALUES (
    '10000000-0000-0000-0000-000000000001', 'Seychelles Demo Premier League',
    'seychelles-demo-premier-league', 'league', 'men', 'SC', '{"demo":true}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name, slug = EXCLUDED.slug, type = EXCLUDED.type, gender = EXCLUDED.gender,
    country_code = EXCLUDED.country_code, metadata = EXCLUDED.metadata;

INSERT INTO seasons (id, league_id, name, starts_on, ends_on, is_current, metadata)
VALUES (
    '10000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000001',
    '2026 Demo Season', '2026-01-01', '2026-12-31', true, '{"demo":true}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    league_id = EXCLUDED.league_id, name = EXCLUDED.name, starts_on = EXCLUDED.starts_on,
    ends_on = EXCLUDED.ends_on, is_current = EXCLUDED.is_current, metadata = EXCLUDED.metadata;

INSERT INTO venues (id, name, city, country_code, capacity, surface, timezone, metadata)
VALUES (
    '10000000-0000-0000-0000-000000000003', 'Demo National Stadium', 'Victoria', 'SC',
    10000, 'grass', 'Indian/Mahe', '{"demo":true}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name, city = EXCLUDED.city, country_code = EXCLUDED.country_code,
    capacity = EXCLUDED.capacity, surface = EXCLUDED.surface, timezone = EXCLUDED.timezone,
    metadata = EXCLUDED.metadata;

INSERT INTO teams (
    id, name, short_name, code, country_code, founded_year, national, venue_id, metadata
)
VALUES
    ('20000000-0000-0000-0000-000000000001', 'Victoria United', 'Victoria', 'VIC', 'SC', 1995, false,
     '10000000-0000-0000-0000-000000000003', '{"demo":true}'::jsonb),
    ('20000000-0000-0000-0000-000000000002', 'Mahé City', 'Mahé', 'MAH', 'SC', 1998, false,
     '10000000-0000-0000-0000-000000000003', '{"demo":true}'::jsonb),
    ('20000000-0000-0000-0000-000000000003', 'Praslin Rovers', 'Praslin', 'PRA', 'SC', 2001, false,
     '10000000-0000-0000-0000-000000000003', '{"demo":true}'::jsonb),
    ('20000000-0000-0000-0000-000000000004', 'La Digue Athletic', 'La Digue', 'LDA', 'SC', 2003, false,
     '10000000-0000-0000-0000-000000000003', '{"demo":true}'::jsonb)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name, short_name = EXCLUDED.short_name, code = EXCLUDED.code,
    country_code = EXCLUDED.country_code, founded_year = EXCLUDED.founded_year,
    national = EXCLUDED.national, venue_id = EXCLUDED.venue_id, metadata = EXCLUDED.metadata;

INSERT INTO people (id, display_name, first_name, last_name, birth_date, country_code, metadata)
VALUES
    ('30000000-0000-0000-0000-000000000001', 'Alex Michel', 'Alex', 'Michel', '1998-04-12', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000002', 'Daniel Rose', 'Daniel', 'Rose', '1997-09-03', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000003', 'Marc Hoareau', 'Marc', 'Hoareau', '1999-02-21', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000004', 'Jules Payet', 'Jules', 'Payet', '2000-11-08', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000005', 'Patrick James', 'Patrick', 'James', '1976-06-17', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000006', 'Sophie Larue', 'Sophie', 'Larue', '1980-01-29', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000007', 'Jean Morel', 'Jean', 'Morel', '1985-03-14', 'SC', '{"demo":true}'::jsonb)
ON CONFLICT (id) DO UPDATE SET
    display_name = EXCLUDED.display_name, first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name, birth_date = EXCLUDED.birth_date,
    country_code = EXCLUDED.country_code, metadata = EXCLUDED.metadata;

INSERT INTO players (person_id, position, detailed_position, preferred_foot, height_cm, weight_kg)
VALUES
    ('30000000-0000-0000-0000-000000000001', 'forward', 'centre_forward', 'right', 181, 76),
    ('30000000-0000-0000-0000-000000000002', 'goalkeeper', 'goalkeeper', 'right', 188, 82),
    ('30000000-0000-0000-0000-000000000003', 'midfielder', 'central_midfield', 'left', 175, 70),
    ('30000000-0000-0000-0000-000000000004', 'defender', 'centre_back', 'right', 184, 79)
ON CONFLICT (person_id) DO UPDATE SET
    position = EXCLUDED.position, detailed_position = EXCLUDED.detailed_position,
    preferred_foot = EXCLUDED.preferred_foot, height_cm = EXCLUDED.height_cm, weight_kg = EXCLUDED.weight_kg;

INSERT INTO coaches (person_id)
VALUES ('30000000-0000-0000-0000-000000000005'), ('30000000-0000-0000-0000-000000000006')
ON CONFLICT (person_id) DO NOTHING;

INSERT INTO team_memberships (id, team_id, person_id, role, shirt_number, starts_on)
VALUES
    ('31000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001', 'player', 9, '2026-01-01'),
    ('31000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000002', 'player', 1, '2026-01-01'),
    ('31000000-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000003', '30000000-0000-0000-0000-000000000003', 'player', 8, '2026-01-01'),
    ('31000000-0000-0000-0000-000000000004', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000004', 'player', 4, '2026-01-01'),
    ('31000000-0000-0000-0000-000000000005', '20000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000005', 'head_coach', NULL, '2026-01-01'),
    ('31000000-0000-0000-0000-000000000006', '20000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000006', 'head_coach', NULL, '2026-01-01')
ON CONFLICT (id) DO UPDATE SET
    team_id = EXCLUDED.team_id, person_id = EXCLUDED.person_id, role = EXCLUDED.role,
    shirt_number = EXCLUDED.shirt_number, starts_on = EXCLUDED.starts_on, ends_on = NULL;

INSERT INTO matches (
    id, league_id, season_id, stage, round, round_sort, group_name, leg, kickoff_at,
    status, period, elapsed_minute, venue_id, home_team_id, away_team_id,
    home_score, away_score, home_half_time_score, away_half_time_score,
    home_extra_time_score, away_extra_time_score, home_penalty_score, away_penalty_score,
    attendance, winner_team_id, version, source_hash, metadata
)
VALUES
    ('40000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001',
     '10000000-0000-0000-0000-000000000002', 'regular_season', 'Round 12', 12, NULL, 1,
     '2026-08-01T12:00:00Z', 'live', 'second_half', 67, '10000000-0000-0000-0000-000000000003',
     '20000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002',
     1, 0, 1, 0, NULL, NULL, NULL, NULL, 8342, NULL, 1,
     digest('demo-live-match-1-v1', 'sha256'), '{"demo":true}'::jsonb),
    ('40000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000001',
     '10000000-0000-0000-0000-000000000002', 'regular_season', 'Round 13', 13, NULL, 1,
     '2026-08-02T14:00:00Z', 'scheduled', NULL, NULL, '10000000-0000-0000-0000-000000000003',
     '20000000-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000004',
     NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 1,
     digest('demo-scheduled-match-1-v1', 'sha256'), '{"demo":true}'::jsonb),
    ('40000000-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000001',
     '10000000-0000-0000-0000-000000000002', 'regular_season', 'Round 11', 11, NULL, 1,
     '2026-07-31T16:00:00Z', 'finished', 'full_time', 90, '10000000-0000-0000-0000-000000000003',
     '20000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000003',
     2, 1, 1, 0, NULL, NULL, NULL, NULL, 7614, '20000000-0000-0000-0000-000000000002', 1,
     digest('demo-finished-match-1-v1', 'sha256'), '{"demo":true}'::jsonb)
ON CONFLICT (id) DO UPDATE SET
    league_id = EXCLUDED.league_id, season_id = EXCLUDED.season_id, stage = EXCLUDED.stage,
    round = EXCLUDED.round, round_sort = EXCLUDED.round_sort, group_name = EXCLUDED.group_name,
    leg = EXCLUDED.leg, kickoff_at = EXCLUDED.kickoff_at, status = EXCLUDED.status,
    period = EXCLUDED.period, elapsed_minute = EXCLUDED.elapsed_minute, venue_id = EXCLUDED.venue_id,
    home_team_id = EXCLUDED.home_team_id, away_team_id = EXCLUDED.away_team_id,
    home_score = EXCLUDED.home_score, away_score = EXCLUDED.away_score,
    home_half_time_score = EXCLUDED.home_half_time_score, away_half_time_score = EXCLUDED.away_half_time_score,
    home_extra_time_score = EXCLUDED.home_extra_time_score, away_extra_time_score = EXCLUDED.away_extra_time_score,
    home_penalty_score = EXCLUDED.home_penalty_score, away_penalty_score = EXCLUDED.away_penalty_score,
    attendance = EXCLUDED.attendance, winner_team_id = EXCLUDED.winner_team_id, version = EXCLUDED.version,
    source_hash = EXCLUDED.source_hash, metadata = EXCLUDED.metadata, updated_at = now();

DELETE FROM match_events
WHERE match_id IN ('40000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000003');

INSERT INTO match_events (
    id, match_id, sequence, period, minute, type, team_id, primary_person_id,
    detail, home_score, away_score, metadata, occurred_at
)
VALUES
    ('41000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001',
     1, 'first_half', 0, 'kickoff', NULL, NULL, 'Match started', 0, 0, '{"demo":true}'::jsonb, '2026-08-01T12:00:00Z'),
    ('41000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000001',
     2, 'first_half', 34, 'goal', '20000000-0000-0000-0000-000000000001',
     '30000000-0000-0000-0000-000000000001', 'Right-footed shot', 1, 0,
     '{"demo":true}'::jsonb, '2026-08-01T12:34:00Z'),
    ('41000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000003',
     1, 'first_half', 0, 'kickoff', NULL, NULL, 'Match started', 0, 0, '{"demo":true}'::jsonb, '2026-07-31T16:00:00Z'),
    ('41000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000003',
     2, 'second_half', 90, 'full_time', NULL, NULL, 'Match finished', 2, 1,
     '{"demo":true}'::jsonb, '2026-07-31T17:50:00Z');

INSERT INTO season_teams (season_id, team_id)
SELECT '10000000-0000-0000-0000-000000000002', id
FROM teams WHERE id::text LIKE '20000000-0000-0000-0000-00000000000%'
ON CONFLICT (season_id, team_id) DO NOTHING;

INSERT INTO match_team_info (match_id, team_id, formation, coach_id, metadata)
VALUES
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', '4-3-3', '30000000-0000-0000-0000-000000000005', '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002', '4-2-3-1', '30000000-0000-0000-0000-000000000006', '{"demo":true}')
ON CONFLICT (match_id, team_id) DO UPDATE SET
    formation = EXCLUDED.formation, coach_id = EXCLUDED.coach_id, metadata = EXCLUDED.metadata;

INSERT INTO match_officials (match_id, person_id, role, metadata)
VALUES ('40000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000007', 'referee', '{"demo":true}')
ON CONFLICT (match_id, person_id, role) DO UPDATE SET metadata = EXCLUDED.metadata;

INSERT INTO player_match_statistics (
    match_id, team_id, person_id, started, minutes_played, goals, shots, shots_on_target, passes, rating, metadata
)
VALUES (
    '40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001',
    '30000000-0000-0000-0000-000000000001', true, 67, 1, 3, 2, 21, 8.10, '{"demo":true}'
)
ON CONFLICT (match_id, person_id) DO UPDATE SET
    team_id = EXCLUDED.team_id, started = EXCLUDED.started, minutes_played = EXCLUDED.minutes_played,
    goals = EXCLUDED.goals, shots = EXCLUDED.shots, shots_on_target = EXCLUDED.shots_on_target,
    passes = EXCLUDED.passes, rating = EXCLUDED.rating, metadata = EXCLUDED.metadata;

INSERT INTO standings (
    season_id, team_id, group_name, position, played, won, drawn, lost, goals_for, goals_against,
    points, form, zone, description, home_played, home_won, home_drawn, home_lost,
    away_played, away_won, away_drawn, away_lost
)
VALUES
    ('10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001', '', 1, 11, 8, 2, 1, 23, 8, 26, 'WWDWW', 'champions_league', 'Championship leader', 6, 5, 1, 0, 5, 3, 1, 1),
    ('10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000002', '', 2, 11, 7, 2, 2, 19, 10, 23, 'WLWWW', NULL, NULL, 6, 4, 1, 1, 5, 3, 1, 1),
    ('10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000003', '', 3, 11, 5, 3, 3, 16, 12, 18, 'DWLWD', NULL, NULL, 6, 3, 2, 1, 5, 2, 1, 2),
    ('10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '', 4, 11, 3, 2, 6, 11, 18, 11, 'LLWDL', 'relegation', 'Relegation zone', 6, 2, 1, 3, 5, 1, 1, 3)
ON CONFLICT (season_id, team_id, group_name) DO UPDATE SET
    position = EXCLUDED.position, played = EXCLUDED.played, won = EXCLUDED.won,
    drawn = EXCLUDED.drawn, lost = EXCLUDED.lost, goals_for = EXCLUDED.goals_for,
    goals_against = EXCLUDED.goals_against, points = EXCLUDED.points, form = EXCLUDED.form,
    zone = EXCLUDED.zone, description = EXCLUDED.description,
    home_played = EXCLUDED.home_played, home_won = EXCLUDED.home_won,
    home_drawn = EXCLUDED.home_drawn, home_lost = EXCLUDED.home_lost,
    away_played = EXCLUDED.away_played, away_won = EXCLUDED.away_won,
    away_drawn = EXCLUDED.away_drawn, away_lost = EXCLUDED.away_lost, updated_at = now();

INSERT INTO external_ids (entity_type, entity_id, provider, external_id)
VALUES
    ('league', '10000000-0000-0000-0000-000000000001', 'demo', 'seychelles-premier-demo'),
    ('season', '10000000-0000-0000-0000-000000000002', 'demo', 'season-2026'),
    ('venue', '10000000-0000-0000-0000-000000000003', 'demo', 'national-stadium-demo'),
    ('team', '20000000-0000-0000-0000-000000000001', 'demo', 'victoria-united'),
    ('team', '20000000-0000-0000-0000-000000000002', 'demo', 'mahe-city'),
    ('team', '20000000-0000-0000-0000-000000000003', 'demo', 'praslin-rovers'),
    ('team', '20000000-0000-0000-0000-000000000004', 'demo', 'la-digue-athletic'),
    ('person', '30000000-0000-0000-0000-000000000001', 'demo', 'alex-michel'),
    ('person', '30000000-0000-0000-0000-000000000002', 'demo', 'daniel-rose'),
    ('person', '30000000-0000-0000-0000-000000000003', 'demo', 'marc-hoareau'),
    ('person', '30000000-0000-0000-0000-000000000004', 'demo', 'jules-payet'),
    ('person', '30000000-0000-0000-0000-000000000005', 'demo', 'coach-james'),
    ('person', '30000000-0000-0000-0000-000000000006', 'demo', 'coach-larue'),
    ('person', '30000000-0000-0000-0000-000000000007', 'demo', 'referee-morel'),
    ('match', '40000000-0000-0000-0000-000000000001', 'demo', 'live-match-1'),
    ('match', '40000000-0000-0000-0000-000000000002', 'demo', 'scheduled-match-1'),
    ('match', '40000000-0000-0000-0000-000000000003', 'demo', 'finished-match-1'),
    ('match_event', '41000000-0000-0000-0000-000000000001', 'demo', 'live-kickoff'),
    ('match_event', '41000000-0000-0000-0000-000000000002', 'demo', 'live-goal-1'),
    ('match_event', '41000000-0000-0000-0000-000000000003', 'demo', 'finished-kickoff'),
    ('match_event', '41000000-0000-0000-0000-000000000004', 'demo', 'finished-full-time')
ON CONFLICT (provider, entity_type, external_id) DO UPDATE SET
    entity_id = EXCLUDED.entity_id, updated_at = now();
