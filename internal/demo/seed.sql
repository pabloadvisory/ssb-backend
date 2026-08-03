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

INSERT INTO venues (
    id, name, city, country_code, address, latitude, longitude, capacity, surface, timezone, metadata
)
VALUES
    (
        '10000000-0000-0000-0000-000000000003', 'Demo National Stadium', 'Victoria', 'SC',
        'Stad Popiler, Roche Caiman', -4.6362, 55.4708,
        10000, 'grass', 'Indian/Mahe', '{"demo":true}'::jsonb
    ),
    (
        '10000000-0000-0000-0000-000000000004', 'Stad Linite', NULL, 'SC',
        NULL, NULL, NULL, NULL, NULL, 'Indian/Mahe', '{"demo":true}'::jsonb
    )
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name, city = EXCLUDED.city, country_code = EXCLUDED.country_code,
    address = EXCLUDED.address, latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude,
    capacity = EXCLUDED.capacity, surface = EXCLUDED.surface, timezone = EXCLUDED.timezone,
    metadata = EXCLUDED.metadata;

INSERT INTO teams (
    id, name, short_name, code, country_code, founded_year, national, venue_id,
    logo_url, primary_color, secondary_color, metadata
)
VALUES
    ('20000000-0000-0000-0000-000000000001', 'Victoria United', 'Victoria', 'VIC', 'SC', 1995, false,
     '10000000-0000-0000-0000-000000000003', NULL, '#0057B8', '#FFFFFF', '{"demo":true}'::jsonb),
    ('20000000-0000-0000-0000-000000000002', 'Mahé City', 'Mahé', 'MAH', 'SC', 1998, false,
     '10000000-0000-0000-0000-000000000003', NULL, '#D71920', '#FFFFFF', '{"demo":true}'::jsonb),
    ('20000000-0000-0000-0000-000000000003', 'Praslin Rovers', 'Praslin', 'PRA', 'SC', 2001, false,
     '10000000-0000-0000-0000-000000000003', NULL, '#009A44', '#FFD700', '{"demo":true}'::jsonb),
    ('20000000-0000-0000-0000-000000000004', 'La Digue Athletic', 'La Digue', 'LDA', 'SC', 2003, false,
     '10000000-0000-0000-0000-000000000003', NULL, '#6A1B9A', '#FFFFFF', '{"demo":true}'::jsonb),
    ('20000000-0000-0000-0000-000000000005', 'Rovers', 'Rovers', 'ROV', 'SC', NULL, false,
     '10000000-0000-0000-0000-000000000004', 'https://ssb.ibuildnothing.com/assets/team-logos/rovers.png',
     '#F7D117', '#111111', '{"demo":true}'::jsonb),
    ('20000000-0000-0000-0000-000000000006', 'St Michel', 'St Michel', 'STM', 'SC', NULL, false,
     '10000000-0000-0000-0000-000000000004', 'https://ssb.ibuildnothing.com/assets/team-logos/st-michel.png',
     '#EF1745', '#111111', '{"demo":true}'::jsonb)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name, short_name = EXCLUDED.short_name, code = EXCLUDED.code,
    country_code = EXCLUDED.country_code, founded_year = EXCLUDED.founded_year,
    national = EXCLUDED.national, venue_id = EXCLUDED.venue_id, logo_url = EXCLUDED.logo_url,
    primary_color = EXCLUDED.primary_color, secondary_color = EXCLUDED.secondary_color,
    metadata = EXCLUDED.metadata;

CREATE TEMP TABLE demo_lineup_players (
    person_id uuid PRIMARY KEY,
    membership_id uuid NOT NULL,
    team_id uuid NOT NULL,
    display_name text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    birth_date date NOT NULL,
    position text NOT NULL,
    detailed_position text NOT NULL,
    preferred_foot text NOT NULL,
    height_cm smallint NOT NULL,
    weight_kg numeric(5,2) NOT NULL,
    shirt_number smallint NOT NULL,
    is_starter boolean NOT NULL,
    is_captain boolean NOT NULL DEFAULT false,
    grid_position text
) ON COMMIT DROP;

