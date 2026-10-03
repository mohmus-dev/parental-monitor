-- Demo seed data for dashboard verification.
-- Run with: psql "$POSTGRES_DSN" -f scripts/seed_demo_data.sql

CREATE TABLE IF NOT EXISTS monitor_searches (
    id TEXT PRIMARY KEY,
    child_id TEXT,
    device_id TEXT,
    query TEXT NOT NULL,
    engine TEXT,
    incognito BOOLEAN DEFAULT false,
    window_title TEXT,
    timestamp TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS monitor_alerts (
    id TEXT PRIMARY KEY,
    child_id TEXT,
    device_id TEXT,
    category TEXT,
    keyword TEXT NOT NULL,
    query TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL
);

INSERT INTO monitor_searches (id, child_id, device_id, query, engine, incognito, window_title, timestamp)
VALUES
    ('seed-search-001', 'demo-child-001', 'demo-device-001', 'how to make money online fast', 'google', false, 'Google Search', NOW() - INTERVAL '2 hours'),
    ('seed-search-002', 'demo-child-001', 'demo-device-001', 'best fast food recipes', 'bing', false, 'Bing Search', NOW() - INTERVAL '55 minutes'),
    ('seed-search-003', 'demo-child-002', 'demo-device-002', 'watch free movies', 'duckduckgo', false, 'DuckDuckGo', NOW() - INTERVAL '21 minutes')
ON CONFLICT (id) DO NOTHING;

INSERT INTO monitor_alerts (id, child_id, device_id, category, keyword, query, timestamp)
VALUES
    ('seed-alert-001', 'demo-child-001', 'demo-device-001', 'adult_content', 'porn', 'porn videos free', NOW() - INTERVAL '3 hours'),
    ('seed-alert-002', 'demo-child-001', 'demo-device-001', 'gambling', 'casino', 'online casino bonus codes', NOW() - INTERVAL '42 minutes'),
    ('seed-alert-003', 'demo-child-002', 'demo-device-002', 'drugs', 'cocaine', 'buy cocaine online', NOW() - INTERVAL '19 minutes')
ON CONFLICT (id) DO NOTHING;

SELECT 'demo data inserted into monitor_searches and monitor_alerts' AS status;
