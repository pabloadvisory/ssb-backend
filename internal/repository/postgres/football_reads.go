package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func (store *Store) GetVenue(ctx context.Context, id string) (football.Venue, error) {
	var venue football.Venue
	err := store.pool.QueryRow(ctx, `
		SELECT venue.id, venue.name, venue.city, venue.country_code, country.name,
			venue.address, venue.latitude, venue.longitude, venue.capacity, venue.surface,
			venue.image_url, venue.timezone, venue.created_at, venue.updated_at
		FROM venues venue
		LEFT JOIN countries country ON country.code = venue.country_code
		WHERE venue.id = $1`, id).Scan(
		&venue.ID, &venue.Name, &venue.City, &venue.CountryCode, &venue.CountryName,
		&venue.Address, &venue.Latitude, &venue.Longitude, &venue.Capacity, &venue.Surface,
		&venue.ImageURL, &venue.Timezone, &venue.CreatedAt, &venue.UpdatedAt,
	)
	return venue, mapError(err)
}

type matchReadSides struct {
	matchID string
	home    football.TeamSummary
	away    football.TeamSummary
}

func (store *Store) getMatchReadSides(ctx context.Context, matchID string) (matchReadSides, error) {
	var sides matchReadSides
	err := store.pool.QueryRow(ctx, `
		SELECT match.id,
			home.id, home.name, home.short_name, home.code, home.logo_url, home.primary_color, home.secondary_color,
			away.id, away.name, away.short_name, away.code, away.logo_url, away.primary_color, away.secondary_color
		FROM matches match
		JOIN teams home ON home.id = match.home_team_id
		JOIN teams away ON away.id = match.away_team_id
		WHERE match.id = $1`, matchID).Scan(
		&sides.matchID,
		&sides.home.ID, &sides.home.Name, &sides.home.ShortName, &sides.home.Code, &sides.home.LogoURL,
		&sides.home.PrimaryColor, &sides.home.SecondaryColor,
		&sides.away.ID, &sides.away.Name, &sides.away.ShortName, &sides.away.Code, &sides.away.LogoURL,
		&sides.away.PrimaryColor, &sides.away.SecondaryColor,
	)
	return sides, mapError(err)
}