INSERT INTO demo_lineup_players (
    person_id, membership_id, team_id, display_name, first_name, last_name,
    birth_date, position, detailed_position, preferred_foot, height_cm, weight_kg,
    shirt_number, is_starter, is_captain, grid_position
)
VALUES
    -- Victoria United: nine additional starters and six additional substitutes.
    ('30000000-0000-0000-0000-000000000012', '31000000-0000-0000-0000-000000000012', '20000000-0000-0000-0000-000000000001', 'Yannick Savy', 'Yannick', 'Savy', '1997-02-14', 'goalkeeper', 'goalkeeper', 'right', 190, 84, 1, true, false, '1:1'),
    ('30000000-0000-0000-0000-000000000013', '31000000-0000-0000-0000-000000000013', '20000000-0000-0000-0000-000000000001', 'Kieran Labrosse', 'Kieran', 'Labrosse', '1999-06-08', 'defender', 'right_back', 'right', 178, 73, 2, true, false, '2:4'),
    ('30000000-0000-0000-0000-000000000014', '31000000-0000-0000-0000-000000000014', '20000000-0000-0000-0000-000000000001', 'Joel Camille', 'Joel', 'Camille', '1996-11-21', 'defender', 'centre_back', 'right', 186, 81, 4, true, false, '2:3'),
    ('30000000-0000-0000-0000-000000000015', '31000000-0000-0000-0000-000000000015', '20000000-0000-0000-0000-000000000001', 'Brandon Mondon', 'Brandon', 'Mondon', '1998-09-17', 'defender', 'centre_back', 'left', 184, 79, 5, true, false, '2:2'),
    ('30000000-0000-0000-0000-000000000016', '31000000-0000-0000-0000-000000000016', '20000000-0000-0000-0000-000000000001', 'Thierry Adeline', 'Thierry', 'Adeline', '2000-01-30', 'defender', 'left_back', 'left', 176, 71, 3, true, false, '2:1'),
    ('30000000-0000-0000-0000-000000000017', '31000000-0000-0000-0000-000000000017', '20000000-0000-0000-0000-000000000001', 'Nigel Freminot', 'Nigel', 'Freminot', '1998-05-12', 'midfielder', 'defensive_midfield', 'right', 181, 76, 6, true, false, '3:1'),
    ('30000000-0000-0000-0000-000000000018', '31000000-0000-0000-0000-000000000018', '20000000-0000-0000-0000-000000000001', 'Kevin Ernesta', 'Kevin', 'Ernesta', '2001-03-09', 'midfielder', 'attacking_midfield', 'left', 175, 69, 10, true, false, '3:3'),
    ('30000000-0000-0000-0000-000000000019', '31000000-0000-0000-0000-000000000019', '20000000-0000-0000-0000-000000000001', 'Daryl Valentin', 'Daryl', 'Valentin', '2002-07-25', 'forward', 'right_winger', 'right', 177, 70, 7, true, false, '4:3'),
    ('30000000-0000-0000-0000-000000000020', '31000000-0000-0000-0000-000000000020', '20000000-0000-0000-0000-000000000001', 'Ryan Alcindor', 'Ryan', 'Alcindor', '2000-10-05', 'forward', 'left_winger', 'left', 174, 68, 14, true, false, '4:1'),
    ('30000000-0000-0000-0000-000000000021', '31000000-0000-0000-0000-000000000021', '20000000-0000-0000-0000-000000000001', 'Cedric Pool', 'Cedric', 'Pool', '1995-12-18', 'goalkeeper', 'goalkeeper', 'right', 187, 82, 12, false, false, NULL),
    ('30000000-0000-0000-0000-000000000022', '31000000-0000-0000-0000-000000000022', '20000000-0000-0000-0000-000000000001', 'Andre Barbe', 'Andre', 'Barbe', '2001-04-16', 'defender', 'right_back', 'right', 179, 74, 15, false, false, NULL),
    ('30000000-0000-0000-0000-000000000023', '31000000-0000-0000-0000-000000000023', '20000000-0000-0000-0000-000000000001', 'Dylan Romain', 'Dylan', 'Romain', '2002-02-27', 'defender', 'centre_back', 'right', 185, 80, 16, false, false, NULL),
    ('30000000-0000-0000-0000-000000000024', '31000000-0000-0000-0000-000000000024', '20000000-0000-0000-0000-000000000001', 'Warren Bonte', 'Warren', 'Bonte', '2003-08-19', 'midfielder', 'central_midfield', 'right', 177, 71, 18, false, false, NULL),
    ('30000000-0000-0000-0000-000000000025', '31000000-0000-0000-0000-000000000025', '20000000-0000-0000-0000-000000000001', 'Fabien Sinon', 'Fabien', 'Sinon', '2004-06-11', 'midfielder', 'attacking_midfield', 'left', 173, 67, 20, false, false, NULL),
    ('30000000-0000-0000-0000-000000000026', '31000000-0000-0000-0000-000000000026', '20000000-0000-0000-0000-000000000001', 'Lucas Tirant', 'Lucas', 'Tirant', '2001-09-02', 'forward', 'centre_forward', 'right', 183, 77, 21, false, false, NULL),
    -- Mahe City: nine additional starters and six additional substitutes.
    ('30000000-0000-0000-0000-000000000027', '31000000-0000-0000-0000-000000000027', '20000000-0000-0000-0000-000000000002', 'Stefan Melanie', 'Stefan', 'Melanie', '1998-03-22', 'defender', 'right_back', 'right', 177, 72, 2, true, false, '2:4'),
    ('30000000-0000-0000-0000-000000000028', '31000000-0000-0000-0000-000000000028', '20000000-0000-0000-0000-000000000002', 'Colin Ah-Weng', 'Colin', 'Ah-Weng', '1996-07-14', 'defender', 'centre_back', 'right', 187, 82, 4, true, false, '2:3'),
    ('30000000-0000-0000-0000-000000000029', '31000000-0000-0000-0000-000000000029', '20000000-0000-0000-0000-000000000002', 'Pascal Agricole', 'Pascal', 'Agricole', '1999-01-26', 'defender', 'left_back', 'left', 176, 71, 3, true, false, '2:1'),
    ('30000000-0000-0000-0000-000000000030', '31000000-0000-0000-0000-000000000030', '20000000-0000-0000-0000-000000000002', 'Marvin Esparon', 'Marvin', 'Esparon', '1997-10-09', 'midfielder', 'defensive_midfield', 'right', 182, 77, 6, true, false, '3:2'),
    ('30000000-0000-0000-0000-000000000031', '31000000-0000-0000-0000-000000000031', '20000000-0000-0000-0000-000000000002', 'Terence Onezia', 'Terence', 'Onezia', '2000-05-04', 'midfielder', 'central_midfield', 'left', 178, 73, 8, true, false, '3:1'),
    ('30000000-0000-0000-0000-000000000032', '31000000-0000-0000-0000-000000000032', '20000000-0000-0000-0000-000000000002', 'Ronny Esther', 'Ronny', 'Esther', '2001-11-15', 'forward', 'right_winger', 'right', 175, 69, 7, true, false, '4:3'),
    ('30000000-0000-0000-0000-000000000033', '31000000-0000-0000-0000-000000000033', '20000000-0000-0000-0000-000000000002', 'Quincy Fanchette', 'Quincy', 'Fanchette', '1999-08-28', 'midfielder', 'attacking_midfield', 'right', 176, 70, 10, true, false, '4:2'),
    ('30000000-0000-0000-0000-000000000034', '31000000-0000-0000-0000-000000000034', '20000000-0000-0000-0000-000000000002', 'Sheldon Madeleine', 'Sheldon', 'Madeleine', '2002-04-07', 'forward', 'left_winger', 'left', 174, 68, 11, true, false, '4:1'),
    ('30000000-0000-0000-0000-000000000035', '31000000-0000-0000-0000-000000000035', '20000000-0000-0000-0000-000000000002', 'Gavin Jeanne', 'Gavin', 'Jeanne', '1998-12-01', 'forward', 'centre_forward', 'right', 184, 79, 9, true, false, '5:2'),
    ('30000000-0000-0000-0000-000000000036', '31000000-0000-0000-0000-000000000036', '20000000-0000-0000-0000-000000000002', 'Perry Monnaie', 'Perry', 'Monnaie', '1996-02-20', 'goalkeeper', 'goalkeeper', 'right', 189, 85, 12, false, false, NULL),
    ('30000000-0000-0000-0000-000000000037', '31000000-0000-0000-0000-000000000037', '20000000-0000-0000-0000-000000000002', 'Alvin Bristol', 'Alvin', 'Bristol', '2000-06-23', 'defender', 'centre_back', 'right', 185, 80, 13, false, false, NULL),
    ('30000000-0000-0000-0000-000000000038', '31000000-0000-0000-0000-000000000038', '20000000-0000-0000-0000-000000000002', 'Damien Decommarmond', 'Damien', 'Decommarmond', '2001-01-13', 'defender', 'left_back', 'left', 178, 72, 14, false, false, NULL),
    ('30000000-0000-0000-0000-000000000039', '31000000-0000-0000-0000-000000000039', '20000000-0000-0000-0000-000000000002', 'Jude Mathiot', 'Jude', 'Mathiot', '2003-03-31', 'midfielder', 'central_midfield', 'right', 176, 70, 15, false, false, NULL),
    ('30000000-0000-0000-0000-000000000040', '31000000-0000-0000-0000-000000000040', '20000000-0000-0000-0000-000000000002', 'Trevor Henriette', 'Trevor', 'Henriette', '2002-09-10', 'forward', 'right_winger', 'right', 175, 69, 17, false, false, NULL),
    ('30000000-0000-0000-0000-000000000041', '31000000-0000-0000-0000-000000000041', '20000000-0000-0000-0000-000000000002', 'Wesley Marie', 'Wesley', 'Marie', '2000-11-06', 'forward', 'centre_forward', 'left', 182, 76, 19, false, false, NULL);

