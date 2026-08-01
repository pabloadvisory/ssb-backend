package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func (service *Football) ListMatchOdds(ctx context.Context, matchID, bookmakerSlug string) (football.MatchOdds, error) {
	return service.store.ListMatchOdds(ctx, matchID, strings.ToLower(strings.TrimSpace(bookmakerSlug)))
}

func (service *Football) UpsertMatchOdds(
	ctx context.Context,
	matchID, source, externalID string,
	command football.UpsertOddsSnapshot,
) (football.MatchOddsSnapshot, error) {
	source, externalID, err := normalizedSourceIdentity(source, externalID)
	if err != nil {
		return football.MatchOddsSnapshot{}, err
	}
	command.BookmakerSlug = strings.ToLower(strings.TrimSpace(command.BookmakerSlug))
	command.BookmakerName = strings.TrimSpace(command.BookmakerName)
	if command.BookmakerSlug == "" || command.BookmakerName == "" || !slugLike(command.BookmakerSlug) {
		return football.MatchOddsSnapshot{}, fmt.Errorf("%w: bookmaker_slug and bookmaker_name are required", football.ErrInvalid)
	}
	if command.ObservedAt.IsZero() || (command.ValidUntil != nil && command.ValidUntil.Before(command.ObservedAt)) {
		return football.MatchOddsSnapshot{}, fmt.Errorf("%w: odds timestamps are invalid", football.ErrInvalid)
	}
	if err := validateHTTPSURL(command.BookmakerLogoURL); err != nil {
		return football.MatchOddsSnapshot{}, err
	}
	if err := validateHTTPSURL(command.BookmakerWebsiteURL); err != nil {
		return football.MatchOddsSnapshot{}, err
	}
	if len(command.Markets) == 0 {
		return football.MatchOddsSnapshot{}, fmt.Errorf("%w: at least one odds market is required", football.ErrInvalid)
	}
	seenMarkets := make(map[string]struct{}, len(command.Markets))
	for marketIndex := range command.Markets {
		market := &command.Markets[marketIndex]
		market.Key = strings.TrimSpace(market.Key)
		market.Name = strings.TrimSpace(market.Name)
		if market.Status == "" {
			market.Status = "open"
		}
		if market.Key == "" || market.Name == "" || (market.Status != "open" && market.Status != "suspended" && market.Status != "settled") {
			return football.MatchOddsSnapshot{}, fmt.Errorf("%w: markets[%d] is invalid", football.ErrInvalid, marketIndex)
		}
		if _, duplicate := seenMarkets[market.Key]; duplicate {
			return football.MatchOddsSnapshot{}, fmt.Errorf("%w: market keys must be unique", football.ErrInvalid)
		}
		seenMarkets[market.Key] = struct{}{}
		if len(market.Selections) == 0 {
			return football.MatchOddsSnapshot{}, fmt.Errorf("%w: markets[%d] needs a selection", football.ErrInvalid, marketIndex)
		}
		for selectionIndex := range market.Selections {
			selection := &market.Selections[selectionIndex]
			selection.Key = strings.TrimSpace(selection.Key)
			selection.Name = strings.TrimSpace(selection.Name)
			if selection.Key == "" || selection.Name == "" || selection.DecimalOdds <= 1 {
				return football.MatchOddsSnapshot{}, fmt.Errorf("%w: markets[%d].selections[%d] is invalid", football.ErrInvalid, marketIndex, selectionIndex)
			}
			if selection.Result != nil {
				switch *selection.Result {
				case "won", "lost", "push", "void":
				default:
					return football.MatchOddsSnapshot{}, fmt.Errorf("%w: odds selection result is invalid", football.ErrInvalid)
				}
			}
			if err := normalizeMetadata(&selection.Metadata); err != nil {
				return football.MatchOddsSnapshot{}, err
			}
		}
		if err := normalizeMetadata(&market.Metadata); err != nil {
			return football.MatchOddsSnapshot{}, err
		}
	}
	if err := normalizeMetadata(&command.Metadata); err != nil {
		return football.MatchOddsSnapshot{}, err
	}
	return service.store.UpsertMatchOdds(ctx, matchID, source, externalID, command)
}