func (store *Store) GetMatchLineups(ctx context.Context, matchID string) (football.MatchLineups, error) {
	sides, err := store.getMatchReadSides(ctx, matchID)
	if err != nil {
		return football.MatchLineups{}, err
	}
	lineups := football.MatchLineups{
		MatchID: sides.matchID,
		Home: football.TeamLineup{
			Team: sides.home, Starters: []football.LineupPlayer{}, Substitutes: []football.LineupPlayer{},
		},
		Away: football.TeamLineup{
			Team: sides.away, Starters: []football.LineupPlayer{}, Substitutes: []football.LineupPlayer{},
		},
	}

	infoRows, err := store.pool.Query(ctx, `
		SELECT info.team_id, info.formation,
			person.id, person.display_name, person.first_name, person.last_name,
			person.birth_date, person.country_code, person.photo_url
		FROM match_team_info info
		LEFT JOIN people person ON person.id = info.coach_id
		WHERE info.match_id = $1 AND info.team_id IN ($2, $3)
		ORDER BY CASE info.team_id WHEN $2 THEN 0 ELSE 1 END`, matchID, sides.home.ID, sides.away.ID)
	if err != nil {
		return football.MatchLineups{}, mapError(err)
	}
	for infoRows.Next() {
		var teamID string
		var formation *string
		var coachID, displayName, firstName, lastName, countryCode, photoURL *string
		var birthDate *time.Time
		if err := infoRows.Scan(
			&teamID, &formation, &coachID, &displayName, &firstName, &lastName,
			&birthDate, &countryCode, &photoURL,
		); err != nil {
			infoRows.Close()
			return football.MatchLineups{}, mapError(err)
		}
		lineup := lineupForTeam(&lineups, teamID)
		if lineup == nil {
			continue
		}
		lineup.Formation = formation
		if coachID != nil && displayName != nil {
			lineup.Coach = &football.Coach{Person: football.Person{
				ID: *coachID, DisplayName: *displayName, FirstName: firstName, LastName: lastName,
				BirthDate: birthDate, CountryCode: countryCode, PhotoURL: photoURL,
			}}
		}
	}
	if err := infoRows.Err(); err != nil {
		infoRows.Close()
		return football.MatchLineups{}, mapError(err)
	}
	infoRows.Close()

	rows, err := store.pool.Query(ctx, `
		SELECT lineup.team_id, lineup.is_starter, lineup.shirt_number, lineup.position,
			lineup.grid_position, lineup.is_captain,
			person.id, person.display_name, person.first_name, person.last_name,
			person.birth_date, person.country_code, person.photo_url,
			player.position, player.detailed_position, player.preferred_foot, player.height_cm,
			entered.id, entered.period, entered.minute, entered.stoppage_minute, entered.secondary_person_id,
			left_match.id, left_match.period, left_match.minute, left_match.stoppage_minute, left_match.primary_person_id
		FROM match_lineups lineup
		JOIN people person ON person.id = lineup.person_id
		LEFT JOIN players player ON player.person_id = person.id
		LEFT JOIN LATERAL (
			SELECT event.id, event.period, event.minute, event.stoppage_minute, event.secondary_person_id
			FROM match_events event
			WHERE event.match_id = lineup.match_id
				AND event.type = 'substitution'
				AND event.primary_person_id = lineup.person_id
			ORDER BY event.sequence
			LIMIT 1
		) entered ON true
		LEFT JOIN LATERAL (
			SELECT event.id, event.period, event.minute, event.stoppage_minute, event.primary_person_id
			FROM match_events event
			WHERE event.match_id = lineup.match_id
				AND event.type = 'substitution'
				AND event.secondary_person_id = lineup.person_id
			ORDER BY event.sequence
			LIMIT 1
		) left_match ON true
		WHERE lineup.match_id = $1 AND lineup.team_id IN ($2, $3)
		ORDER BY CASE lineup.team_id WHEN $2 THEN 0 ELSE 1 END,
			lineup.is_starter DESC, lineup.grid_position NULLS LAST,
			lineup.shirt_number NULLS LAST, person.display_name, person.id`, matchID, sides.home.ID, sides.away.ID)
	if err != nil {
		return football.MatchLineups{}, mapError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var teamID string
		var starter bool
		var player football.LineupPlayer
		var enteredEventID, leftEventID *string
		var enteredPeriod, leftPeriod *football.MatchPeriod
		var enteredMinute, enteredStoppage, leftMinute, leftStoppage *int16
		var replacedPlayerID, replacementPlayerID *string
		if err := rows.Scan(
			&teamID, &starter, &player.ShirtNumber, &player.LineupPosition,
			&player.GridPosition, &player.IsCaptain,
			&player.Player.ID, &player.Player.DisplayName, &player.Player.FirstName, &player.Player.LastName,
			&player.Player.BirthDate, &player.Player.CountryCode, &player.Player.PhotoURL,
			&player.Player.Position, &player.Player.DetailedPosition, &player.Player.PreferredFoot, &player.Player.HeightCM,
			&enteredEventID, &enteredPeriod, &enteredMinute, &enteredStoppage, &replacedPlayerID,
			&leftEventID, &leftPeriod, &leftMinute, &leftStoppage, &replacementPlayerID,
		); err != nil {
			return football.MatchLineups{}, mapError(err)
		}
		if enteredEventID != nil {
			player.SubstitutedIn = &football.SubstitutionDetail{
				Period: enteredPeriod, Minute: enteredMinute, StoppageMinute: enteredStoppage,
				ReplacedPlayerID: replacedPlayerID,
			}
		}
		if leftEventID != nil {
			player.SubstitutedOut = &football.SubstitutionDetail{
				Period: leftPeriod, Minute: leftMinute, StoppageMinute: leftStoppage,
				ReplacementPlayerID: replacementPlayerID,
			}
		}
		player.SubstitutionStatus = substitutionStatus(starter, player.SubstitutedIn != nil, player.SubstitutedOut != nil)

		lineup := lineupForTeam(&lineups, teamID)
		if lineup == nil {
			continue
		}
		if starter {
			lineup.Starters = append(lineup.Starters, player)
		} else {
			lineup.Substitutes = append(lineup.Substitutes, player)
		}
	}
	return lineups, mapError(rows.Err())
}