INSERT INTO people (id, display_name, first_name, last_name, birth_date, country_code, metadata)
VALUES
    ('30000000-0000-0000-0000-000000000001', 'Alex Michel', 'Alex', 'Michel', '1998-04-12', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000002', 'Daniel Rose', 'Daniel', 'Rose', '1997-09-03', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000003', 'Marc Hoareau', 'Marc', 'Hoareau', '1999-02-21', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000004', 'Jules Payet', 'Jules', 'Payet', '2000-11-08', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000005', 'Patrick James', 'Patrick', 'James', '1976-06-17', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000006', 'Sophie Larue', 'Sophie', 'Larue', '1980-01-29', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000007', 'Jean Morel', 'Jean', 'Morel', '1985-03-14', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000008', 'Mason Desaubin', 'Mason', 'Desaubin', '2001-08-11', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000009', 'Liam Vidot', 'Liam', 'Vidot', '2002-05-19', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000010', 'Noah Sinon', 'Noah', 'Sinon', '1999-12-02', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000011', 'Ethan Julie', 'Ethan', 'Julie', '2003-07-23', 'SC', '{"demo":true}'::jsonb),
    ('30000000-0000-0000-0000-000000000042', 'Noris Arissol', 'Noris', 'Arissol', NULL, 'SC', '{"demo":true}'::jsonb)
ON CONFLICT (id) DO UPDATE SET
    display_name = EXCLUDED.display_name, first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name, birth_date = EXCLUDED.birth_date,
    country_code = EXCLUDED.country_code, metadata = EXCLUDED.metadata;

INSERT INTO people (id, display_name, first_name, last_name, birth_date, country_code, metadata)
SELECT person_id, display_name, first_name, last_name, birth_date, 'SC', '{"demo":true}'::jsonb
FROM demo_lineup_players
ON CONFLICT (id) DO UPDATE SET
    display_name = EXCLUDED.display_name, first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name, birth_date = EXCLUDED.birth_date,
    country_code = EXCLUDED.country_code, metadata = EXCLUDED.metadata;

INSERT INTO players (person_id, position, detailed_position, preferred_foot, height_cm, weight_kg)
VALUES
    ('30000000-0000-0000-0000-000000000001', 'forward', 'centre_forward', 'right', 181, 76),
    ('30000000-0000-0000-0000-000000000002', 'goalkeeper', 'goalkeeper', 'right', 188, 82),
    ('30000000-0000-0000-0000-000000000003', 'midfielder', 'central_midfield', 'left', 175, 70),
    ('30000000-0000-0000-0000-000000000004', 'defender', 'centre_back', 'right', 184, 79),
    ('30000000-0000-0000-0000-000000000008', 'midfielder', 'central_midfield', 'right', 178, 72),
    ('30000000-0000-0000-0000-000000000009', 'forward', 'right_winger', 'left', 174, 68),
    ('30000000-0000-0000-0000-000000000010', 'defender', 'centre_back', 'right', 186, 81),
    ('30000000-0000-0000-0000-000000000011', 'midfielder', 'attacking_midfield', 'right', 176, 70)
ON CONFLICT (person_id) DO UPDATE SET
    position = EXCLUDED.position, detailed_position = EXCLUDED.detailed_position,
    preferred_foot = EXCLUDED.preferred_foot, height_cm = EXCLUDED.height_cm, weight_kg = EXCLUDED.weight_kg;

INSERT INTO players (person_id, position, detailed_position, preferred_foot, height_cm, weight_kg)
SELECT person_id, position, detailed_position, preferred_foot, height_cm, weight_kg
FROM demo_lineup_players
ON CONFLICT (person_id) DO UPDATE SET
    position = EXCLUDED.position, detailed_position = EXCLUDED.detailed_position,
    preferred_foot = EXCLUDED.preferred_foot, height_cm = EXCLUDED.height_cm,
    weight_kg = EXCLUDED.weight_kg;

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
    ('31000000-0000-0000-0000-000000000006', '20000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000006', 'head_coach', NULL, '2026-01-01'),
    ('31000000-0000-0000-0000-000000000008', '20000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000008', 'player', 8, '2026-01-01'),
    ('31000000-0000-0000-0000-000000000009', '20000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000009', 'player', 11, '2026-01-01'),
    ('31000000-0000-0000-0000-000000000010', '20000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000010', 'player', 5, '2026-01-01'),
    ('31000000-0000-0000-0000-000000000011', '20000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000011', 'player', 18, '2026-01-01')
ON CONFLICT (id) DO UPDATE SET
    team_id = EXCLUDED.team_id, person_id = EXCLUDED.person_id, role = EXCLUDED.role,
    shirt_number = EXCLUDED.shirt_number, starts_on = EXCLUDED.starts_on, ends_on = NULL;

INSERT INTO team_memberships (id, team_id, person_id, role, shirt_number, starts_on)
SELECT membership_id, team_id, person_id, 'player', shirt_number, '2026-01-01'
FROM demo_lineup_players
ON CONFLICT (id) DO UPDATE SET
    team_id = EXCLUDED.team_id, person_id = EXCLUDED.person_id, role = EXCLUDED.role,
    shirt_number = EXCLUDED.shirt_number, starts_on = EXCLUDED.starts_on, ends_on = NULL;

UPDATE team_memberships
SET transfer_type = 'permanent', is_loan = false, parent_team_id = NULL
WHERE id::text LIKE '31000000-0000-0000-0000-0000000000%';