func (service *Football) ListMatchBroadcasts(ctx context.Context, matchID, countryCode string) (football.MatchBroadcasts, error) {
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	if countryCode != "" && !isoCountryCode(countryCode) {
		return football.MatchBroadcasts{}, fmt.Errorf("%w: country_code must be a two-letter ISO code", football.ErrInvalid)
	}
	return service.store.ListMatchBroadcasts(ctx, matchID, countryCode)
}

func (service *Football) UpsertMatchBroadcast(
	ctx context.Context,
	matchID, source, externalID string,
	command football.UpsertMatchBroadcast,
) (football.MatchBroadcast, error) {
	source, externalID, err := normalizedSourceIdentity(source, externalID)
	if err != nil {
		return football.MatchBroadcast{}, err
	}
	command.NetworkName = strings.TrimSpace(command.NetworkName)
	if command.NetworkName == "" || command.ObservedAt.IsZero() {
		return football.MatchBroadcast{}, fmt.Errorf("%w: network_name and observed_at are required", football.ErrInvalid)
	}
	switch command.Kind {
	case football.BroadcastTelevision, football.BroadcastRadio, football.BroadcastStream:
	default:
		return football.MatchBroadcast{}, fmt.Errorf("%w: broadcast kind is invalid", football.ErrInvalid)
	}
	switch command.AvailabilityScope {
	case football.BroadcastGlobal, football.BroadcastTerritorial, football.BroadcastUnknown:
	default:
		return football.MatchBroadcast{}, fmt.Errorf("%w: availability_scope is invalid", football.ErrInvalid)
	}
	switch command.Status {
	case football.BroadcastScheduled, football.BroadcastLive, football.BroadcastEnded, football.BroadcastCancelled, football.BroadcastUnavailable:
	default:
		return football.MatchBroadcast{}, fmt.Errorf("%w: broadcast status is invalid", football.ErrInvalid)
	}
	if command.StartsAt != nil && command.EndsAt != nil && command.EndsAt.Before(*command.StartsAt) {
		return football.MatchBroadcast{}, fmt.Errorf("%w: ends_at must not be before starts_at", football.ErrInvalid)
	}
	if err := validateHTTPSURL(command.WebURL); err != nil {
		return football.MatchBroadcast{}, err
	}
	if err := validateHTTPSURL(command.DeepLinkURL); err != nil {
		return football.MatchBroadcast{}, err
	}
	regions := make([]string, 0, len(command.Regions))
	seenRegions := make(map[string]struct{}, len(command.Regions))
	for _, region := range command.Regions {
		region = strings.ToUpper(strings.TrimSpace(region))
		if !isoCountryCode(region) {
			return football.MatchBroadcast{}, fmt.Errorf("%w: broadcast regions must be two-letter ISO codes", football.ErrInvalid)
		}
		if _, duplicate := seenRegions[region]; !duplicate {
			seenRegions[region] = struct{}{}
			regions = append(regions, region)
		}
	}
	if command.AvailabilityScope == football.BroadcastTerritorial && len(regions) == 0 {
		return football.MatchBroadcast{}, fmt.Errorf("%w: territorial broadcasts require regions", football.ErrInvalid)
	}
	command.Regions = regions
	if command.LanguageTags == nil {
		command.LanguageTags = []string{}
	}
	if err := normalizeMetadata(&command.Metadata); err != nil {
		return football.MatchBroadcast{}, err
	}
	return service.store.UpsertMatchBroadcast(ctx, matchID, source, externalID, command)
}

func (service *Football) GetMatchWeather(ctx context.Context, matchID string) (football.MatchWeather, error) {
	return service.store.GetMatchWeather(ctx, matchID)
}

