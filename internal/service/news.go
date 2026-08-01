package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/pabloadvisory/ssb-backend/internal/domain/news"
)

type News struct {
	store news.Store
}

func NewNews(store news.Store) *News {
	return &News{store: store}
}

func (service *News) ListPublishedArticles(ctx context.Context, filter news.Filter) ([]news.ArticleSummary, error) {
	if filter.Category != "" && !filter.Category.Valid() {
		return nil, fmt.Errorf("%w: unknown category", news.ErrInvalid)
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	if filter.Limit < 1 || filter.Limit > 101 {
		return nil, fmt.Errorf("%w: limit must be between 1 and 101", news.ErrInvalid)
	}
	return service.store.ListPublishedArticles(ctx, filter)
}

func (service *News) GetPublishedArticleBySlug(ctx context.Context, slug string) (news.Article, error) {
	slug = strings.TrimSpace(slug)
	if !validSlug(slug) {
		return news.Article{}, fmt.Errorf("%w: slug is invalid", news.ErrInvalid)
	}
	return service.store.GetPublishedArticleBySlug(ctx, slug)
}

func (service *News) UpsertArticle(ctx context.Context, source, externalID string, command news.UpsertArticle) (news.Article, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	externalID = strings.TrimSpace(externalID)
	if source == "" || len(source) > 64 || externalID == "" || len(externalID) > 256 {
		return news.Article{}, fmt.Errorf("%w: source and external_id are required", news.ErrInvalid)
	}
	for _, character := range source {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' && character != '_' {
			return news.Article{}, fmt.Errorf("%w: source contains unsupported characters", news.ErrInvalid)
		}
	}

	command.Slug = strings.TrimSpace(command.Slug)
	command.Title = strings.TrimSpace(command.Title)
	command.Summary = strings.TrimSpace(command.Summary)
	command.BodyMarkdown = strings.TrimSpace(command.BodyMarkdown)
	if command.Status == "" {
		command.Status = news.StatusDraft
	}
	if command.Category == "" {
		command.Category = news.CategoryStory
	}
	if !validSlug(command.Slug) {
		return news.Article{}, fmt.Errorf("%w: slug must contain lowercase letters, numbers, and single hyphens", news.ErrInvalid)
	}
	if !withinRunes(command.Title, 1, 200) {
		return news.Article{}, fmt.Errorf("%w: title must be between 1 and 200 characters", news.ErrInvalid)
	}
	if !withinRunes(command.Summary, 1, 500) {
		return news.Article{}, fmt.Errorf("%w: summary must be between 1 and 500 characters", news.ErrInvalid)
	}
	if !withinRunes(command.BodyMarkdown, 1, 200_000) {
		return news.Article{}, fmt.Errorf("%w: body_markdown must be between 1 and 200000 characters", news.ErrInvalid)
	}
	if !command.Category.Valid() {
		return news.Article{}, fmt.Errorf("%w: unknown category", news.ErrInvalid)
	}
	if !command.Status.Valid() {
		return news.Article{}, fmt.Errorf("%w: unknown status", news.ErrInvalid)
	}
	if command.Status == news.StatusPublished && command.PublishedAt == nil {
		return news.Article{}, fmt.Errorf("%w: published_at is required for published articles", news.ErrInvalid)
	}
	if command.HeroImageURL != nil {
		normalized := strings.TrimSpace(*command.HeroImageURL)
		if normalized == "" || len(normalized) > 2048 || !validHTTPURL(normalized) {
			return news.Article{}, fmt.Errorf("%w: hero_image_url must be an absolute HTTP(S) URL", news.ErrInvalid)
		}
		command.HeroImageURL = &normalized
	}
	if err := normalizeOptional(&command.HeroImageAlt, 300, "hero_image_alt"); err != nil {
		return news.Article{}, err
	}
	if err := normalizeOptional(&command.AuthorName, 120, "author_name"); err != nil {
		return news.Article{}, err
	}
	return service.store.UpsertArticle(ctx, source, externalID, command)
}

func validSlug(value string) bool {
	if len(value) < 1 || len(value) > 160 || strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
		return false
	}
	previousHyphen := false
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9':
			previousHyphen = false
		case character == '-' && !previousHyphen:
			previousHyphen = true
		default:
			return false
		}
	}
	return true
}

func withinRunes(value string, minimum, maximum int) bool {
	count := utf8.RuneCountInString(value)
	return count >= minimum && count <= maximum
}

func validHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func normalizeOptional(value **string, maximum int, field string) error {
	if *value == nil {
		return nil
	}
	normalized := strings.TrimSpace(**value)
	if normalized == "" {
		*value = nil
		return nil
	}
	if !withinRunes(normalized, 1, maximum) {
		return fmt.Errorf("%w: %s must not exceed %d characters", news.ErrInvalid, field, maximum)
	}
	*value = &normalized
	return nil
}
