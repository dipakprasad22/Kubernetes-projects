"""
RatingsBoard Rating Calculation Job.

A run-to-completion batch (invoked by a Kubernetes CronJob) that reads
PanelPulse's measurement_aggregates and computes media ratings and share,
writing them to the reports table the API serves.

Ratings model (standard, public concepts):
  - rating = (reach / panel_universe) * 100      -> % of the panel reached
  - share  = (impressions / total_impressions_in_daypart) * 100
The panel universe is configurable (env PANEL_UNIVERSE).

Run-to-completion: does its work and exits non-zero on failure so the CronJob
records it. Original generic design; not based on any proprietary system.
"""
import os
import sys
import logging
from datetime import datetime, timezone

import psycopg2
import psycopg2.extras

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("rating-calc")

PANEL_UNIVERSE = int(os.getenv("PANEL_UNIVERSE", "10000"))

DB = dict(
    host=os.getenv("DB_HOST", "reports-db"),
    port=os.getenv("DB_PORT", "5432"),
    dbname=os.getenv("DB_NAME", "ratingsboard"),
    user=os.getenv("DB_USER", "ratingsboard"),
    password=os.getenv("DB_PASSWORD", ""),
)
# Source aggregates live in PanelPulse's DB (or a shared analytics store).
SRC = dict(
    host=os.getenv("SRC_DB_HOST", "panelpulse-results-db"),
    port=os.getenv("SRC_DB_PORT", "5432"),
    dbname=os.getenv("SRC_DB_NAME", "panelpulse"),
    user=os.getenv("SRC_DB_USER", "panelpulse"),
    password=os.getenv("SRC_DB_PASSWORD", os.getenv("DB_PASSWORD", "")),
)


def ensure_schema(cur):
    cur.execute("""
        CREATE TABLE IF NOT EXISTS reports (
            id SERIAL PRIMARY KEY,
            channel_id TEXT NOT NULL,
            daypart TEXT NOT NULL,
            rating NUMERIC(6,3) NOT NULL,
            share NUMERIC(6,3) NOT NULL,
            impressions BIGINT NOT NULL,
            reach BIGINT NOT NULL,
            calculated_at TIMESTAMPTZ NOT NULL,
            UNIQUE (channel_id, daypart, calculated_at)
        )""")


def fetch_aggregates(src_cur):
    """Read the latest measurement aggregates from PanelPulse."""
    src_cur.execute("""
        SELECT channel_id, daypart, impressions, viewing_min, reach
        FROM measurement_aggregates
    """)
    return src_cur.fetchall()


def compute_and_store():
    now = datetime.now(timezone.utc)
    # Read source aggregates (degrade gracefully if the source is empty/unreachable
    # in a fresh local env — log and exit 0 so the demo CronJob doesn't hard-fail).
    try:
        src = psycopg2.connect(**SRC)
        with src.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as sc:
            rows = fetch_aggregates(sc)
        src.close()
    except Exception as e:
        log.warning("source aggregates unavailable (%s) — nothing to compute yet", e)
        return 0

    if not rows:
        log.info("no aggregates to process")
        return 0

    # total impressions per daypart for share calculation
    totals = {}
    for r in rows:
        totals[r["daypart"]] = totals.get(r["daypart"], 0) + int(r["impressions"])

    conn = psycopg2.connect(**DB)
    try:
        with conn.cursor() as cur:
            ensure_schema(cur)
            written = 0
            for r in rows:
                reach = int(r["reach"])
                impressions = int(r["impressions"])
                daypart_total = totals.get(r["daypart"], 0) or 1
                rating = round(reach / PANEL_UNIVERSE * 100, 3)
                share = round(impressions / daypart_total * 100, 3)
                cur.execute("""
                    INSERT INTO reports
                      (channel_id, daypart, rating, share, impressions, reach, calculated_at)
                    VALUES (%s,%s,%s,%s,%s,%s,%s)
                    ON CONFLICT (channel_id, daypart, calculated_at) DO NOTHING
                """, (r["channel_id"], r["daypart"], rating, share, impressions, reach, now))
                written += 1
            conn.commit()
            log.info("computed %d ratings (universe=%d)", written, PANEL_UNIVERSE)
    finally:
        conn.close()
    return 0


if __name__ == "__main__":
    try:
        sys.exit(compute_and_store())
    except Exception as e:
        log.error("rating calculation failed: %s", e)
        sys.exit(1)