INSERT INTO team_memberships (
    id, team_id, person_id, role, shirt_number, starts_on, ends_on,
    is_loan, parent_team_id, transfer_type
)
VALUES (
    '31000000-0000-0000-0000-000000000101',
    '20000000-0000-0000-0000-000000000003',
    '30000000-0000-0000-0000-000000000001',
    'player', 19, '2024-01-01', '2025-12-31', false, NULL, 'permanent'
)
ON CONFLICT (id) DO UPDATE SET
    team_id = EXCLUDED.team_id, person_id = EXCLUDED.person_id, role = EXCLUDED.role,
    shirt_number = EXCLUDED.shirt_number, starts_on = EXCLUDED.starts_on,
    ends_on = EXCLUDED.ends_on, is_loan = EXCLUDED.is_loan,
    parent_team_id = EXCLUDED.parent_team_id, transfer_type = EXCLUDED.transfer_type;

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
     digest('demo-finished-match-1-v1', 'sha256'), '{"demo":true}'::jsonb),
    ('40000000-0000-0000-0000-000000000004', '10000000-0000-0000-0000-000000000001',
     '10000000-0000-0000-0000-000000000002', 'regular_season', 'Round 7', 7, NULL, 1,
     '2026-06-28T14:00:00Z', 'finished', 'full_time', 90, '10000000-0000-0000-0000-000000000003',
     '20000000-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000002',
     1, 1, 0, 1, NULL, NULL, NULL, NULL, 6901, NULL, 1,
     digest('demo-finished-match-2-v1', 'sha256'), '{"demo":true}'::jsonb),
    ('40000000-0000-0000-0000-000000000005', '10000000-0000-0000-0000-000000000001',
     '10000000-0000-0000-0000-000000000002', 'regular_season', 'Round 3', 3, NULL, 1,
     '2026-03-15T14:00:00Z', 'finished', 'full_time', 90, '10000000-0000-0000-0000-000000000003',
     '20000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000003',
     0, 2, 0, 1, NULL, NULL, NULL, NULL, 6218, '20000000-0000-0000-0000-000000000003', 1,
     digest('demo-finished-match-3-v1', 'sha256'), '{"demo":true}'::jsonb),
    ('40000000-0000-0000-0000-000000000006', '10000000-0000-0000-0000-000000000001',
     '10000000-0000-0000-0000-000000000002', 'playoff', 'Premier League Playoff', 1, NULL, 1,
     '2026-05-29T14:30:00Z', 'finished', 'full_time', 90, '10000000-0000-0000-0000-000000000004',
     '20000000-0000-0000-0000-000000000006', '20000000-0000-0000-0000-000000000005',
     4, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '20000000-0000-0000-0000-000000000006', 1,
     digest('demo-st-michel-rovers-playoff-v1', 'sha256'), '{"demo":true}'::jsonb)
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
WHERE match_id IN (
    '40000000-0000-0000-0000-000000000001',
    '40000000-0000-0000-0000-000000000003',
    '40000000-0000-0000-0000-000000000006'
);

INSERT INTO match_events (
    id, match_id, sequence, period, minute, type, team_id, primary_person_id, secondary_person_id,
    detail, home_score, away_score, metadata, occurred_at
)
VALUES
    ('41000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001',
     1, 'first_half', 0, 'kickoff', NULL, NULL, NULL, 'Match started', 0, 0, '{"demo":true}'::jsonb, '2026-08-01T12:00:00Z'),
    ('41000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000001',
     2, 'first_half', 34, 'goal', '20000000-0000-0000-0000-000000000001',
     '30000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000008', 'Right-footed shot; assisted by Mason Desaubin', 1, 0,
     '{"demo":true}'::jsonb, '2026-08-01T12:34:00Z'),
    ('41000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000001',
     3, 'second_half', 50, 'yellow_card', '20000000-0000-0000-0000-000000000002',
     '30000000-0000-0000-0000-000000000010', NULL, 'Unsporting behaviour', 1, 0,
     '{"card_reason":"unsporting_behaviour","demo":true}'::jsonb, '2026-08-01T13:05:00Z'),
    ('41000000-0000-0000-0000-000000000006', '40000000-0000-0000-0000-000000000001',
     4, 'second_half', 62, 'substitution', '20000000-0000-0000-0000-000000000001',
     '30000000-0000-0000-0000-000000000009', '30000000-0000-0000-0000-000000000001',
     'Liam Vidot replaces Alex Michel', 1, 0, '{"demo":true}'::jsonb, '2026-08-01T13:17:00Z'),
    ('41000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000003',
     1, 'first_half', 0, 'kickoff', NULL, NULL, NULL, 'Match started', 0, 0, '{"demo":true}'::jsonb, '2026-07-31T16:00:00Z'),
    ('41000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000003',
     2, 'second_half', 90, 'full_time', NULL, NULL, NULL, 'Match finished', 2, 1,
     '{"demo":true}'::jsonb, '2026-07-31T17:50:00Z'),
    ('41000000-0000-0000-0000-000000000007', '40000000-0000-0000-0000-000000000006',
     1, 'first_half', 0, 'kickoff', NULL, NULL, NULL, 'Match started', 0, 0,
     '{"demo":true}'::jsonb, '2026-05-29T14:30:00Z'),
    ('41000000-0000-0000-0000-000000000008', '40000000-0000-0000-0000-000000000006',
     2, 'second_half', 90, 'full_time', NULL, NULL, NULL, 'Match finished', 4, 3,
     '{"demo":true}'::jsonb, '2026-05-29T16:20:00Z');

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

INSERT INTO match_lineups (
    match_id, team_id, person_id, position, grid_position, shirt_number,
    is_starter, is_captain, metadata
)
VALUES
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001',
     '30000000-0000-0000-0000-000000000001', 'forward', '4:2', 9, true, true, '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001',
     '30000000-0000-0000-0000-000000000008', 'midfielder', '3:2', 8, true, false, '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001',
     '30000000-0000-0000-0000-000000000009', 'forward', NULL, 11, false, false, '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002',
     '30000000-0000-0000-0000-000000000002', 'goalkeeper', '1:1', 1, true, false, '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002',
     '30000000-0000-0000-0000-000000000010', 'defender', '2:2', 5, true, true, '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002',
     '30000000-0000-0000-0000-000000000011', 'midfielder', NULL, 18, false, false, '{"demo":true}')
ON CONFLICT (match_id, team_id, person_id) DO UPDATE SET
    position = EXCLUDED.position, grid_position = EXCLUDED.grid_position,
    shirt_number = EXCLUDED.shirt_number, is_starter = EXCLUDED.is_starter,
    is_captain = EXCLUDED.is_captain, metadata = EXCLUDED.metadata;

