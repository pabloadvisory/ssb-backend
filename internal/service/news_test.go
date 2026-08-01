package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/domain/news"
)

type newsStoreStub struct {
	command news.UpsertArticle
}

func (*newsStoreStub) ListPublishedArticles(context.Context, news.Filter) ([]news.ArticleSummary, error) {
	return nil, nil
}

func (*newsStoreStub) GetPublishedArticleBySlug(context.Context, string) (news.Article, error) {
	return news.Article{}, news.ErrNotFound
}

func (store *newsStoreStub) UpsertArticle(_ context.Context, _, _ string, command news.UpsertArticle) (news.Article, error) {
	store.command = command
	return news.Article{ArticleSummary: news.ArticleSummary{Slug: command.Slug}}, nil
}

func TestNewsUpsertDefaultsToSafeDraft(t *testing.T) {
	t.Parallel()

	store := &newsStoreStub{}
	article, err := NewNews(store).UpsertArticle(context.Background(), " CMS ", "story-1", news.UpsertArticle{
		Slug: "season-opener", Title: " Season opener ", Summary: " Preview ", BodyMarkdown: " Body ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if article.Slug != "season-opener" || store.command.Status != news.StatusDraft || store.command.Category != news.CategoryStory {
		t.Fatalf("unexpected normalized command: %+v", store.command)
	}
	if store.command.Title != "Season opener" || store.command.Summary != "Preview" || store.command.BodyMarkdown != "Body" {
		t.Fatalf("editorial whitespace was not normalized: %+v", store.command)
	}
}

func TestNewsPublishedArticleRequiresExplicitPublicationTime(t *testing.T) {
	t.Parallel()

	_, err := NewNews(&newsStoreStub{}).UpsertArticle(context.Background(), "cms", "story-1", news.UpsertArticle{
		Slug: "season-opener", Title: "Season opener", Summary: "Preview", BodyMarkdown: "Body",
		Status: news.StatusPublished,
	})
	if !errors.Is(err, news.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestNewsUpsertAcceptsScheduledPublication(t *testing.T) {
	t.Parallel()

	publishedAt := time.Now().UTC().Add(time.Hour)
	_, err := NewNews(&newsStoreStub{}).UpsertArticle(context.Background(), "cms", "story-1", news.UpsertArticle{
		Slug: "season-opener", Title: "Season opener", Summary: "Preview", BodyMarkdown: "Body",
		Status: news.StatusPublished, PublishedAt: &publishedAt, Category: news.CategoryAnnouncement,
	})
	if err != nil {
		t.Fatalf("scheduled article rejected: %v", err)
	}
}

func TestNewsRejectsInvalidSlugAndImageURL(t *testing.T) {
	t.Parallel()

	imageURL := "javascript:alert(1)"
	service := NewNews(&newsStoreStub{})
	for name, command := range map[string]news.UpsertArticle{
		"slug": {
			Slug: "Season Opener", Title: "Season opener", Summary: "Preview", BodyMarkdown: "Body",
		},
		"image": {
			Slug: "season-opener", Title: "Season opener", Summary: "Preview", BodyMarkdown: "Body", HeroImageURL: &imageURL,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := service.UpsertArticle(context.Background(), "cms", name, command); !errors.Is(err, news.ErrInvalid) {
				t.Fatalf("expected ErrInvalid, got %v", err)
			}
		})
	}
}