func (service *Football) UpsertMatchWeather(
	ctx context.Context,
	matchID, source, externalID string,
	command football.UpsertWeatherSnapshot,
) (football.WeatherSnapshot, error) {
	source, externalID, err := normalizedSourceIdentity(source, externalID)
	if err != nil {
		return football.WeatherSnapshot{}, err
	}
	if command.Kind != football.WeatherForecast && command.Kind != football.WeatherObserved {
		return football.WeatherSnapshot{}, fmt.Errorf("%w: weather kind is invalid", football.ErrInvalid)
	}
	if command.ValidAt.IsZero() || command.IssuedAt.IsZero() {
		return football.WeatherSnapshot{}, fmt.Errorf("%w: valid_at and issued_at are required", football.ErrInvalid)
	}
	for name, value := range map[string]*float64{
		"humidity_percent":                  command.HumidityPercent,
		"precipitation_probability_percent": command.PrecipitationProbabilityPercent,
	} {
		if value != nil && (*value < 0 || *value > 100) {
			return football.WeatherSnapshot{}, fmt.Errorf("%w: %s must be between 0 and 100", football.ErrInvalid, name)
		}
	}
	for name, value := range map[string]*float64{
		"precipitation_mm": command.PrecipitationMM,
		"wind_speed_kph":   command.WindSpeedKPH,
		"wind_gust_kph":    command.WindGustKPH,
		"visibility_km":    command.VisibilityKM,
	} {
		if value != nil && *value < 0 {
			return football.WeatherSnapshot{}, fmt.Errorf("%w: %s must not be negative", football.ErrInvalid, name)
		}
	}
	if command.PressureHPA != nil && *command.PressureHPA <= 0 {
		return football.WeatherSnapshot{}, fmt.Errorf("%w: pressure_hpa must be positive", football.ErrInvalid)
	}
	if command.WindDirectionDegrees != nil && (*command.WindDirectionDegrees < 0 || *command.WindDirectionDegrees > 359) {
		return football.WeatherSnapshot{}, fmt.Errorf("%w: wind_direction_degrees must be between 0 and 359", football.ErrInvalid)
	}
	if err := validateHTTPSURL(command.IconURL); err != nil {
		return football.WeatherSnapshot{}, err
	}
	if err := normalizeMetadata(&command.Metadata); err != nil {
		return football.WeatherSnapshot{}, err
	}
	return service.store.UpsertMatchWeather(ctx, matchID, source, externalID, command)
}

func (service *Football) GetMatchPrediction(ctx context.Context, matchID string) (football.MatchPrediction, error) {
	return service.store.GetMatchPrediction(ctx, matchID)
}

func (service *Football) GetInstallationPrediction(ctx context.Context, installationID, credential, matchID string) (football.MatchPrediction, error) {
	hash := sha256.Sum256([]byte(credential))
	return service.store.GetInstallationPrediction(ctx, installationID, hash[:], matchID)
}

func (service *Football) SetInstallationPrediction(
	ctx context.Context,
	installationID, credential, matchID string,
	selection football.PredictionSelection,
) (football.MatchPrediction, error) {
	if selection != football.PredictionHome && selection != football.PredictionDraw && selection != football.PredictionAway {
		return football.MatchPrediction{}, fmt.Errorf("%w: selection must be home, draw, or away", football.ErrInvalid)
	}
	hash := sha256.Sum256([]byte(credential))
	return service.store.SetInstallationPrediction(ctx, installationID, hash[:], matchID, selection)
}