INSERT INTO match_lineups (
    match_id, team_id, person_id, position, grid_position, shirt_number,
    is_starter, is_captain, metadata
)
SELECT
    '40000000-0000-0000-0000-000000000001', team_id, person_id, position,
    grid_position, shirt_number, is_starter, is_captain, '{"demo":true}'::jsonb
FROM demo_lineup_players
ON CONFLICT (match_id, team_id, person_id) DO UPDATE SET
    position = EXCLUDED.position, grid_position = EXCLUDED.grid_position,
    shirt_number = EXCLUDED.shirt_number, is_starter = EXCLUDED.is_starter,
    is_captain = EXCLUDED.is_captain, metadata = EXCLUDED.metadata;

INSERT INTO match_officials (match_id, person_id, role, metadata)
VALUES
    ('40000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000007', 'referee', '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000006', '30000000-0000-0000-0000-000000000042', 'referee', '{"demo":true}')
ON CONFLICT (match_id, person_id, role) DO UPDATE SET metadata = EXCLUDED.metadata;

INSERT INTO player_match_statistics (
    match_id, team_id, person_id, started, minutes_played, goals, assists,
    shots, shots_on_target, passes, tackles, saves, yellow_cards, red_cards, rating, metadata
)
VALUES
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001',
     '30000000-0000-0000-0000-000000000001', true, 62, 1, 0, 3, 2, 21, 1, 0, 0, 0, 8.10, '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001',
     '30000000-0000-0000-0000-000000000008', true, 67, 0, 1, 1, 0, 34, 3, 0, 0, 0, 7.50, '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001',
     '30000000-0000-0000-0000-000000000009', false, 5, 0, 0, 1, 0, 4, 0, 0, 0, 0, 6.70, '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002',
     '30000000-0000-0000-0000-000000000002', true, 67, 0, 0, 0, 0, 17, 0, 4, 0, 0, 7.20, '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002',
     '30000000-0000-0000-0000-000000000010', true, 67, 0, 0, 0, 0, 29, 4, 0, 1, 0, 6.80, '{"demo":true}')
ON CONFLICT (match_id, person_id) DO UPDATE SET
    team_id = EXCLUDED.team_id, started = EXCLUDED.started, minutes_played = EXCLUDED.minutes_played,
    goals = EXCLUDED.goals, assists = EXCLUDED.assists, shots = EXCLUDED.shots,
    shots_on_target = EXCLUDED.shots_on_target, passes = EXCLUDED.passes,
    tackles = EXCLUDED.tackles, saves = EXCLUDED.saves,
    yellow_cards = EXCLUDED.yellow_cards, red_cards = EXCLUDED.red_cards,
    rating = EXCLUDED.rating, metadata = EXCLUDED.metadata;

UPDATE player_match_statistics
SET passes_completed = CASE person_id
        WHEN '30000000-0000-0000-0000-000000000001' THEN 17
        WHEN '30000000-0000-0000-0000-000000000008' THEN 29
        WHEN '30000000-0000-0000-0000-000000000009' THEN 3
        WHEN '30000000-0000-0000-0000-000000000002' THEN 13
        WHEN '30000000-0000-0000-0000-000000000010' THEN 23
    END,
    key_passes = CASE person_id
        WHEN '30000000-0000-0000-0000-000000000001' THEN 2
        WHEN '30000000-0000-0000-0000-000000000008' THEN 4
        ELSE 0
    END,
    interceptions = CASE person_id
        WHEN '30000000-0000-0000-0000-000000000010' THEN 4
        ELSE 1
    END,
    clearances = CASE person_id
        WHEN '30000000-0000-0000-0000-000000000010' THEN 6
        ELSE 0
    END,
    blocks = CASE person_id
        WHEN '30000000-0000-0000-0000-000000000010' THEN 2
        ELSE 0
    END,
    duels = CASE person_id
        WHEN '30000000-0000-0000-0000-000000000001' THEN 9
        WHEN '30000000-0000-0000-0000-000000000008' THEN 8
        ELSE 4
    END,
    duels_won = CASE person_id
        WHEN '30000000-0000-0000-0000-000000000001' THEN 6
        WHEN '30000000-0000-0000-0000-000000000008' THEN 5
        ELSE 2
    END,
    expected_goals = CASE person_id
        WHEN '30000000-0000-0000-0000-000000000001' THEN 0.730
        WHEN '30000000-0000-0000-0000-000000000008' THEN 0.080
        WHEN '30000000-0000-0000-0000-000000000009' THEN 0.050
        ELSE 0.000
    END,
    expected_assists = CASE person_id
        WHEN '30000000-0000-0000-0000-000000000001' THEN 0.180
        WHEN '30000000-0000-0000-0000-000000000008' THEN 0.420
        ELSE 0.000
    END
WHERE match_id = '40000000-0000-0000-0000-000000000001';

INSERT INTO player_trait_snapshots (
    id, person_id, team_id, league_id, season_id, source, external_id,
    position_group, minimum_minutes, cohort_size, player_minutes, observed_at, metadata
)
VALUES (
    '70000000-0000-0000-0000-000000000001',
    '30000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000002',
    'demo', 'alex-michel-2026-traits', 'forward', 450, 42, 882,
    '2026-08-01T13:20:00Z', '{"demo":true}'
)
ON CONFLICT (source, external_id) DO UPDATE SET
    person_id = EXCLUDED.person_id, team_id = EXCLUDED.team_id,
    league_id = EXCLUDED.league_id, season_id = EXCLUDED.season_id,
    position_group = EXCLUDED.position_group, minimum_minutes = EXCLUDED.minimum_minutes,
    cohort_size = EXCLUDED.cohort_size, player_minutes = EXCLUDED.player_minutes,
    observed_at = EXCLUDED.observed_at, metadata = EXCLUDED.metadata;

DELETE FROM player_trait_metrics
WHERE snapshot_id = '70000000-0000-0000-0000-000000000001';

INSERT INTO player_trait_metrics (
    snapshot_id, metric_key, label, category, raw_value, per_90_value,
    percentile, unit, direction
)
VALUES
    ('70000000-0000-0000-0000-000000000001', 'goals', 'Goals', 'attacking', 9, 0.92, 91, 'per_90', 'higher_is_better'),
    ('70000000-0000-0000-0000-000000000001', 'expected_goals', 'Expected goals', 'attacking', 7.8, 0.80, 86, 'xg_per_90', 'higher_is_better'),
    ('70000000-0000-0000-0000-000000000001', 'key_passes', 'Key passes', 'creation', 24, 2.45, 78, 'per_90', 'higher_is_better'),
    ('70000000-0000-0000-0000-000000000001', 'pressures', 'Pressures', 'defending', 138, 14.08, 69, 'per_90', 'higher_is_better');