func lineupForTeam(lineups *football.MatchLineups, teamID string) *football.TeamLineup {
	switch teamID {
	case lineups.Home.Team.ID:
		return &lineups.Home
	case lineups.Away.Team.ID:
		return &lineups.Away
	default:
		return nil
	}
}

func substitutionStatus(starter, entered, left bool) string {
	switch {
	case entered && left:
		return "substituted_in_and_out"
	case entered:
		return "substituted_in"
	case left:
		return "substituted_out"
	case starter:
		return "not_substituted"
	default:
		return "unused_substitute"
	}
}

func (store *Store) GetMatchStatistics(ctx context.Context, matchID string) (football.MatchStatistics, error) {
	sides, err := store.getMatchReadSides(ctx, matchID)
	if err != nil {
		return football.MatchStatistics{}, err
	}
	statistics := football.MatchStatistics{
		MatchID: sides.matchID,
		Home:    football.TeamMatchStatistics{Team: sides.home, Players: []football.PlayerStatistics{}},
		Away:    football.TeamMatchStatistics{Team: sides.away, Players: []football.PlayerStatistics{}},
	}

	totalRows, err := store.pool.Query(ctx, `
		SELECT team_id, possession, shots, shots_on_target, shots_off_target, blocked_shots,
			shots_inside_box, shots_outside_box, corners, passes, passes_completed, pass_accuracy,
			fouls, offsides, yellow_cards, red_cards, saves, tackles, interceptions, clearances,
			expected_goals, metadata, updated_at
		FROM match_team_statistics
		WHERE match_id = $1 AND team_id IN ($2, $3)
		ORDER BY CASE team_id WHEN $2 THEN 0 ELSE 1 END`, matchID, sides.home.ID, sides.away.ID)
	if err != nil {
		return football.MatchStatistics{}, mapError(err)
	}
	for totalRows.Next() {
		var teamID string
		var totals football.MatchTeamTotals
		if err := totalRows.Scan(
			&teamID, &totals.Possession, &totals.Shots, &totals.ShotsOnTarget, &totals.ShotsOffTarget,
			&totals.BlockedShots, &totals.ShotsInsideBox, &totals.ShotsOutsideBox, &totals.Corners,
			&totals.Passes, &totals.PassesCompleted, &totals.PassAccuracy, &totals.Fouls, &totals.Offsides,
			&totals.YellowCards, &totals.RedCards, &totals.Saves, &totals.Tackles, &totals.Interceptions,
			&totals.Clearances, &totals.ExpectedGoals, &totals.Metadata, &totals.UpdatedAt,
		); err != nil {
			totalRows.Close()
			return football.MatchStatistics{}, mapError(err)
		}
		if side := statisticsForTeam(&statistics, teamID); side != nil {
			side.Totals = &totals
		}
	}
	if err := totalRows.Err(); err != nil {
		totalRows.Close()
		return football.MatchStatistics{}, mapError(err)
	}
	totalRows.Close()

	rows, err := store.pool.Query(ctx, `
		SELECT statistic.team_id,
			person.id, person.display_name, person.first_name, person.last_name,
			person.birth_date, person.country_code, person.photo_url,
			player.position, player.detailed_position, player.preferred_foot, player.height_cm,
			statistic.started, statistic.minutes_played, statistic.goals, statistic.assists,
			statistic.shots, statistic.shots_on_target, statistic.passes, statistic.passes_completed,
			statistic.key_passes, statistic.tackles, statistic.interceptions, statistic.clearances,
			statistic.blocks, statistic.duels, statistic.duels_won, statistic.saves,
			statistic.yellow_cards, statistic.red_cards, statistic.rating,
			statistic.expected_goals, statistic.expected_assists
		FROM player_match_statistics statistic
		JOIN people person ON person.id = statistic.person_id
		JOIN players player ON player.person_id = person.id
		WHERE statistic.match_id = $1 AND statistic.team_id IN ($2, $3)
		ORDER BY CASE statistic.team_id WHEN $2 THEN 0 ELSE 1 END,
			statistic.started DESC, statistic.minutes_played DESC, person.display_name, person.id`, matchID, sides.home.ID, sides.away.ID)
	if err != nil {
		return football.MatchStatistics{}, mapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var teamID string
		var statistic football.PlayerStatistics
		if err := rows.Scan(
			&teamID,
			&statistic.Player.ID, &statistic.Player.DisplayName, &statistic.Player.FirstName, &statistic.Player.LastName,
			&statistic.Player.BirthDate, &statistic.Player.CountryCode, &statistic.Player.PhotoURL,
			&statistic.Player.Position, &statistic.Player.DetailedPosition, &statistic.Player.PreferredFoot, &statistic.Player.HeightCM,
			&statistic.Started, &statistic.MinutesPlayed, &statistic.Goals, &statistic.Assists,
			&statistic.Shots, &statistic.ShotsOnTarget, &statistic.Passes, &statistic.PassesCompleted,
			&statistic.KeyPasses, &statistic.Tackles, &statistic.Interceptions, &statistic.Clearances,
			&statistic.Blocks, &statistic.Duels, &statistic.DuelsWon, &statistic.Saves,
			&statistic.YellowCards, &statistic.RedCards, &statistic.Rating,
			&statistic.ExpectedGoals, &statistic.ExpectedAssists,
		); err != nil {
			return football.MatchStatistics{}, mapError(err)
		}
		if side := statisticsForTeam(&statistics, teamID); side != nil {
			side.Players = append(side.Players, statistic)
		}
	}
	return statistics, mapError(rows.Err())
}

