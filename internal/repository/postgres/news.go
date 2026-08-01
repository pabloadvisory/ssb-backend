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
	"github.com/pabloadvisory/ssb-backend/internal/domain/news"
)

const articleSummaryColumns = `id, slug, title, summary, hero_image_url, hero_image_alt, author_name, category, featured,
	related_league_id, related_team_id, related_match_id, published_at, version, updated_at`

const articleColumns = articleSummaryColumns + `, body_markdown, status, created_at`

func scanArticleSummary(row scanner) (news.ArticleSummary, error) {
	var article news.ArticleSummary
	err := row.Scan(
		&article.ID, &article.Slug, &article.Title, &article.Summary,
		&article.HeroImageURL, &article.HeroImageAlt, &article.AuthorName,
		&article.Category, &article.Featured, &article.RelatedLeagueID,
		&article.RelatedTeamID, &article.RelatedMatchID, &article.PublishedAt,
		&article.Version, &article.UpdatedAt,
	)
	return article, mapNewsError(err)
}

func scanArticle(row scanner) (news.Article, error) {
	var article news.Article
	err := row.Scan(
		&article.ID, &article.Slug, &article.Title, &article.Summary,
		&article.HeroImageURL, &article.HeroImageAlt, &article.AuthorName,
		&article.Category, &article.Featured, &article.RelatedLeagueID,
		&article.RelatedTeamID, &article.RelatedMatchID, &article.PublishedAt,
		&article.Version, &article.UpdatedAt, &article.BodyMarkdown,
		&article.Status, &article.CreatedAt,
	)
	return article, mapNewsError(err)
}

func (store *Store) ListPublishedArticles(ctx context.Context, filter news.Filter) ([]news.ArticleSummary, error) {
	query := strings.Builder{}
	query.WriteString("SELECT " + articleSummaryColumns + " FROM news_articles WHERE status = 'published' AND published_at <= now()")
	args := make([]any, 0, 8)
	add := func(clause string, value any) {
		args = append(args, value)
		fmt.Fprintf(&query, clause, len(args))
	}
	if filter.Category != "" {
		add(" AND category = $%d", filter.Category)
	}
	if filter.Featured != nil {
		add(" AND featured = $%d", *filter.Featured)
	}
	if filter.LeagueID != "" {
		add(" AND related_league_id = $%d", filter.LeagueID)
	}
	if filter.TeamID != "" {
		add(" AND related_team_id = $%d", filter.TeamID)
	}
	if filter.MatchID != "" {
		add(" AND related_match_id = $%d", filter.MatchID)
	}
	if filter.BeforePublishedAt != nil {
		args = append(args, *filter.BeforePublishedAt, filter.BeforeID)
		fmt.Fprintf(&query, " AND (published_at, id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.Limit)
	fmt.Fprintf(&query, " ORDER BY published_at DESC, id DESC LIMIT $%d", len(args))

	rows, err := store.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, mapNewsError(err)
	}
	defer rows.Close()

	articles := make([]news.ArticleSummary, 0, filter.Limit)
	for rows.Next() {
		article, err := scanArticleSummary(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	return articles, mapNewsError(rows.Err())
}

func (store *Store) GetPublishedArticleBySlug(ctx context.Context, slug string) (news.Article, error) {
	return scanArticle(store.pool.QueryRow(ctx,
		"SELECT "+articleColumns+" FROM news_articles WHERE slug = $1 AND status = 'published' AND published_at <= now()",
		slug,
	))
}

func (store *Store) UpsertArticle(ctx context.Context, source, externalID string, command news.UpsertArticle) (news.Article, error) {
	serialized, err := json.Marshal(command)
	if err != nil {
		return news.Article{}, fmt.Errorf("encode news article: %w", err)
	}
	hash := sha256.Sum256(serialized)

	article, err := scanArticle(store.pool.QueryRow(ctx, `
		INSERT INTO news_articles (
			source, external_id, slug, title, summary, body_markdown, hero_image_url,
			hero_image_alt, author_name, category, featured, related_league_id,
			related_team_id, related_match_id, status, published_at, source_hash
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (source, external_id) DO UPDATE SET
			slug=EXCLUDED.slug, title=EXCLUDED.title, summary=EXCLUDED.summary,
			body_markdown=EXCLUDED.body_markdown, hero_image_url=EXCLUDED.hero_image_url,
			hero_image_alt=EXCLUDED.hero_image_alt, author_name=EXCLUDED.author_name,
			category=EXCLUDED.category, featured=EXCLUDED.featured,
			related_league_id=EXCLUDED.related_league_id,
			related_team_id=EXCLUDED.related_team_id,
			related_match_id=EXCLUDED.related_match_id, status=EXCLUDED.status,
			published_at=EXCLUDED.published_at, source_hash=EXCLUDED.source_hash,
			version=news_articles.version+1, updated_at=now()
		WHERE news_articles.source_hash IS DISTINCT FROM EXCLUDED.source_hash
		RETURNING `+articleColumns,
		source, externalID, command.Slug, command.Title, command.Summary,
		command.BodyMarkdown, command.HeroImageURL, command.HeroImageAlt,
		command.AuthorName, command.Category, command.Featured,
		command.RelatedLeagueID, command.RelatedTeamID, command.RelatedMatchID,
		command.Status, command.PublishedAt, hash[:],
	))
	if err == nil {
		return article, nil
	}
	if !errors.Is(err, news.ErrNotFound) {
		return news.Article{}, err
	}
	return scanArticle(store.pool.QueryRow(ctx,
		"SELECT "+articleColumns+" FROM news_articles WHERE source = $1 AND external_id = $2",
		source, externalID,
	))
}

func mapNewsError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return news.ErrNotFound
	}
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		switch databaseError.Code {
		case "22P02", "22007", "23503", "23514":
			return fmt.Errorf("%w: %s", news.ErrInvalid, databaseError.ConstraintName)
		case "23505":
			return fmt.Errorf("%w: %s", news.ErrConflict, databaseError.ConstraintName)
		}
	}
	return err
}

var _ news.Store = (*Store)(nil)
