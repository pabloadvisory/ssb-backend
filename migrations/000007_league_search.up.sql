CREATE INDEX leagues_search_trgm_idx
    ON leagues USING gin (name gin_trgm_ops);