func statisticsForTeam(statistics *football.MatchStatistics, teamID string) *football.TeamMatchStatistics {
	switch teamID {
	case statistics.Home.Team.ID:
		return &statistics.Home
	case statistics.Away.Team.ID:
		return &statistics.Away
	default:
		return nil
	}
}

func (store *Store) ListSeasonStandings(ctx context.Context, seasonID string) (football.SeasonStandings, error) {
	var canonicalID string
	if err := store.pool.QueryRow(ctx, `SELECT id FROM seasons WHERE id = $1`, seasonID).Scan(&canonicalID); err != nil {
		return football.SeasonStandings{}, mapError(err)
	}
	standings := football.SeasonStandings{SeasonID: canonicalID, Data: []football.StandingEntry{}}
	rows, err := store.pool.Query(ctx, `
		SELECT standing.group_name, standing.position,
			team.id, team.name, team.short_name, team.code, team.logo_url, team.primary_color, team.secondary_color,
			standing.played, standing.won, standing.drawn, standing.lost,
			standing.goals_for, standing.goals_against, standing.points,
			standing.form, standing.zone, standing.description,
			standing.home_played, standing.home_won, standing.home_drawn, standing.home_lost,
			standing.away_played, standing.away_won, standing.away_drawn, standing.away_lost,
			standing.updated_at
		FROM standings standing
		JOIN teams team ON team.id = standing.team_id
		WHERE standing.season_id = $1
		ORDER BY standing.group_name, standing.position, team.name, team.id`, canonicalID)
	if err != nil {
		return football.SeasonStandings{}, mapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var entry football.StandingEntry
		var homePlayed, homeWon, homeDrawn, homeLost *int16
		var awayPlayed, awayWon, awayDrawn, awayLost *int16
		if err := rows.Scan(
			&entry.GroupName, &entry.Position,
			&entry.Team.ID, &entry.Team.Name, &entry.Team.ShortName, &entry.Team.Code, &entry.Team.LogoURL,
			&entry.Team.PrimaryColor, &entry.Team.SecondaryColor,
			&entry.Played, &entry.Won, &entry.Drawn, &entry.Lost,
			&entry.GoalsFor, &entry.GoalsAgainst, &entry.Points,
			&entry.Form, &entry.Zone, &entry.Description,
			&homePlayed, &homeWon, &homeDrawn, &homeLost,
			&awayPlayed, &awayWon, &awayDrawn, &awayLost,
			&entry.UpdatedAt,
		); err != nil {
			return football.SeasonStandings{}, mapError(err)
		}
		entry.GoalDifference = entry.GoalsFor - entry.GoalsAgainst
		if homePlayed != nil && homeWon != nil && homeDrawn != nil && homeLost != nil {
			entry.HomeRecord = &football.StandingRecord{
				Played: *homePlayed, Won: *homeWon, Drawn: *homeDrawn, Lost: *homeLost,
			}
		}
		if awayPlayed != nil && awayWon != nil && awayDrawn != nil && awayLost != nil {
			entry.AwayRecord = &football.StandingRecord{
				Played: *awayPlayed, Won: *awayWon, Drawn: *awayDrawn, Lost: *awayLost,
			}
		}
		standings.Data = append(standings.Data, entry)
	}
	return standings, mapError(rows.Err())
}