INSERT INTO player_spatial_snapshots (
    id, match_id, person_id, team_id, source, external_id,
    orientation, observed_at, metadata
)
VALUES (
    '71000000-0000-0000-0000-000000000001',
    '40000000-0000-0000-0000-000000000001',
    '30000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000001',
    'demo', 'live-match-1-alex-spatial', 'attacking_left_to_right',
    '2026-08-01T13:20:00Z', '{"demo":true}'
)
ON CONFLICT (source, external_id) DO UPDATE SET
    match_id = EXCLUDED.match_id, person_id = EXCLUDED.person_id,
    team_id = EXCLUDED.team_id, orientation = EXCLUDED.orientation,
    observed_at = EXCLUDED.observed_at, metadata = EXCLUDED.metadata;

DELETE FROM player_touch_points
WHERE snapshot_id = '71000000-0000-0000-0000-000000000001';
DELETE FROM player_shots
WHERE snapshot_id = '71000000-0000-0000-0000-000000000001';

INSERT INTO player_touch_points (
    snapshot_id, sequence, minute, x, y, intensity, touch_type
)
VALUES
    ('71000000-0000-0000-0000-000000000001', 1, 8, 58, 42, 1.0, 'receive'),
    ('71000000-0000-0000-0000-000000000001', 2, 18, 72, 28, 1.2, 'carry'),
    ('71000000-0000-0000-0000-000000000001', 3, 34, 88, 51, 1.8, 'shot'),
    ('71000000-0000-0000-0000-000000000001', 4, 47, 77, 66, 1.1, 'receive'),
    ('71000000-0000-0000-0000-000000000001', 5, 61, 84, 39, 1.3, 'carry');

INSERT INTO player_shots (
    id, snapshot_id, sequence, match_event_id, minute, x, y,
    expected_goals, outcome, body_part, shot_type
)
VALUES
    ('72000000-0000-0000-0000-000000000001', '71000000-0000-0000-0000-000000000001',
     1, NULL, 19, 83, 44, 0.1200, 'saved', 'right_foot', 'open_play'),
    ('72000000-0000-0000-0000-000000000002', '71000000-0000-0000-0000-000000000001',
     2, '41000000-0000-0000-0000-000000000002', 34, 88, 51, 0.6100, 'goal', 'right_foot', 'open_play');

INSERT INTO player_valuations (
    id, person_id, team_id, source, external_id, amount_minor,
    currency, valued_on, observed_at, metadata
)
VALUES (
    '73000000-0000-0000-0000-000000000001',
    '30000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000001',
    'demo', 'alex-michel-2026-valuation', 12500000, 'EUR',
    '2026-08-01', '2026-08-01T10:00:00Z', '{"demo":true}'
)
ON CONFLICT (source, external_id) DO UPDATE SET
    person_id = EXCLUDED.person_id, team_id = EXCLUDED.team_id,
    amount_minor = EXCLUDED.amount_minor, currency = EXCLUDED.currency,
    valued_on = EXCLUDED.valued_on, observed_at = EXCLUDED.observed_at,
    metadata = EXCLUDED.metadata;

INSERT INTO match_team_statistics (
    match_id, team_id, possession, shots, shots_on_target, shots_off_target,
    blocked_shots, shots_inside_box, shots_outside_box, corners, passes,
    passes_completed, pass_accuracy, fouls, offsides, yellow_cards, red_cards,
    saves, tackles, interceptions, clearances, expected_goals, metadata
)
VALUES
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001',
     54.30, 12, 5, 4, 3, 8, 4, 6, 451, 386, 85.59, 11, 2, 1, 0, 4, 17, 8, 12, 1.420, '{"demo":true}'),
    ('40000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002',
     45.70, 9, 4, 3, 2, 5, 4, 4, 382, 308, 80.63, 14, 1, 2, 0, 4, 19, 7, 15, 0.860, '{"demo":true}')
ON CONFLICT (match_id, team_id) DO UPDATE SET
    possession = EXCLUDED.possession, shots = EXCLUDED.shots,
    shots_on_target = EXCLUDED.shots_on_target, shots_off_target = EXCLUDED.shots_off_target,
    blocked_shots = EXCLUDED.blocked_shots, shots_inside_box = EXCLUDED.shots_inside_box,
    shots_outside_box = EXCLUDED.shots_outside_box, corners = EXCLUDED.corners,
    passes = EXCLUDED.passes, passes_completed = EXCLUDED.passes_completed,
    pass_accuracy = EXCLUDED.pass_accuracy, fouls = EXCLUDED.fouls,
    offsides = EXCLUDED.offsides, yellow_cards = EXCLUDED.yellow_cards,
    red_cards = EXCLUDED.red_cards, saves = EXCLUDED.saves, tackles = EXCLUDED.tackles,
    interceptions = EXCLUDED.interceptions, clearances = EXCLUDED.clearances,
    expected_goals = EXCLUDED.expected_goals, metadata = EXCLUDED.metadata;

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

INSERT INTO bookmakers (id, slug, name, website_url, metadata)
VALUES (
    '60000000-0000-0000-0000-000000000001', 'demo-sportsbook', 'Demo Sportsbook',
    'https://example.com/demo-sportsbook', '{"demo":true}'
)
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name, website_url = EXCLUDED.website_url, metadata = EXCLUDED.metadata;

INSERT INTO odds_snapshots (
    id, match_id, bookmaker_id, source, external_id, observed_at, valid_until, metadata
)
VALUES (
    '61000000-0000-0000-0000-000000000001',
    '40000000-0000-0000-0000-000000000002',
    '60000000-0000-0000-0000-000000000001',
    'demo', 'scheduled-match-1-odds', '2026-08-01T11:00:00Z', '2026-08-02T14:00:00Z',
    '{"demo":true}'
)
ON CONFLICT (source, external_id) DO UPDATE SET
    match_id = EXCLUDED.match_id, bookmaker_id = EXCLUDED.bookmaker_id,
    observed_at = EXCLUDED.observed_at, valid_until = EXCLUDED.valid_until,
    metadata = EXCLUDED.metadata;

DELETE FROM odds_markets WHERE snapshot_id = '61000000-0000-0000-0000-000000000001';

