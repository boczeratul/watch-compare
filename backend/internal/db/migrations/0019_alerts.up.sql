-- Push alerts: a saved search per device. subscriber_id is the random id the app or browser
-- generated and registered with OneSignal as its external_id; notifications target it.
CREATE TABLE IF NOT EXISTS alerts (
    id                BIGSERIAL PRIMARY KEY,
    subscriber_id     TEXT NOT NULL,
    name              TEXT NOT NULL,
    query             TEXT NOT NULL,                    -- URL-encoded /api/v1/listings criteria incl. currency
    checked_at        TIMESTAMPTZ NOT NULL DEFAULT now(), -- listings first seen after this are new finds
    last_notified_at  TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (subscriber_id, query)
);

-- New-find lookups filter on first_seen_at.
CREATE INDEX IF NOT EXISTS listings_first_seen_idx ON listings (first_seen_at DESC) WHERE is_active;