func (store *Store) ListMatchOfficials(ctx context.Context, matchID string) (football.MatchOfficials, error) {
	var canonicalID string
	if err := store.pool.QueryRow(ctx, `SELECT id FROM matches WHERE id = $1`, matchID).Scan(&canonicalID); err != nil {
		return football.MatchOfficials{}, mapError(err)
	}
	officials := football.MatchOfficials{MatchID: canonicalID, Data: []football.MatchOfficialDetail{}}
	rows, err := store.pool.Query(ctx, `
		SELECT official.role,
			person.id, person.display_name, person.first_name, person.last_name,
			person.birth_date, person.country_code, person.photo_url
		FROM match_officials official
		JOIN people person ON person.id = official.person_id
		WHERE official.match_id = $1
		ORDER BY CASE official.role
			WHEN 'referee' THEN 0
			WHEN 'assistant_referee' THEN 1
			WHEN 'fourth_official' THEN 2
			WHEN 'var' THEN 3
			WHEN 'assistant_var' THEN 4
			ELSE 5
		END, person.display_name, person.id`, canonicalID)
	if err != nil {
		return football.MatchOfficials{}, mapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var official football.MatchOfficialDetail
		if err := rows.Scan(
			&official.Role,
			&official.Person.ID, &official.Person.DisplayName, &official.Person.FirstName, &official.Person.LastName,
			&official.Person.BirthDate, &official.Person.CountryCode, &official.Person.PhotoURL,
		); err != nil {
			return football.MatchOfficials{}, mapError(err)
		}
		officials.Data = append(officials.Data, official)
	}
	return officials, mapError(rows.Err())
}

