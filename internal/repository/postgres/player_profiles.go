package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

const playerAggregateSelect = `
	COUNT(*)::integer,
	COUNT(*) FILTER (WHERE statistic.started)::integer,
	COALESCE(SUM(statistic.minutes_played), 0)::integer,
	COALESCE(SUM(statistic.goals), 0)::integer,
	COALESCE(SUM(statistic.assists), 0)::integer,
	COALESCE(SUM(statistic.shots), 0)::integer,
	COALESCE(SUM(statistic.shots_on_target), 0)::integer,
	COALESCE(SUM(statistic.passes), 0)::integer,
	SUM(statistic.passes_completed)::integer,
	CASE
		WHEN COUNT(statistic.passes_completed) = 0 OR SUM(statistic.passes) = 0 THEN NULL
		ELSE (SUM(statistic.passes_completed)::double precision / SUM(statistic.passes)::double precision) * 100
	END,
	SUM(statistic.key_passes)::integer,
	COALESCE(SUM(statistic.tackles), 0)::integer,
	SUM(statistic.interceptions)::integer,
	SUM(statistic.clearances)::integer,
	SUM(statistic.blocks)::integer,
	SUM(statistic.duels)::integer,
	SUM(statistic.duels_won)::integer,
	COALESCE(SUM(statistic.saves), 0)::integer,
	COALESCE(SUM(statistic.yellow_cards), 0)::integer,
	COALESCE(SUM(statistic.red_cards), 0)::integer,
	SUM(statistic.expected_goals)::double precision,
	SUM(statistic.expected_assists)::double precision,
	AVG(statistic.rating)::double precision,
	COUNT(*)::integer,
	COUNT(statistic.rating)::integer,
	COUNT(*) FILTER (WHERE
		statistic.passes_completed IS NOT NULL OR statistic.key_passes IS NOT NULL OR
		statistic.interceptions IS NOT NULL OR statistic.clearances IS NOT NULL OR
		statistic.blocks IS NOT NULL OR statistic.duels IS NOT NULL OR
		statistic.duels_won IS NOT NULL OR statistic.expected_goals IS NOT NULL OR
		statistic.expected_assists IS NOT NULL
	)::integer,
	MAX(statistic.updated_at)`

func playerAggregateDestinations(totals *football.PlayerPerformanceTotals, coverage *football.PlayerStatisticsCoverage, updatedAt **time.Time) []any {
	return []any{
		&totals.Appearances, &totals.Starts, &totals.MinutesPlayed, &totals.Goals, &totals.Assists,
		&totals.Shots, &totals.ShotsOnTarget, &totals.Passes, &totals.PassesCompleted, &totals.PassAccuracy,
		&totals.KeyPasses, &totals.Tackles, &totals.Interceptions, &totals.Clearances, &totals.Blocks,
		&totals.Duels, &totals.DuelsWon, &totals.Saves, &totals.YellowCards, &totals.RedCards,
		&totals.ExpectedGoals, &totals.ExpectedAssists, &totals.AverageRating,
		&coverage.Matches, &coverage.RatedMatches, &coverage.AdvancedMatches, updatedAt,
	}
}

func (store *Store) ensurePlayer(ctx context.Context, playerID string) error {
	var canonicalID string
	return mapError(store.pool.QueryRow(ctx, `SELECT person_id FROM players WHERE person_id = $1`, playerID).Scan(&canonicalID))
}

func teamType(national bool) string {
	if national {
		return "national"
	}
	return "club"
}

