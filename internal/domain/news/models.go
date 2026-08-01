package news

import "time"

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

func (status Status) Valid() bool {
	switch status {
	case StatusDraft, StatusPublished, StatusArchived:
		return true
	default:
		return false
	}
}

type Category string

const (
	CategoryStory        Category = "story"
	CategoryMatchReport  Category = "match_report"
	CategoryAnnouncement Category = "announcement"
)

func (category Category) Valid() bool {
	switch category {
	case CategoryStory, CategoryMatchReport, CategoryAnnouncement:
		return true
	default:
		return false
	}
}

// ArticleSummary is the feed representation. Article bodies are intentionally
// omitted so list responses stay small on mobile connections.
type ArticleSummary struct {
	ID              string     `json:"id"`
	Slug            string     `json:"slug"`
	Title           string     `json:"title"`
	Summary         string     `json:"summary"`
	HeroImageURL    *string    `json:"hero_image_url,omitempty"`
	HeroImageAlt    *string    `json:"hero_image_alt,omitempty"`
	AuthorName      *string    `json:"author_name,omitempty"`
	Category        Category   `json:"category"`
	Featured        bool       `json:"featured"`
	RelatedLeagueID *string    `json:"related_league_id,omitempty"`
	RelatedTeamID   *string    `json:"related_team_id,omitempty"`
	RelatedMatchID  *string    `json:"related_match_id,omitempty"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	Version         int64      `json:"version"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Article struct {
	ArticleSummary
	BodyMarkdown string    `json:"body_markdown"`
	Status       Status    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// UpsertArticle is a complete editorial representation. PublishedAt is
// required for published articles, which keeps retries deterministic and also
// allows future timestamps to schedule publication.
type UpsertArticle struct {
	Slug            string     `json:"slug"`
	Title           string     `json:"title"`
	Summary         string     `json:"summary"`
	BodyMarkdown    string     `json:"body_markdown"`
	HeroImageURL    *string    `json:"hero_image_url,omitempty"`
	HeroImageAlt    *string    `json:"hero_image_alt,omitempty"`
	AuthorName      *string    `json:"author_name,omitempty"`
	Category        Category   `json:"category"`
	Featured        bool       `json:"featured"`
	RelatedLeagueID *string    `json:"related_league_id,omitempty"`
	RelatedTeamID   *string    `json:"related_team_id,omitempty"`
	RelatedMatchID  *string    `json:"related_match_id,omitempty"`
	Status          Status     `json:"status"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
}