func (store *Store) Search(ctx context.Context, filter football.SearchFilter) ([]football.SearchResult, error) {
	includeAll := len(filter.Types) == 0
	includeLeague, includeTeam, includePlayer, includeCoach, includeFixture := includeAll, includeAll, includeAll, includeAll, includeAll
	for _, entityType := range filter.Types {
		switch entityType {
		case football.SearchLeague:
			includeLeague = true
		case football.SearchTeam:
			includeTeam = true
		case football.SearchPlayer:
			includePlayer = true
		case football.SearchCoach:
			includeCoach = true
		case football.SearchFixture:
			includeFixture = true
		}
	}

	rows, err := store.pool.Query(ctx, `
		WITH candidates AS (
			SELECT 'league'::text AS entity_type, league.id, league.name,
				league.logo_url AS image_url, league.country_code,
				NULL::text AS team_code, NULL::text AS position,
				league.id AS league_id, NULL::uuid AS season_id,
				NULL::timestamptz AS kickoff_at, NULL::text AS match_status,
				NULL::uuid AS home_team_id, NULL::text AS home_team_name,
				NULL::uuid AS away_team_id, NULL::text AS away_team_name,
				similarity(league.name, $1) AS relevance
			FROM leagues league
			WHERE $2 AND league.name ILIKE '%' || $1 || '%'
			UNION ALL
			SELECT 'team'::text AS entity_type, team.id, team.name,
				team.logo_url AS image_url, team.country_code, team.code AS team_code,
				NULL::text AS position, NULL::uuid, NULL::uuid, NULL::timestamptz,
				NULL::text, NULL::uuid, NULL::text, NULL::uuid, NULL::text,
				GREATEST(
					similarity(team.name, $1),
					similarity(COALESCE(team.short_name, ''), $1),
					similarity(COALESCE(team.code, ''), $1)
				) AS relevance
			FROM teams team
			WHERE $3 AND
				(team.name || ' ' || COALESCE(team.short_name, '') || ' ' || COALESCE(team.code, ''))
					ILIKE '%' || $1 || '%'
			UNION ALL
			SELECT 'player'::text, person.id, person.display_name,
				person.photo_url, person.country_code, NULL::text, player.position,
				NULL::uuid, NULL::uuid, NULL::timestamptz, NULL::text,
				NULL::uuid, NULL::text, NULL::uuid, NULL::text,
				similarity(person.display_name, $1)
			FROM people person
			JOIN players player ON player.person_id = person.id
			WHERE $4 AND person.display_name ILIKE '%' || $1 || '%'
			UNION ALL
			SELECT 'coach'::text, person.id, person.display_name,
				person.photo_url, person.country_code, NULL::text, NULL::text,
				NULL::uuid, NULL::uuid, NULL::timestamptz, NULL::text,
				NULL::uuid, NULL::text, NULL::uuid, NULL::text,
				similarity(person.display_name, $1)
			FROM people person
			JOIN coaches coach ON coach.person_id = person.id
			WHERE $5 AND person.display_name ILIKE '%' || $1 || '%'
			UNION ALL
			SELECT 'fixture'::text, match.id, home.name || ' v ' || away.name,
				league.logo_url, league.country_code, NULL::text, NULL::text,
				match.league_id, match.season_id, match.kickoff_at, match.status,
				home.id, home.name, away.id, away.name,
				similarity(
					home.name || ' ' || away.name || ' ' || league.name || ' ' || COALESCE(match.round, ''),
					$1
				)
			FROM matches match
			JOIN leagues league ON league.id = match.league_id
			JOIN teams home ON home.id = match.home_team_id
			JOIN teams away ON away.id = match.away_team_id
			WHERE $6 AND
				(home.name || ' ' || away.name || ' ' || league.name || ' ' || COALESCE(match.round, ''))
					ILIKE '%' || $1 || '%'
		)
		SELECT entity_type, id, name, image_url, country_code, team_code, position,
			league_id, season_id, kickoff_at, match_status,
			home_team_id, home_team_name, away_team_id, away_team_name
		FROM candidates
		ORDER BY (lower(name) = lower($1)) DESC,
			(lower(name) LIKE lower($1) || '%') DESC,
			relevance DESC, lower(name), entity_type, id
		LIMIT $7`, filter.Query, includeLeague, includeTeam, includePlayer, includeCoach, includeFixture, filter.Limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	results := make([]football.SearchResult, 0, filter.Limit)
	for rows.Next() {
		var result football.SearchResult
		if err := rows.Scan(
			&result.EntityType, &result.ID, &result.Name, &result.ImageURL,
			&result.CountryCode, &result.TeamCode, &result.Position,
			&result.LeagueID, &result.SeasonID, &result.KickoffAt, &result.MatchStatus,
			&result.HomeTeamID, &result.HomeTeamName, &result.AwayTeamID, &result.AwayTeamName,
		); err != nil {
			return nil, mapError(err)
		}
		results = append(results, result)
	}
	return results, mapError(rows.Err())
}

func (store *Store) ListHeadToHeadMatches(ctx context.Context, filter football.HeadToHeadFilter) ([]football.Match, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM matches
		WHERE LEAST(home_team_id, away_team_id) = LEAST($1::uuid, $2::uuid)
			AND GREATEST(home_team_id, away_team_id) = GREATEST($1::uuid, $2::uuid)
			AND status IN ('finished', 'awarded')
			AND home_score IS NOT NULL AND away_score IS NOT NULL
		ORDER BY kickoff_at DESC, id DESC
		LIMIT $3`, matchColumns)
	rows, err := store.pool.Query(ctx, query, filter.TeamAID, filter.TeamBID, filter.Limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	matches := make([]football.Match, 0, filter.Limit)
	for rows.Next() {
		match, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}
	return matches, mapError(rows.Err())
}