func (store *Store) ListPlayerMemberships(ctx context.Context, playerID string) (football.PlayerMemberships, error) {
	if err := store.ensurePlayer(ctx, playerID); err != nil {
		return football.PlayerMemberships{}, err
	}
	result := football.PlayerMemberships{PlayerID: playerID, Data: []football.PlayerMembership{}}
	rows, err := store.pool.Query(ctx, `
		SELECT membership.id,
			team.id, team.name, team.short_name, team.code, team.logo_url, team.primary_color, team.secondary_color,
			team.national, membership.role, membership.shirt_number, membership.starts_on, membership.ends_on,
			membership.is_loan, membership.transfer_type,
			parent.id, parent.name, parent.short_name, parent.code, parent.logo_url, parent.primary_color, parent.secondary_color,
			((membership.starts_on IS NULL OR membership.starts_on <= CURRENT_DATE) AND
			 (membership.ends_on IS NULL OR membership.ends_on >= CURRENT_DATE)) AS is_current
		FROM team_memberships membership
		JOIN teams team ON team.id = membership.team_id
		LEFT JOIN teams parent ON parent.id = membership.parent_team_id
		WHERE membership.person_id = $1
		ORDER BY is_current DESC, membership.starts_on DESC NULLS LAST,
			membership.ends_on DESC NULLS FIRST, team.name, membership.id`, playerID)
	if err != nil {
		return football.PlayerMemberships{}, mapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var membership football.PlayerMembership
		var national bool
		var parentID, parentName, parentShortName, parentCode, parentLogoURL, parentPrimary, parentSecondary *string
		if err := rows.Scan(
			&membership.ID,
			&membership.Team.ID, &membership.Team.Name, &membership.Team.ShortName, &membership.Team.Code,
			&membership.Team.LogoURL, &membership.Team.PrimaryColor, &membership.Team.SecondaryColor,
			&national, &membership.Role, &membership.ShirtNumber, &membership.StartsOn, &membership.EndsOn,
			&membership.IsLoan, &membership.TransferType,
			&parentID, &parentName, &parentShortName, &parentCode, &parentLogoURL, &parentPrimary, &parentSecondary,
			&membership.IsCurrent,
		); err != nil {
			return football.PlayerMemberships{}, mapError(err)
		}
		membership.TeamType = teamType(national)
		if parentID != nil {
			membership.ParentTeam = &football.TeamSummary{
				ID: *parentID, Name: valueOrEmpty(parentName), ShortName: parentShortName, Code: parentCode,
				LogoURL: parentLogoURL, PrimaryColor: parentPrimary, SecondaryColor: parentSecondary,
			}
		}
		result.Data = append(result.Data, membership)
	}
	return result, mapError(rows.Err())
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (store *Store) ListPlayerMatches(ctx context.Context, playerID string, filter football.PlayerMatchFilter) ([]football.PlayerMatchHistoryItem, error) {
	if err := store.ensurePlayer(ctx, playerID); err != nil {
		return nil, err
	}
	if filter.SeasonID != "" && filter.LeagueID != "" {
		if _, _, err := store.resolveSeasonLeague(ctx, filter.SeasonID, filter.LeagueID); err != nil {
			return nil, err
		}
	}
	query := strings.Builder{}
	query.WriteString(`
		SELECT match.id, match.league_id, match.season_id, match.stage, match.round, match.round_sort,
			match.group_name, match.leg, match.kickoff_at, match.status, match.period, match.elapsed_minute,
			match.venue_id, match.home_team_id, match.away_team_id, match.home_score, match.away_score,
			match.home_half_time_score, match.away_half_time_score, match.home_extra_time_score,
			match.away_extra_time_score, match.home_penalty_score, match.away_penalty_score,
			match.attendance, match.first_leg_match_id, match.winner_team_id, match.version, match.metadata,
			match.created_at, match.updated_at,
			team.id, team.name, team.short_name, team.code, team.logo_url, team.primary_color, team.secondary_color,
			opponent.id, opponent.name, opponent.short_name, opponent.code, opponent.logo_url,
			opponent.primary_color, opponent.secondary_color,
			CASE WHEN statistic.team_id = match.home_team_id THEN 'home' ELSE 'away' END,
			statistic.started, statistic.minutes_played, statistic.goals, statistic.assists,
			statistic.shots, statistic.shots_on_target, statistic.passes,
			statistic.yellow_cards, statistic.red_cards, statistic.rating,
			entered.id, entered.period, entered.minute, entered.stoppage_minute, entered.secondary_person_id,
			left_event.id, left_event.period, left_event.minute, left_event.stoppage_minute, left_event.primary_person_id
		FROM player_match_statistics statistic
		JOIN matches match ON match.id = statistic.match_id
		JOIN teams team ON team.id = statistic.team_id
		JOIN teams opponent ON opponent.id = CASE
			WHEN statistic.team_id = match.home_team_id THEN match.away_team_id ELSE match.home_team_id END
		LEFT JOIN LATERAL (
			SELECT event.id, event.period, event.minute, event.stoppage_minute, event.secondary_person_id
			FROM match_events event
			WHERE event.match_id = match.id AND event.type = 'substitution' AND event.primary_person_id = statistic.person_id
			ORDER BY event.sequence, event.id LIMIT 1
		) entered ON true
		LEFT JOIN LATERAL (
			SELECT event.id, event.period, event.minute, event.stoppage_minute, event.primary_person_id
			FROM match_events event
			WHERE event.match_id = match.id AND event.type = 'substitution' AND event.secondary_person_id = statistic.person_id
			ORDER BY event.sequence, event.id LIMIT 1
		) left_event ON true
		WHERE statistic.person_id = $1`)
	args := []any{playerID}
	add := func(format string, value any) {
		args = append(args, value)
		fmt.Fprintf(&query, format, len(args))
	}
	if filter.SeasonID != "" {
		add(" AND match.season_id = $%d", filter.SeasonID)
	}
	if filter.LeagueID != "" {
		add(" AND match.league_id = $%d", filter.LeagueID)
	}
	if filter.BeforeKickoff != nil {
		args = append(args, *filter.BeforeKickoff, filter.BeforeMatchID)
		fmt.Fprintf(&query, " AND (match.kickoff_at, match.id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.Limit)
	fmt.Fprintf(&query, " ORDER BY match.kickoff_at DESC, match.id DESC LIMIT $%d", len(args))

	rows, err := store.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	items := make([]football.PlayerMatchHistoryItem, 0, filter.Limit)
	for rows.Next() {
		var item football.PlayerMatchHistoryItem
		var enteredID, leftID *string
		var enteredPeriod, leftPeriod *football.MatchPeriod
		var enteredMinute, enteredStoppage, leftMinute, leftStoppage *int16
		var replacedPlayerID, replacementPlayerID *string
		if err := rows.Scan(
			&item.Fixture.ID, &item.Fixture.LeagueID, &item.Fixture.SeasonID, &item.Fixture.Stage,
			&item.Fixture.Round, &item.Fixture.RoundSort, &item.Fixture.GroupName, &item.Fixture.Leg,
			&item.Fixture.KickoffAt, &item.Fixture.Status, &item.Fixture.Period, &item.Fixture.ElapsedMinute,
			&item.Fixture.VenueID, &item.Fixture.HomeTeamID, &item.Fixture.AwayTeamID,
			&item.Fixture.HomeScore, &item.Fixture.AwayScore, &item.Fixture.HomeHTScore, &item.Fixture.AwayHTScore,
			&item.Fixture.HomeExtraTimeScore, &item.Fixture.AwayExtraTimeScore,
			&item.Fixture.HomePenaltyScore, &item.Fixture.AwayPenaltyScore, &item.Fixture.Attendance,
			&item.Fixture.FirstLegMatchID, &item.Fixture.WinnerTeamID, &item.Fixture.Version,
			&item.Fixture.Metadata, &item.Fixture.CreatedAt, &item.Fixture.UpdatedAt,
			&item.Team.ID, &item.Team.Name, &item.Team.ShortName, &item.Team.Code, &item.Team.LogoURL,
			&item.Team.PrimaryColor, &item.Team.SecondaryColor,
			&item.Opponent.ID, &item.Opponent.Name, &item.Opponent.ShortName, &item.Opponent.Code,
			&item.Opponent.LogoURL, &item.Opponent.PrimaryColor, &item.Opponent.SecondaryColor,
			&item.VenueSide, &item.Started, &item.Minutes, &item.Goals, &item.Assists, &item.Shots,
			&item.ShotsOnTarget, &item.Passes, &item.YellowCards, &item.RedCards, &item.Rating,
			&enteredID, &enteredPeriod, &enteredMinute, &enteredStoppage, &replacedPlayerID,
			&leftID, &leftPeriod, &leftMinute, &leftStoppage, &replacementPlayerID,
		); err != nil {
			return nil, mapError(err)
		}
		item.Result = playerMatchResult(item.Fixture, item.Team.ID)
		if enteredID != nil {
			item.Substitution.EnteredAt = &football.SubstitutionDetail{
				Period: enteredPeriod, Minute: enteredMinute, StoppageMinute: enteredStoppage,
				ReplacedPlayerID: replacedPlayerID,
			}
			item.Substitution.ReplacedPlayerID = replacedPlayerID
		}
		if leftID != nil {
			item.Substitution.LeftAt = &football.SubstitutionDetail{
				Period: leftPeriod, Minute: leftMinute, StoppageMinute: leftStoppage,
				ReplacementPlayerID: replacementPlayerID,
			}
			item.Substitution.ReplacementPlayerID = replacementPlayerID
		}
		items = append(items, item)
	}
	return items, mapError(rows.Err())
}

func playerMatchResult(match football.Match, teamID string) *string {
	if match.Status != football.MatchFinished && match.Status != football.MatchAwarded {
		return nil
	}
	if match.WinnerTeamID != nil {
		result := "loss"
		if *match.WinnerTeamID == teamID {
			result = "win"
		}
		return &result
	}
	if match.HomeScore == nil || match.AwayScore == nil {
		return nil
	}
	if *match.HomeScore == *match.AwayScore {
		result := "draw"
		return &result
	}
	winnerID := match.AwayTeamID
	if *match.HomeScore > *match.AwayScore {
		winnerID = match.HomeTeamID
	}
	result := "loss"
	if winnerID == teamID {
		result = "win"
	}
	return &result
}

const seasonLeagueSelect = `
	season.id, season.league_id, season.name, season.starts_on, season.ends_on, season.is_current,
	season.created_at, season.updated_at,
	league.id, league.name, league.slug, league.type, league.gender, league.country_code, league.logo_url,
	current_season.id, league.created_at, league.updated_at`

func scanSeasonLeague(row scanner) (football.Season, football.League, error) {
	var season football.Season
	var league football.League
	err := row.Scan(
		&season.ID, &season.LeagueID, &season.Name, &season.StartsOn, &season.EndsOn, &season.IsCurrent,
		&season.CreatedAt, &season.UpdatedAt,
		&league.ID, &league.Name, &league.Slug, &league.Type, &league.Gender, &league.CountryCode,
		&league.LogoURL, &league.CurrentSeasonID, &league.CreatedAt, &league.UpdatedAt,
	)
	return season, league, mapError(err)
}

func (store *Store) resolveSeasonLeague(ctx context.Context, seasonID, leagueID string) (football.Season, football.League, error) {
	if seasonID == "" && leagueID == "" {
		return football.Season{}, football.League{}, fmt.Errorf("%w: season_id or league_id is required", football.ErrInvalid)
	}
	base := ` FROM seasons season
		JOIN leagues league ON league.id = season.league_id
		LEFT JOIN seasons current_season ON current_season.league_id = league.id AND current_season.is_current `
	var season football.Season
	var league football.League
	var err error
	if seasonID != "" {
		season, league, err = scanSeasonLeague(store.pool.QueryRow(ctx, `SELECT `+seasonLeagueSelect+base+`WHERE season.id = $1`, seasonID))
	} else {
		season, league, err = scanSeasonLeague(store.pool.QueryRow(ctx, `SELECT `+seasonLeagueSelect+base+`WHERE season.league_id = $1 AND season.is_current`, leagueID))
	}
	if err != nil {
		return football.Season{}, football.League{}, err
	}
	if leagueID != "" && league.ID != leagueID {
		return football.Season{}, football.League{}, fmt.Errorf("%w: season_id does not belong to league_id", football.ErrInvalid)
	}
	return season, league, nil
}

func (store *Store) GetPlayerSeasonStatistics(ctx context.Context, playerID, seasonID, leagueID string) (football.PlayerSeasonStatistics, error) {
	if err := store.ensurePlayer(ctx, playerID); err != nil {
		return football.PlayerSeasonStatistics{}, err
	}
	season, league, err := store.resolveSeasonLeague(ctx, seasonID, leagueID)
	if err != nil {
		return football.PlayerSeasonStatistics{}, err
	}
	result := football.PlayerSeasonStatistics{
		PlayerID: playerID, Season: season, League: league, ByTeam: []football.PlayerTeamStatistics{},
	}
	args := playerAggregateDestinations(&result.Statistics, &result.Coverage, &result.UpdatedAt)
	if err := store.pool.QueryRow(ctx, `SELECT `+playerAggregateSelect+`
		FROM player_match_statistics statistic
		JOIN matches match ON match.id = statistic.match_id
		WHERE statistic.person_id = $1 AND match.season_id = $2`, playerID, season.ID).Scan(args...); err != nil {
		return football.PlayerSeasonStatistics{}, mapError(err)
	}

	rows, err := store.pool.Query(ctx, `
		SELECT team.id, team.name, team.short_name, team.code, team.logo_url, team.primary_color, team.secondary_color,
		`+playerAggregateSelect+`
		FROM player_match_statistics statistic
		JOIN matches match ON match.id = statistic.match_id
		JOIN teams team ON team.id = statistic.team_id
		WHERE statistic.person_id = $1 AND match.season_id = $2
		GROUP BY team.id, team.name, team.short_name, team.code, team.logo_url, team.primary_color, team.secondary_color
		ORDER BY COUNT(*) DESC, team.name, team.id`, playerID, season.ID)
	if err != nil {
		return football.PlayerSeasonStatistics{}, mapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var item football.PlayerTeamStatistics
		var ignoredUpdatedAt *time.Time
		destinations := []any{
			&item.Team.ID, &item.Team.Name, &item.Team.ShortName, &item.Team.Code, &item.Team.LogoURL,
			&item.Team.PrimaryColor, &item.Team.SecondaryColor,
		}
		destinations = append(destinations, playerAggregateDestinations(&item.Statistics, &item.Coverage, &ignoredUpdatedAt)...)
		if err := rows.Scan(destinations...); err != nil {
			return football.PlayerSeasonStatistics{}, mapError(err)
		}
		result.ByTeam = append(result.ByTeam, item)
	}
	return result, mapError(rows.Err())
}

func (store *Store) GetPlayerCareer(ctx context.Context, playerID string) (football.PlayerCareer, error) {
	memberships, err := store.ListPlayerMemberships(ctx, playerID)
	if err != nil {
		return football.PlayerCareer{}, err
	}
	career := football.PlayerCareer{PlayerID: playerID, Spells: make([]football.PlayerCareerSpell, 0, len(memberships.Data))}
	for _, membership := range memberships.Data {
		spell := football.PlayerCareerSpell{Membership: membership, Seasons: []football.PlayerCareerSeason{}}
		rows, err := store.pool.Query(ctx, `
			SELECT `+seasonLeagueSelect+`, `+playerAggregateSelect+`
			FROM player_match_statistics statistic
			JOIN matches match ON match.id = statistic.match_id
			JOIN seasons season ON season.id = match.season_id
			JOIN leagues league ON league.id = season.league_id
			LEFT JOIN seasons current_season ON current_season.league_id = league.id AND current_season.is_current
			WHERE statistic.person_id = $1 AND statistic.team_id = $2
				AND ($3::date IS NULL OR match.kickoff_at::date >= $3::date)
				AND ($4::date IS NULL OR match.kickoff_at::date <= $4::date)
			GROUP BY season.id, season.league_id, season.name, season.starts_on, season.ends_on, season.is_current,
				season.created_at, season.updated_at, league.id, league.name, league.slug, league.type,
				league.gender, league.country_code, league.logo_url, current_season.id, league.created_at, league.updated_at
			ORDER BY season.starts_on DESC, season.id`, playerID, membership.Team.ID, membership.StartsOn, membership.EndsOn)
		if err != nil {
			return football.PlayerCareer{}, mapError(err)
		}
		for rows.Next() {
			var item football.PlayerCareerSeason
			var ignoredUpdatedAt *time.Time
			destinations := []any{
				&item.Season.ID, &item.Season.LeagueID, &item.Season.Name, &item.Season.StartsOn,
				&item.Season.EndsOn, &item.Season.IsCurrent, &item.Season.CreatedAt, &item.Season.UpdatedAt,
				&item.League.ID, &item.League.Name, &item.League.Slug, &item.League.Type, &item.League.Gender,
				&item.League.CountryCode, &item.League.LogoURL, &item.League.CurrentSeasonID,
				&item.League.CreatedAt, &item.League.UpdatedAt,
			}
			destinations = append(destinations, playerAggregateDestinations(&item.Statistics, &item.Coverage, &ignoredUpdatedAt)...)
			if err := rows.Scan(destinations...); err != nil {
				rows.Close()
				return football.PlayerCareer{}, mapError(err)
			}
			spell.Seasons = append(spell.Seasons, item)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return football.PlayerCareer{}, mapError(err)
		}
		rows.Close()
		career.Spells = append(career.Spells, spell)
	}
	return career, nil
}

func (store *Store) ListPlayers(ctx context.Context, filter football.PlayerDiscoveryFilter) ([]football.PlayerDiscoveryResult, error) {
	var resolvedSeasonID string
	if filter.SeasonID != "" || filter.LeagueID != "" {
		season, _, err := store.resolveSeasonLeague(ctx, filter.SeasonID, filter.LeagueID)
		if err != nil {
			return nil, err
		}
		resolvedSeasonID = season.ID
	}
	query := strings.Builder{}
	query.WriteString(`SELECT person.id, person.display_name, person.first_name, person.last_name,
		person.birth_date, person.country_code, person.photo_url,
		player.position, player.detailed_position, player.preferred_foot, player.height_cm
		FROM players player JOIN people person ON person.id = player.person_id WHERE true`)
	args := make([]any, 0, 8)
	add := func(format string, value any) {
		args = append(args, value)
		fmt.Fprintf(&query, format, len(args))
	}
	if filter.Query != "" {
		add(" AND person.display_name ILIKE '%%' || $%d || '%%'", filter.Query)
	}
	if filter.Position != "" {
		add(" AND player.position = $%d", filter.Position)
	}
	if resolvedSeasonID != "" {
		args = append(args, resolvedSeasonID)
		seasonArg := len(args)
		teamClause := ""
		if filter.TeamID != "" {
			args = append(args, filter.TeamID)
			teamClause = fmt.Sprintf(" AND membership.team_id = $%d", len(args))
		}
		fmt.Fprintf(&query, ` AND EXISTS (
			SELECT 1 FROM team_memberships membership
			JOIN season_teams season_team ON season_team.team_id = membership.team_id
			JOIN seasons filter_season ON filter_season.id = season_team.season_id
			WHERE membership.person_id = person.id AND season_team.season_id = $%d%s
				AND (membership.starts_on IS NULL OR membership.starts_on <= filter_season.ends_on)
				AND (membership.ends_on IS NULL OR membership.ends_on >= filter_season.starts_on)
		)`, seasonArg, teamClause)
	} else if filter.TeamID != "" {
		args = append(args, filter.TeamID)
		fmt.Fprintf(&query, ` AND EXISTS (
			SELECT 1 FROM team_memberships membership
			WHERE membership.person_id = person.id AND membership.team_id = $%d
				AND (membership.starts_on IS NULL OR membership.starts_on <= CURRENT_DATE)
				AND (membership.ends_on IS NULL OR membership.ends_on >= CURRENT_DATE)
		)`, len(args))
	}
	if filter.AfterID != "" {
		args = append(args, strings.ToLower(filter.AfterName), filter.AfterID)
		fmt.Fprintf(&query, " AND (lower(person.display_name), person.id) > ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.Limit)
	fmt.Fprintf(&query, " ORDER BY lower(person.display_name), person.id LIMIT $%d", len(args))
	rows, err := store.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, mapError(err)
	}
	players := make([]football.Player, 0, filter.Limit)
	for rows.Next() {
		var player football.Player
		if err := rows.Scan(
			&player.ID, &player.DisplayName, &player.FirstName, &player.LastName, &player.BirthDate,
			&player.CountryCode, &player.PhotoURL, &player.Position, &player.DetailedPosition,
			&player.PreferredFoot, &player.HeightCM,
		); err != nil {
			rows.Close()
			return nil, mapError(err)
		}
		players = append(players, player)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, mapError(err)
	}
	rows.Close()

	results := make([]football.PlayerDiscoveryResult, 0, len(players))
	for _, player := range players {
		memberships, err := store.ListPlayerMemberships(ctx, player.ID)
		if err != nil {
			return nil, err
		}
		result := football.PlayerDiscoveryResult{Player: player, CurrentMemberships: []football.PlayerMembership{}}
		for _, membership := range memberships.Data {
			if membership.IsCurrent {
				result.CurrentMemberships = append(result.CurrentMemberships, membership)
			}
		}
		if resolvedSeasonID != "" {
			statistics, err := store.GetPlayerSeasonStatistics(ctx, player.ID, resolvedSeasonID, filter.LeagueID)
			if err != nil {
				return nil, err
			}
			result.SeasonStatistics = &statistics
		}
		results = append(results, result)
	}
	return results, nil
}

var comparisonMetrics = []string{
	"appearances", "starts", "minutes_played", "goals", "assists", "shots", "shots_on_target",
	"passes", "passes_completed", "pass_accuracy", "key_passes", "tackles", "interceptions",
	"clearances", "blocks", "duels", "duels_won", "saves", "yellow_cards", "red_cards",
	"expected_goals", "expected_assists", "average_rating",
}

func (store *Store) ComparePlayers(ctx context.Context, filter football.PlayerComparisonFilter) (football.PlayerComparison, error) {
	season, league, err := store.resolveSeasonLeague(ctx, filter.SeasonID, filter.LeagueID)
	if err != nil {
		return football.PlayerComparison{}, err
	}
	comparison := football.PlayerComparison{
		Season: season, League: league, Metrics: append([]string(nil), comparisonMetrics...),
		Players: []football.PlayerComparisonEntry{}, Compatible: true, Warnings: []string{},
	}
	for _, playerID := range filter.PlayerIDs {
		player, err := store.GetPlayer(ctx, playerID)
		if err != nil {
			return football.PlayerComparison{}, err
		}
		statistics, err := store.GetPlayerSeasonStatistics(ctx, playerID, season.ID, league.ID)
		if err != nil {
			return football.PlayerComparison{}, err
		}
		comparison.Players = append(comparison.Players, football.PlayerComparisonEntry{
			Player: player, Statistics: statistics.Statistics, Coverage: statistics.Coverage,
		})
		if statistics.Statistics.Appearances == 0 {
			comparison.Compatible = false
			comparison.Warnings = append(comparison.Warnings, fmt.Sprintf("player %s has no appearances in the selected season", playerID))
		}
	}
	return comparison, nil
}
