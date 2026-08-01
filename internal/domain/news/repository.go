package news

import (
	"context"
	"time"
)

type Filter struct {
	Category          Category
	Featured          *bool
	LeagueID          string
	TeamID            string
	MatchID           string
	Limit             int
	BeforePublishedAt *time.Time
	BeforeID          string
}

type Store interface {
	ListPublishedArticles(context.Context, Filter) ([]ArticleSummary, error)
	GetPublishedArticleBySlug(context.Context, string) (Article, error)
	UpsertArticle(context.Context, string, string, UpsertArticle) (Article, error)
}