INSERT INTO odds_markets (id, snapshot_id, market_key, name, status, metadata)
VALUES
    ('62000000-0000-0000-0000-000000000001', '61000000-0000-0000-0000-000000000001',
     'match_winner', 'Match winner', 'open', '{"demo":true}'),
    ('62000000-0000-0000-0000-000000000002', '61000000-0000-0000-0000-000000000001',
     'total_goals', 'Total goals', 'open', '{"demo":true}');

INSERT INTO odds_selections (
    id, market_id, selection_key, name, line, decimal_odds, metadata
)
VALUES
    ('63000000-0000-0000-0000-000000000001', '62000000-0000-0000-0000-000000000001',
     'home', 'Praslin Rovers', NULL, 2.1500, '{"demo":true}'),
    ('63000000-0000-0000-0000-000000000002', '62000000-0000-0000-0000-000000000001',
     'draw', 'Draw', NULL, 3.1000, '{"demo":true}'),
    ('63000000-0000-0000-0000-000000000003', '62000000-0000-0000-0000-000000000001',
     'away', 'La Digue Athletic', NULL, 3.2500, '{"demo":true}'),
    ('63000000-0000-0000-0000-000000000004', '62000000-0000-0000-0000-000000000002',
     'over', 'Over 2.5', 2.500, 1.9200, '{"demo":true}'),
    ('63000000-0000-0000-0000-000000000005', '62000000-0000-0000-0000-000000000002',
     'under', 'Under 2.5', 2.500, 1.8800, '{"demo":true}');

INSERT INTO match_broadcasts (
    id, match_id, source, external_id, network_name, service_name, kind,
    availability_scope, language_tags, starts_at, ends_at, is_free,
    requires_subscription, web_url, deep_link_url, status, observed_at, metadata
)
VALUES (
    '64000000-0000-0000-0000-000000000001',
    '40000000-0000-0000-0000-000000000002',
    'demo', 'scheduled-match-1-stream', 'SeySoccer Demo', 'SeySoccer Live', 'stream',
    'territorial', ARRAY['en','fr'], '2026-08-02T13:45:00Z', '2026-08-02T16:00:00Z',
    true, false, 'https://example.com/seysoccer/live', 'https://example.com/seysoccer/live',
    'scheduled', '2026-08-01T11:00:00Z', '{"demo":true}'
)
ON CONFLICT (source, external_id) DO UPDATE SET
    match_id = EXCLUDED.match_id, network_name = EXCLUDED.network_name,
    service_name = EXCLUDED.service_name, kind = EXCLUDED.kind,
    availability_scope = EXCLUDED.availability_scope, language_tags = EXCLUDED.language_tags,
    starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at, is_free = EXCLUDED.is_free,
    requires_subscription = EXCLUDED.requires_subscription, web_url = EXCLUDED.web_url,
    deep_link_url = EXCLUDED.deep_link_url, status = EXCLUDED.status,
    observed_at = EXCLUDED.observed_at, metadata = EXCLUDED.metadata;

INSERT INTO match_broadcast_regions (broadcast_id, country_code)
VALUES ('64000000-0000-0000-0000-000000000001', 'SC')
ON CONFLICT (broadcast_id, country_code) DO NOTHING;

INSERT INTO match_weather_snapshots (
    id, match_id, source, external_id, kind, valid_at, issued_at,
    temperature_c, feels_like_c, humidity_percent,
    precipitation_probability_percent, precipitation_mm, wind_speed_kph,
    wind_gust_kph, wind_direction_degrees, pressure_hpa, visibility_km,
    condition_code, condition_text, icon_url, metadata
)
VALUES (
    '65000000-0000-0000-0000-000000000001',
    '40000000-0000-0000-0000-000000000002',
    'demo', 'scheduled-match-1-forecast', 'forecast',
    '2026-08-02T14:00:00Z', '2026-08-01T11:00:00Z',
    27.40, 30.10, 78.00, 35.00, 0.40, 16.50, 24.00, 125,
    1011.20, 10.00, 'partly_cloudy', 'Partly cloudy',
    'https://example.com/weather/partly-cloudy.png', '{"demo":true}'
)
ON CONFLICT (source, external_id) DO UPDATE SET
    match_id = EXCLUDED.match_id, kind = EXCLUDED.kind, valid_at = EXCLUDED.valid_at,
    issued_at = EXCLUDED.issued_at, temperature_c = EXCLUDED.temperature_c,
    feels_like_c = EXCLUDED.feels_like_c, humidity_percent = EXCLUDED.humidity_percent,
    precipitation_probability_percent = EXCLUDED.precipitation_probability_percent,
    precipitation_mm = EXCLUDED.precipitation_mm, wind_speed_kph = EXCLUDED.wind_speed_kph,
    wind_gust_kph = EXCLUDED.wind_gust_kph,
    wind_direction_degrees = EXCLUDED.wind_direction_degrees,
    pressure_hpa = EXCLUDED.pressure_hpa, visibility_km = EXCLUDED.visibility_km,
    condition_code = EXCLUDED.condition_code, condition_text = EXCLUDED.condition_text,
    icon_url = EXCLUDED.icon_url, metadata = EXCLUDED.metadata;