func (service *Football) ReplaceMatchCoverage(ctx context.Context, matchID, source string, command football.MatchCoverageUpdate) error {
	source, _, err := normalizedSourceIdentity(source, "dataset")
	if err != nil {
		return err
	}
	if command.TeamInfo == nil && command.Lineups == nil && command.TeamStatistics == nil && command.PlayerStatistics == nil && command.Officials == nil {
		return fmt.Errorf("%w: at least one coverage dataset is required", football.ErrInvalid)
	}
	if command.TeamInfo != nil {
		for index := range *command.TeamInfo {
			item := &(*command.TeamInfo)[index]
			if item.TeamID == "" {
				return fmt.Errorf("%w: team_info[%d].team_id is required", football.ErrInvalid, index)
			}
			if err := normalizeMetadata(&item.Metadata); err != nil {
				return err
			}
		}
	}
	if command.Lineups != nil {
		for index := range *command.Lineups {
			item := &(*command.Lineups)[index]
			if item.TeamID == "" || item.PersonID == "" || (item.ShirtNumber != nil && (*item.ShirtNumber < 0 || *item.ShirtNumber > 99)) {
				return fmt.Errorf("%w: lineups[%d] is invalid", football.ErrInvalid, index)
			}
			if err := normalizeMetadata(&item.Metadata); err != nil {
				return err
			}
		}
	}
	if command.TeamStatistics != nil {
		for index := range *command.TeamStatistics {
			item := &(*command.TeamStatistics)[index]
			if item.TeamID == "" {
				return fmt.Errorf("%w: team_statistics[%d].team_id is required", football.ErrInvalid, index)
			}
			if err := normalizeMetadata(&item.Metadata); err != nil {
				return err
			}
		}
	}
	if command.PlayerStatistics != nil {
		for index := range *command.PlayerStatistics {
			item := &(*command.PlayerStatistics)[index]
			if item.TeamID == "" || item.PersonID == "" || item.MinutesPlayed < 0 || item.MinutesPlayed > 200 || item.Goals < 0 || item.Assists < 0 || item.Shots < 0 || item.ShotsOnTarget < 0 || item.Passes < 0 || item.Tackles < 0 || item.Saves < 0 || item.YellowCards < 0 || item.RedCards < 0 || (item.Rating != nil && (*item.Rating < 0 || *item.Rating > 10)) {
				return fmt.Errorf("%w: player_statistics[%d] is invalid", football.ErrInvalid, index)
			}
			if err := normalizeMetadata(&item.Metadata); err != nil {
				return err
			}
		}
	}
	if command.Officials != nil {
		for index := range *command.Officials {
			item := &(*command.Officials)[index]
			if item.PersonID == "" || !validOfficialRole(item.Role) {
				return fmt.Errorf("%w: officials[%d] is invalid", football.ErrInvalid, index)
			}
			if err := normalizeMetadata(&item.Metadata); err != nil {
				return err
			}
		}
	}
	return service.store.ReplaceMatchCoverage(ctx, matchID, source, command)
}

func (service *Football) ReplaceSeasonStandings(ctx context.Context, seasonID, source string, command football.StandingsUpdate) error {
	source, _, err := normalizedSourceIdentity(source, "dataset")
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(command.Data))
	for index, row := range command.Data {
		key := row.TeamID + "\x00" + row.GroupName
		if row.TeamID == "" || row.Position < 1 || row.Played < 0 || row.Won < 0 || row.Drawn < 0 || row.Lost < 0 || row.GoalsFor < 0 || row.GoalsAgainst < 0 || row.Points < 0 {
			return fmt.Errorf("%w: data[%d] is invalid", football.ErrInvalid, index)
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("%w: standing team/group pairs must be unique", football.ErrInvalid)
		}
		seen[key] = struct{}{}
	}
	return service.store.ReplaceSeasonStandings(ctx, seasonID, source, command)
}

func normalizedSourceIdentity(source, externalID string) (string, string, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	externalID = strings.TrimSpace(externalID)
	if source == "" || len(source) > 64 || externalID == "" || len(externalID) > 256 {
		return "", "", fmt.Errorf("%w: source and external_id are required", football.ErrInvalid)
	}
	for _, character := range source {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' && character != '_' {
			return "", "", fmt.Errorf("%w: source contains unsupported characters", football.ErrInvalid)
		}
	}
	return source, externalID, nil
}

func normalizeMetadata(metadata *json.RawMessage) error {
	if len(*metadata) == 0 {
		*metadata = json.RawMessage(`{}`)
	}
	if !json.Valid(*metadata) {
		return fmt.Errorf("%w: metadata must be valid JSON", football.ErrInvalid)
	}
	return nil
}

func validateHTTPSURL(value *string) error {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("%w: URLs must use absolute https URLs without embedded credentials", football.ErrInvalid)
	}
	*value = trimmed
	return nil
}

func isoCountryCode(value string) bool {
	return len(value) == 2 && unicode.IsUpper(rune(value[0])) && unicode.IsUpper(rune(value[1]))
}

func slugLike(value string) bool {
	if value == "" || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	return !strings.Contains(value, "--")
}

func validOfficialRole(role football.MatchOfficialRole) bool {
	switch role {
	case football.OfficialReferee, football.OfficialAssistantReferee, football.OfficialFourth, football.OfficialVAR, football.OfficialAssistantVAR:
		return true
	default:
		return false
	}
}
