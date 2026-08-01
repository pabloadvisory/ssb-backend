package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

const weatherColumns = `id, match_id, source, external_id, kind, valid_at, issued_at, received_at,
	temperature_c, feels_like_c, humidity_percent, precipitation_probability_percent,
	precipitation_mm, wind_speed_kph, wind_gust_kph, wind_direction_degrees,
	pressure_hpa, visibility_km, condition_code, condition_text, icon_url, metadata`

func scanWeatherSnapshot(row scanner) (football.WeatherSnapshot, error) {
	var weather football.WeatherSnapshot
	err := row.Scan(
		&weather.ID, &weather.MatchID, &weather.Source, &weather.ExternalID,
		&weather.Kind, &weather.ValidAt, &weather.IssuedAt, &weather.ReceivedAt,
		&weather.TemperatureC, &weather.FeelsLikeC, &weather.HumidityPercent,
		&weather.PrecipitationProbabilityPercent, &weather.PrecipitationMM,
		&weather.WindSpeedKPH, &weather.WindGustKPH, &weather.WindDirectionDegrees,
		&weather.PressureHPA, &weather.VisibilityKM, &weather.ConditionCode,
		&weather.ConditionText, &weather.IconURL, &weather.Metadata,
	)
	return weather, mapError(err)
}

func (store *Store) GetMatchWeather(ctx context.Context, matchID string) (football.MatchWeather, error) {
	result := football.MatchWeather{MatchID: matchID}
	var kickoffAt time.Time
	if err := store.pool.QueryRow(ctx, `SELECT kickoff_at FROM matches WHERE id = $1`, matchID).Scan(&kickoffAt); err != nil {
		return football.MatchWeather{}, mapError(err)
	}

	forecast, err := scanWeatherSnapshot(store.pool.QueryRow(ctx, `
		SELECT `+weatherColumns+`
		FROM match_weather_snapshots
		WHERE match_id = $1 AND kind = 'forecast'
		ORDER BY abs(extract(epoch FROM (valid_at - $2::timestamptz))),
			issued_at DESC, id DESC
		LIMIT 1`, matchID, kickoffAt))
	if err == nil {
		result.Forecast = &forecast
	} else if !errors.Is(err, football.ErrNotFound) {
		return football.MatchWeather{}, err
	}

	observation, err := scanWeatherSnapshot(store.pool.QueryRow(ctx, `
		SELECT `+weatherColumns+`
		FROM match_weather_snapshots
		WHERE match_id = $1 AND kind = 'observed'
		ORDER BY valid_at DESC, issued_at DESC, id DESC
		LIMIT 1`, matchID))
	if err == nil {
		result.Observation = &observation
	} else if !errors.Is(err, football.ErrNotFound) {
		return football.MatchWeather{}, err
	}

	return result, nil
}

func (store *Store) UpsertMatchWeather(
	ctx context.Context,
	matchID, source, externalID string,
	command football.UpsertWeatherSnapshot,
) (football.WeatherSnapshot, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return football.WeatherSnapshot{}, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureFootballMatch(ctx, tx, matchID); err != nil {
		return football.WeatherSnapshot{}, err
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended('weather:' || $1 || ':' || $2, 0))`, source, externalID); err != nil {
		return football.WeatherSnapshot{}, mapError(err)
	}

	var snapshotID, existingMatchID string
	err = tx.QueryRow(ctx, `
		SELECT id, match_id FROM match_weather_snapshots
		WHERE source = $1 AND external_id = $2
		FOR UPDATE`, source, externalID).Scan(&snapshotID, &existingMatchID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `
			INSERT INTO match_weather_snapshots (
				match_id, source, external_id, kind, valid_at, issued_at,
				temperature_c, feels_like_c, humidity_percent,
				precipitation_probability_percent, precipitation_mm,
				wind_speed_kph, wind_gust_kph, wind_direction_degrees,
				pressure_hpa, visibility_km, condition_code, condition_text, icon_url, metadata
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
			RETURNING id`, matchID, source, externalID, command.Kind, command.ValidAt,
			command.IssuedAt, command.TemperatureC, command.FeelsLikeC,
			command.HumidityPercent, command.PrecipitationProbabilityPercent,
			command.PrecipitationMM, command.WindSpeedKPH, command.WindGustKPH,
			command.WindDirectionDegrees, command.PressureHPA, command.VisibilityKM,
			command.ConditionCode, command.ConditionText, command.IconURL,
			repositoryMetadata(command.Metadata)).Scan(&snapshotID)
		if err != nil {
			return football.WeatherSnapshot{}, mapError(err)
		}
	case err != nil:
		return football.WeatherSnapshot{}, mapError(err)
	case existingMatchID != matchID:
		return football.WeatherSnapshot{}, fmt.Errorf(
			"%w: weather source and external_id belong to another match", football.ErrConflict,
		)
	default:
		_, err = tx.Exec(ctx, `
			UPDATE match_weather_snapshots SET
				kind = $2, valid_at = $3, issued_at = $4, received_at = now(),
				temperature_c = $5, feels_like_c = $6, humidity_percent = $7,
				precipitation_probability_percent = $8, precipitation_mm = $9,
				wind_speed_kph = $10, wind_gust_kph = $11, wind_direction_degrees = $12,
				pressure_hpa = $13, visibility_km = $14, condition_code = $15,
				condition_text = $16, icon_url = $17, metadata = $18
			WHERE id = $1`, snapshotID, command.Kind, command.ValidAt, command.IssuedAt,
			command.TemperatureC, command.FeelsLikeC, command.HumidityPercent,
			command.PrecipitationProbabilityPercent, command.PrecipitationMM,
			command.WindSpeedKPH, command.WindGustKPH, command.WindDirectionDegrees,
			command.PressureHPA, command.VisibilityKM, command.ConditionCode,
			command.ConditionText, command.IconURL, repositoryMetadata(command.Metadata))
		if err != nil {
			return football.WeatherSnapshot{}, mapError(err)
		}
	}

	snapshot, err := scanWeatherSnapshot(tx.QueryRow(ctx, `
		SELECT `+weatherColumns+` FROM match_weather_snapshots WHERE id = $1`, snapshotID))
	if err != nil {
		return football.WeatherSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return football.WeatherSnapshot{}, mapError(err)
	}
	return snapshot, nil
}