INSERT INTO external_ids (entity_type, entity_id, provider, external_id)
VALUES
    ('league', '10000000-0000-0000-0000-000000000001', 'demo', 'seychelles-premier-demo'),
    ('season', '10000000-0000-0000-0000-000000000002', 'demo', 'season-2026'),
    ('venue', '10000000-0000-0000-0000-000000000003', 'demo', 'national-stadium-demo'),
    ('venue', '10000000-0000-0000-0000-000000000004', 'demo', 'stad-linite'),
    ('team', '20000000-0000-0000-0000-000000000001', 'demo', 'victoria-united'),
    ('team', '20000000-0000-0000-0000-000000000002', 'demo', 'mahe-city'),
    ('team', '20000000-0000-0000-0000-000000000003', 'demo', 'praslin-rovers'),
    ('team', '20000000-0000-0000-0000-000000000004', 'demo', 'la-digue-athletic'),
    ('team', '20000000-0000-0000-0000-000000000005', 'demo', 'rovers'),
    ('team', '20000000-0000-0000-0000-000000000006', 'demo', 'st-michel'),
    ('person', '30000000-0000-0000-0000-000000000001', 'demo', 'alex-michel'),
    ('person', '30000000-0000-0000-0000-000000000002', 'demo', 'daniel-rose'),
    ('person', '30000000-0000-0000-0000-000000000003', 'demo', 'marc-hoareau'),
    ('person', '30000000-0000-0000-0000-000000000004', 'demo', 'jules-payet'),
    ('person', '30000000-0000-0000-0000-000000000005', 'demo', 'coach-james'),
    ('person', '30000000-0000-0000-0000-000000000006', 'demo', 'coach-larue'),
    ('person', '30000000-0000-0000-0000-000000000007', 'demo', 'referee-morel'),
    ('person', '30000000-0000-0000-0000-000000000008', 'demo', 'mason-desaubin'),
    ('person', '30000000-0000-0000-0000-000000000009', 'demo', 'liam-vidot'),
    ('person', '30000000-0000-0000-0000-000000000010', 'demo', 'noah-sinon'),
    ('person', '30000000-0000-0000-0000-000000000011', 'demo', 'ethan-julie'),
    ('person', '30000000-0000-0000-0000-000000000042', 'demo', 'referee-noris-arissol'),
    ('match', '40000000-0000-0000-0000-000000000001', 'demo', 'live-match-1'),
    ('match', '40000000-0000-0000-0000-000000000002', 'demo', 'scheduled-match-1'),
    ('match', '40000000-0000-0000-0000-000000000003', 'demo', 'finished-match-1'),
    ('match', '40000000-0000-0000-0000-000000000004', 'demo', 'finished-match-2'),
    ('match', '40000000-0000-0000-0000-000000000005', 'demo', 'finished-match-3'),
    ('match', '40000000-0000-0000-0000-000000000006', 'demo', 'st-michel-rovers-playoff'),
    ('match_event', '41000000-0000-0000-0000-000000000001', 'demo', 'live-kickoff'),
    ('match_event', '41000000-0000-0000-0000-000000000002', 'demo', 'live-goal-1'),
    ('match_event', '41000000-0000-0000-0000-000000000005', 'demo', 'live-yellow-1'),
    ('match_event', '41000000-0000-0000-0000-000000000006', 'demo', 'live-substitution-1'),
    ('match_event', '41000000-0000-0000-0000-000000000003', 'demo', 'finished-kickoff'),
    ('match_event', '41000000-0000-0000-0000-000000000004', 'demo', 'finished-full-time'),
    ('match_event', '41000000-0000-0000-0000-000000000007', 'demo', 'st-michel-rovers-kickoff'),
    ('match_event', '41000000-0000-0000-0000-000000000008', 'demo', 'st-michel-rovers-full-time')
ON CONFLICT (provider, entity_type, external_id) DO UPDATE SET
    entity_id = EXCLUDED.entity_id, updated_at = now();

INSERT INTO news_articles (
    id, source, external_id, slug, title, summary, body_markdown,
    hero_image_url, hero_image_alt, author_name, category, featured,
    related_league_id, related_team_id, related_match_id, status,
    published_at, source_hash
)
VALUES
    (
        '50000000-0000-0000-0000-000000000001', 'demo', 'victoria-mahe-report',
        'victoria-edge-mahe-in-demo-opener', 'Victoria edge Mahé in demo opener',
        'Victoria United took all three points after a disciplined second-half performance against Mahé City.',
        E'## Victoria make the perfect start\n\nVictoria United opened the demo season with a narrow win over Mahé City. The match report is linked to the fixture so clients can offer a direct route back to match details.\n\n### At a glance\n\n- Victoria United 1–0 Mahé City\n- Winning goal: 34th minute\n- Venue: Demo National Stadium',
        NULL, NULL,
        'SSB Newsroom', 'match_report', true,
        '10000000-0000-0000-0000-000000000001',
        '20000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000001',
        'published', '2026-08-01T10:00:00Z', digest('demo-news-victoria-mahe-v1', 'sha256')
    ),
    (
        '50000000-0000-0000-0000-000000000002', 'demo', 'weekend-preview',
        'weekend-football-preview', 'Weekend football preview',
        'Four clubs return to action across the Seychelles Demo Premier League this weekend.',
        E'## This weekend in the Demo Premier League\n\nThe next round brings two fixtures and plenty to watch, with Victoria United looking to protect their early lead. Follow SeySoccer for live scores and match events.',
        NULL, NULL, 'SSB Newsroom', 'story', false,
        '10000000-0000-0000-0000-000000000001', NULL, NULL,
        'published', '2026-07-31T09:00:00Z', digest('demo-news-weekend-preview-v1', 'sha256')
    ),
    (
        '50000000-0000-0000-0000-000000000003', 'demo', 'transfer-draft',
        'transfer-window-notes', 'Transfer window notes',
        'Editorial notes for a future transfer-window announcement.',
        'This draft must never appear in the public feed.',
        NULL, NULL, 'SSB Newsroom', 'announcement', false,
        NULL, NULL, NULL, 'draft', NULL, digest('demo-news-transfer-draft-v1', 'sha256')
    ),
    (
        '50000000-0000-0000-0000-000000000004', 'demo', 'st-michel-rovers-playoff-report',
        'st-michel-edge-rovers-in-playoff', 'St Michel edge Rovers in playoff',
        'St Michel defeated Rovers 4–3 at Stad Linite in the Premier League playoff.',
        E'## St Michel win seven-goal playoff\n\nSt Michel defeated Rovers 4–3 at Stad Linite on Friday, May 29, 2026. Noris Arissol refereed the Premier League playoff.\n\n### At a glance\n\n- St Michel 4–3 Rovers\n- Result: Full time\n- Venue: Stad Linite\n- Kickoff: 6:30 PM',
        NULL, NULL, 'SSB Newsroom', 'match_report', false,
        '10000000-0000-0000-0000-000000000001',
        '20000000-0000-0000-0000-000000000006',
        '40000000-0000-0000-0000-000000000006',
        'published', '2026-05-29T16:30:00Z', digest('demo-news-st-michel-rovers-playoff-v1', 'sha256')
    )
ON CONFLICT (source, external_id) DO UPDATE SET
    slug = EXCLUDED.slug, title = EXCLUDED.title, summary = EXCLUDED.summary,
    body_markdown = EXCLUDED.body_markdown, hero_image_url = EXCLUDED.hero_image_url,
    hero_image_alt = EXCLUDED.hero_image_alt, author_name = EXCLUDED.author_name,
    category = EXCLUDED.category, featured = EXCLUDED.featured,
    related_league_id = EXCLUDED.related_league_id,
    related_team_id = EXCLUDED.related_team_id,
    related_match_id = EXCLUDED.related_match_id, status = EXCLUDED.status,
    published_at = EXCLUDED.published_at, source_hash = EXCLUDED.source_hash,
    updated_at = now()
WHERE news_articles.source_hash IS DISTINCT FROM EXCLUDED.source_hash;
