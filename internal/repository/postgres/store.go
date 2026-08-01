package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
	"github.com/pabloadvisory/ssb-backend/internal/eventing"
	"github.com/pabloadvisory/ssb-backend/internal/notification"
	"github.com/pabloadvisory/ssb-backend/internal/realtime"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type scanner interface {
	Scan(...any) error
}

func scanLeague(row scanner) (football.League, error) {
	var league football.League
	err := row.Scan(
		&league.ID, &league.Name, &league.Slug, &league.Type, &league.Gender,
		&league.CountryCode, &league.LogoURL, &league.CurrentSeasonID,
		&league.CreatedAt, &league.UpdatedAt,
	)
	return league, mapError(err)
}

func (store *Store) ListLeagues(ctx context.Context, filter football.LeagueFilter) ([]football.League, error) {
	query := strings.Builder{}
	query.WriteString(`SELECT league.id, league.name, league.slug, league.type, league.gender,
		league.country_code, league.logo_url, current_season.id, league.created_at, league.updated_at
		FROM leagues league
		LEFT JOIN seasons current_season
			ON current_season.league_id = league.id AND current_season.is_current
		WHERE true`)
	args := make([]any, 0, 4)
	if filter.CountryCode != "" {
		args = append(args, strings.ToUpper(filter.CountryCode))
		fmt.Fprintf(&query, " AND league.country_code = $%d", len(args))
	}
	if filter.AfterID != "" {
		args = append(args, filter.AfterName, filter.AfterID)
		fmt.Fprintf(&query, " AND (league.name, league.id) > ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.Limit)
	fmt.Fprintf(&query, " ORDER BY league.name, league.id LIMIT $%d", len(args))

	rows, err := store.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	leagues := make([]football.League, 0, filter.Limit)
	for rows.Next() {
		league, err := scanLeague(rows)
		if err != nil {
			return nil, err
		}
		leagues = append(leagues, league)
	}
	return leagues, mapError(rows.Err())
}

func (store *Store) GetLeague(ctx context.Context, id string) (football.League, error) {
	return scanLeague(store.pool.QueryRow(ctx, `
		SELECT league.id, league.name, league.slug, league.type, league.gender,
			league.country_code, league.logo_url, current_season.id,
			league.created_at, league.updated_at
		FROM leagues league
		LEFT JOIN seasons current_season
			ON current_season.league_id = league.id AND current_season.is_current
		WHERE league.id = $1`, id))
}

func (store *Store) ListLeagueSeasons(ctx context.Context, leagueID string) (football.LeagueSeasons, error) {
	var canonicalID string
	if err := store.pool.QueryRow(ctx, `SELECT id FROM leagues WHERE id = $1`, leagueID).Scan(&canonicalID); err != nil {
		return football.LeagueSeasons{}, mapError(err)
	}
	result := football.LeagueSeasons{LeagueID: canonicalID, Data: []football.Season{}}
	rows, err := store.pool.Query(ctx, `
		SELECT id, league_id, name, starts_on, ends_on, is_current, created_at, updated_at
		FROM seasons
		WHERE league_id = $1
		ORDER BY is_current DESC, starts_on DESC, ends_on DESC, id`, canonicalID)
	if err != nil {
		return football.LeagueSeasons{}, mapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var season football.Season
		if err := rows.Scan(
			&season.ID, &season.LeagueID, &season.Name, &season.StartsOn, &season.EndsOn,
			&season.IsCurrent, &season.CreatedAt, &season.UpdatedAt,
		); err != nil {
			return football.LeagueSeasons{}, mapError(err)
		}
		result.Data = append(result.Data, season)
	}
	return result, mapError(rows.Err())
}

func (store *Store) GetTeam(ctx context.Context, id string) (football.Team, error) {
	var team football.Team
	var venueJSON []byte
	err := store.pool.QueryRow(ctx, `
		SELECT t.id, t.name, t.short_name, t.code, t.country_code, t.founded_year, t.logo_url,
			t.primary_color, t.secondary_color,
			CASE WHEN v.id IS NULL THEN NULL ELSE jsonb_build_object(
				'id', v.id, 'name', v.name, 'city', v.city, 'country_code', v.country_code,
				'country_name', country.name, 'address', v.address, 'latitude', v.latitude,
				'longitude', v.longitude, 'capacity', v.capacity, 'surface', v.surface,
				'image_url', v.image_url, 'timezone', v.timezone,
				'created_at', v.created_at, 'updated_at', v.updated_at
			) END,
			t.created_at, t.updated_at
		FROM teams t
		LEFT JOIN venues v ON v.id = t.venue_id
		LEFT JOIN countries country ON country.code = v.country_code
		WHERE t.id = $1`, id).Scan(
		&team.ID, &team.Name, &team.ShortName, &team.Code, &team.CountryCode, &team.FoundedYear, &team.LogoURL,
		&team.PrimaryColor, &team.SecondaryColor,
		&venueJSON, &team.CreatedAt, &team.UpdatedAt,
	)
	if err != nil {
		return football.Team{}, mapError(err)
	}
	if len(venueJSON) > 0 {
		var venue football.Venue
		if err := json.Unmarshal(venueJSON, &venue); err != nil {
			return football.Team{}, fmt.Errorf("decode venue: %w", err)
		}
		team.Venue = &venue
	}
	return team, nil
}

func scanPerson(row scanner, target *football.Person) error {
	return mapError(row.Scan(&target.ID, &target.DisplayName, &target.FirstName, &target.LastName, &target.BirthDate, &target.CountryCode, &target.PhotoURL))
}

func (store *Store) GetPlayer(ctx context.Context, id string) (football.Player, error) {
	var player football.Player
	err := store.pool.QueryRow(ctx, `
		SELECT p.id, p.display_name, p.first_name, p.last_name, p.birth_date, p.country_code, p.photo_url,
			pl.position, pl.detailed_position, pl.preferred_foot, pl.height_cm
		FROM people p JOIN players pl ON pl.person_id = p.id WHERE p.id = $1`, id).Scan(
		&player.ID, &player.DisplayName, &player.FirstName, &player.LastName, &player.BirthDate, &player.CountryCode, &player.PhotoURL,
		&player.Position, &player.DetailedPosition, &player.PreferredFoot, &player.HeightCM,
	)
	return player, mapError(err)
}

func (store *Store) GetCoach(ctx context.Context, id string) (football.Coach, error) {
	var coach football.Coach
	err := scanPerson(store.pool.QueryRow(ctx, `
		SELECT p.id, p.display_name, p.first_name, p.last_name, p.birth_date, p.country_code, p.photo_url
		FROM people p JOIN coaches c ON c.person_id = p.id WHERE p.id = $1`, id), &coach.Person)
	return coach, err
}

const matchColumns = `id, league_id, season_id, stage, round, round_sort, group_name, leg, kickoff_at, status, period, elapsed_minute,
	venue_id, home_team_id, away_team_id, home_score, away_score, home_half_time_score, away_half_time_score,
	home_extra_time_score, away_extra_time_score, home_penalty_score, away_penalty_score, attendance,
	first_leg_match_id, winner_team_id, version, metadata, created_at, updated_at`

func scanMatch(row scanner) (football.Match, error) {
	var match football.Match
	err := row.Scan(
		&match.ID, &match.LeagueID, &match.SeasonID, &match.Stage, &match.Round, &match.RoundSort, &match.GroupName, &match.Leg, &match.KickoffAt,
		&match.Status, &match.Period, &match.ElapsedMinute, &match.VenueID, &match.HomeTeamID, &match.AwayTeamID,
		&match.HomeScore, &match.AwayScore, &match.HomeHTScore, &match.AwayHTScore,
		&match.HomeExtraTimeScore, &match.AwayExtraTimeScore, &match.HomePenaltyScore, &match.AwayPenaltyScore,
		&match.Attendance, &match.FirstLegMatchID, &match.WinnerTeamID,
		&match.Version, &match.Metadata, &match.CreatedAt, &match.UpdatedAt,
	)
	return match, mapError(err)
}

func (store *Store) ListMatches(ctx context.Context, filter football.MatchFilter) ([]football.Match, error) {
	query := strings.Builder{}
	query.WriteString("SELECT " + matchColumns + " FROM matches WHERE true")
	args := make([]any, 0, 10)
	add := func(clause string, value any) {
		args = append(args, value)
		fmt.Fprintf(&query, clause, len(args))
	}
	if filter.LeagueID != "" {
		add(" AND league_id = $%d", filter.LeagueID)
	}
	if filter.SeasonID != "" {
		add(" AND season_id = $%d", filter.SeasonID)
	}
	if filter.TeamID != "" {
		args = append(args, filter.TeamID)
		fmt.Fprintf(&query, " AND (home_team_id = $%d OR away_team_id = $%d)", len(args), len(args))
	}
	if filter.Status != "" {
		add(" AND status = $%d", filter.Status)
	}
	if filter.From != nil {
		add(" AND kickoff_at >= $%d", *filter.From)
	}
	if filter.To != nil {
		add(" AND kickoff_at <= $%d", *filter.To)
	}
	if filter.AfterKickoff != nil {
		args = append(args, *filter.AfterKickoff, filter.AfterMatchID)
		fmt.Fprintf(&query, " AND (kickoff_at, id) > ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.Limit)
	fmt.Fprintf(&query, " ORDER BY kickoff_at, id LIMIT $%d", len(args))

	rows, err := store.pool.Query(ctx, query.String(), args...)
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

func (store *Store) GetMatch(ctx context.Context, id string) (football.Match, error) {
	return scanMatch(store.pool.QueryRow(ctx, "SELECT "+matchColumns+" FROM matches WHERE id = $1", id))
}

func (store *Store) ListMatchEvents(ctx context.Context, matchID string, filter football.EventFilter) ([]football.MatchEvent, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, match_id, sequence, period, minute, stoppage_minute, type, team_id,
			primary_person_id, secondary_person_id, detail, home_score, away_score, metadata, occurred_at, created_at
		FROM match_events WHERE match_id = $1 AND sequence > $2 ORDER BY sequence LIMIT $3`, matchID, filter.AfterSequence, filter.Limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	events := make([]football.MatchEvent, 0, filter.Limit)
	for rows.Next() {
		var event football.MatchEvent
		if err := rows.Scan(
			&event.ID, &event.MatchID, &event.Sequence, &event.Period, &event.Minute, &event.StoppageMinute,
			&event.Type, &event.TeamID, &event.PrimaryPersonID, &event.SecondaryPersonID, &event.Detail,
			&event.HomeScore, &event.AwayScore, &event.Metadata, &event.OccurredAt, &event.CreatedAt,
		); err != nil {
			return nil, mapError(err)
		}
		events = append(events, event)
	}
	return events, mapError(rows.Err())
}

func (store *Store) UpsertMatchSnapshot(ctx context.Context, provider, externalID string, snapshot football.MatchSnapshot) (football.Match, error) {
	serialized, err := json.Marshal(snapshot)
	if err != nil {
		return football.Match{}, fmt.Errorf("encode match snapshot: %w", err)
	}
	hash := sha256.Sum256(serialized)

	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return football.Match{}, fmt.Errorf("begin match upsert: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1 || ':' || $2, 0))`, provider, externalID); err != nil {
		return football.Match{}, mapError(err)
	}

	var matchID string
	var previousStatus football.MatchStatus
	var previousHomeScore, previousAwayScore *int16
	existed := true
	if err := tx.QueryRow(ctx, `
		SELECT match.id, match.status, match.home_score, match.away_score
		FROM external_ids external
		JOIN matches match ON match.id = external.entity_id
		WHERE external.provider = $1 AND external.entity_type = 'match' AND external.external_id = $2`, provider, externalID).Scan(
		&matchID, &previousStatus, &previousHomeScore, &previousAwayScore,
	); errors.Is(err, pgx.ErrNoRows) {
		existed = false
	} else if err != nil {
		return football.Match{}, mapError(err)
	}

	metadata := snapshot.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	var match football.Match
	changed := false
	if !existed {
		match, err = scanMatch(tx.QueryRow(ctx, `
			INSERT INTO matches (
				league_id, season_id, stage, round, round_sort, group_name, leg, kickoff_at, status, period,
				elapsed_minute, venue_id, home_team_id, away_team_id, home_score, away_score,
				home_half_time_score, away_half_time_score, home_extra_time_score, away_extra_time_score,
				home_penalty_score, away_penalty_score, attendance, first_leg_match_id, winner_team_id, source_hash, metadata
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
			RETURNING `+matchColumns,
			snapshot.LeagueID, snapshot.SeasonID, snapshot.Stage, snapshot.Round, snapshot.RoundSort, snapshot.GroupName,
			snapshot.Leg, snapshot.KickoffAt, snapshot.Status, snapshot.Period, snapshot.ElapsedMinute, snapshot.VenueID,
			snapshot.HomeTeamID, snapshot.AwayTeamID, snapshot.HomeScore, snapshot.AwayScore,
			snapshot.HomeHTScore, snapshot.AwayHTScore, snapshot.HomeExtraTimeScore, snapshot.AwayExtraTimeScore,
			snapshot.HomePenaltyScore, snapshot.AwayPenaltyScore, snapshot.Attendance, snapshot.FirstLegMatchID,
			snapshot.WinnerTeamID, hash[:], metadata,
		))
		if err != nil {
			return football.Match{}, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO external_ids (entity_type, entity_id, provider, external_id)
			VALUES ('match', $1, $2, $3)`, match.ID, provider, externalID); err != nil {
			return football.Match{}, mapError(err)
		}
		changed = true
	} else {
		match, err = scanMatch(tx.QueryRow(ctx, `
			UPDATE matches SET
				league_id=$2, season_id=$3, stage=$4, round=$5, round_sort=$6, group_name=$7, leg=$8,
				kickoff_at=$9, status=$10, period=$11, elapsed_minute=$12, venue_id=$13,
				home_team_id=$14, away_team_id=$15, home_score=$16, away_score=$17,
				home_half_time_score=$18, away_half_time_score=$19,
				home_extra_time_score=$20, away_extra_time_score=$21,
				home_penalty_score=$22, away_penalty_score=$23, attendance=$24,
				first_leg_match_id=$25, winner_team_id=$26, source_hash=$27, metadata=$28,
				version=version+1, updated_at=now()
			WHERE id=$1 AND source_hash IS DISTINCT FROM $27
			RETURNING `+matchColumns,
			matchID, snapshot.LeagueID, snapshot.SeasonID, snapshot.Stage, snapshot.Round, snapshot.RoundSort,
			snapshot.GroupName, snapshot.Leg, snapshot.KickoffAt, snapshot.Status, snapshot.Period,
			snapshot.ElapsedMinute, snapshot.VenueID, snapshot.HomeTeamID, snapshot.AwayTeamID,
			snapshot.HomeScore, snapshot.AwayScore, snapshot.HomeHTScore, snapshot.AwayHTScore,
			snapshot.HomeExtraTimeScore, snapshot.AwayExtraTimeScore, snapshot.HomePenaltyScore,
			snapshot.AwayPenaltyScore, snapshot.Attendance, snapshot.FirstLegMatchID, snapshot.WinnerTeamID,
			hash[:], metadata,
		))
		if err == nil {
			changed = true
		} else if errors.Is(err, football.ErrNotFound) {
			match, err = scanMatch(tx.QueryRow(ctx, "SELECT "+matchColumns+" FROM matches WHERE id = $1", matchID))
		}
		if err != nil {
			return football.Match{}, err
		}
	}

	if changed {
		for _, event := range snapshot.Events {
			eventMetadata := event.Metadata
			if len(eventMetadata) == 0 {
				eventMetadata = json.RawMessage(`{}`)
			}
			var eventID string
			if err := tx.QueryRow(ctx, `
				INSERT INTO match_events (
					match_id, sequence, period, minute, stoppage_minute, type, team_id,
					primary_person_id, secondary_person_id, detail, home_score, away_score, metadata, occurred_at
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
				ON CONFLICT (match_id, sequence) DO UPDATE SET
					period = EXCLUDED.period, minute = EXCLUDED.minute, stoppage_minute = EXCLUDED.stoppage_minute,
					type = EXCLUDED.type, team_id = EXCLUDED.team_id,
					primary_person_id = EXCLUDED.primary_person_id, secondary_person_id = EXCLUDED.secondary_person_id,
					detail = EXCLUDED.detail, home_score = EXCLUDED.home_score, away_score = EXCLUDED.away_score,
					metadata = EXCLUDED.metadata, occurred_at = EXCLUDED.occurred_at, updated_at = now()
				RETURNING id`,
				match.ID, event.Sequence, event.Period, event.Minute, event.StoppageMinute, event.Type,
				event.TeamID, event.PrimaryPersonID, event.SecondaryPersonID, event.Detail,
				event.HomeScore, event.AwayScore, eventMetadata, event.OccurredAt,
			).Scan(&eventID); err != nil {
				return football.Match{}, mapError(err)
			}
			if event.ExternalID != "" {
				if _, err := tx.Exec(ctx, `
					DELETE FROM external_ids
					WHERE entity_type = 'match_event' AND entity_id = $1 AND provider = $2 AND external_id <> $3`,
					eventID, provider, event.ExternalID,
				); err != nil {
					return football.Match{}, mapError(err)
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO external_ids (entity_type, entity_id, provider, external_id)
					VALUES ('match_event', $1, $2, $3)
					ON CONFLICT (provider, entity_type, external_id) DO UPDATE
					SET entity_id = EXCLUDED.entity_id, updated_at = now()`, eventID, provider, event.ExternalID); err != nil {
					return football.Match{}, mapError(err)
				}
			}
		}

		var homeTeamName, awayTeamName string
		if err := tx.QueryRow(ctx, `
			SELECT home.name, away.name FROM teams home, teams away
			WHERE home.id = $1 AND away.id = $2`, match.HomeTeamID, match.AwayTeamID).Scan(&homeTeamName, &awayTeamName); err != nil {
			return football.Match{}, mapError(err)
		}
		changedEvent := eventing.MatchChanged{
			Realtime: realtime.Update{MatchID: match.ID, Type: "match.updated", Version: match.Version},
			Notification: notification.MatchChange{
				Existed: existed,
				Previous: notification.MatchState{
					Status: previousStatus, HomeScore: previousHomeScore, AwayScore: previousAwayScore,
				},
				Current: notification.MatchUpdate{
					MatchID: match.ID, Version: match.Version, Status: match.Status,
					HomeTeamID: match.HomeTeamID, HomeTeamName: homeTeamName,
					AwayTeamID: match.AwayTeamID, AwayTeamName: awayTeamName,
					HomeScore: match.HomeScore, AwayScore: match.AwayScore,
					ElapsedMinute: match.ElapsedMinute, Period: match.Period, KickoffAtUnix: match.KickoffAt.Unix(),
				},
			},
		}
		payload, err := json.Marshal(changedEvent)
		if err != nil {
			return football.Match{}, fmt.Errorf("encode match changed event: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
			VALUES ('match', $1, $2, $3::jsonb)`, match.ID, eventing.MatchChangedV1, payload); err != nil {
			return football.Match{}, mapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return football.Match{}, mapError(err)
	}
	return match, nil
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return football.ErrNotFound
	}
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		switch databaseError.Code {
		case "22P02", "22007", "23503", "23514":
			return fmt.Errorf("%w: %s", football.ErrInvalid, databaseError.ConstraintName)
		case "23505":
			return fmt.Errorf("%w: %s", football.ErrConflict, databaseError.ConstraintName)
		}
	}
	return err
}

var _ football.Store = (*Store)(nil)
