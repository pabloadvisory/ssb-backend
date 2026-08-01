CREATE TABLE news_articles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source varchar(64) NOT NULL,
    external_id varchar(256) NOT NULL,
    slug varchar(160) NOT NULL,
    title varchar(200) NOT NULL,
    summary varchar(500) NOT NULL,
    body_markdown text NOT NULL,
    hero_image_url varchar(2048),
    hero_image_alt varchar(300),
    author_name varchar(120),
    category varchar(32) NOT NULL DEFAULT 'story',
    featured boolean NOT NULL DEFAULT false,
    related_league_id uuid REFERENCES leagues(id) ON DELETE SET NULL,
    related_team_id uuid REFERENCES teams(id) ON DELETE SET NULL,
    related_match_id uuid REFERENCES matches(id) ON DELETE SET NULL,
    status varchar(16) NOT NULL DEFAULT 'draft',
    published_at timestamptz,
    source_hash bytea NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    UNIQUE (slug),
    CONSTRAINT news_articles_source_not_blank CHECK (btrim(source) <> ''),
    CONSTRAINT news_articles_external_id_not_blank CHECK (btrim(external_id) <> ''),
    CONSTRAINT news_articles_slug_valid CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT news_articles_title_not_blank CHECK (btrim(title) <> ''),
    CONSTRAINT news_articles_summary_not_blank CHECK (btrim(summary) <> ''),
    CONSTRAINT news_articles_body_not_blank CHECK (btrim(body_markdown) <> ''),
    CONSTRAINT news_articles_category_valid CHECK (category IN ('story', 'match_report', 'announcement')),
    CONSTRAINT news_articles_status_valid CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT news_articles_published_at_required CHECK (status <> 'published' OR published_at IS NOT NULL),
    CONSTRAINT news_articles_version_positive CHECK (version > 0)
);

CREATE INDEX news_articles_feed_idx
    ON news_articles (status, published_at DESC, id DESC);
CREATE INDEX news_articles_category_feed_idx
    ON news_articles (category, status, published_at DESC, id DESC);
CREATE INDEX news_articles_league_feed_idx
    ON news_articles (related_league_id, status, published_at DESC, id DESC)
    WHERE related_league_id IS NOT NULL;
CREATE INDEX news_articles_team_feed_idx
    ON news_articles (related_team_id, status, published_at DESC, id DESC)
    WHERE related_team_id IS NOT NULL;
CREATE INDEX news_articles_match_feed_idx
    ON news_articles (related_match_id, status, published_at DESC, id DESC)
    WHERE related_match_id IS NOT NULL;

CREATE TRIGGER news_articles_touch_updated_at
    BEFORE UPDATE ON news_articles FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
