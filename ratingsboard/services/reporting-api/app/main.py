"""
RatingsBoard Reporting API.

Serves computed media ratings to the dashboard and API consumers, and exposes a
Prometheus /metrics endpoint so the service is observable (request rate, errors,
latency — the RED method). Reads from the reports table that the rating-calc
CronJob populates from PanelPulse's measurement_aggregates.

Endpoints:
  GET /api/ratings                 - list latest ratings (optional ?channel= &daypart=)
  GET /api/ratings/top             - top channels by rating
  GET /healthz                     - liveness
  GET /readyz                      - readiness (verifies DB)
  GET /metrics                     - Prometheus metrics

Original, generic reference design using public measurement concepts
(ratings, share, impressions, reach). Not based on any proprietary system.
"""
import os
import time
import logging
from contextlib import contextmanager

import psycopg2
import psycopg2.extras
from fastapi import FastAPI, Response, Query
from fastapi.responses import JSONResponse
from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST

logging.basicConfig(level=logging.INFO)
log = logging.getLogger("reporting-api")

app = FastAPI(title="RatingsBoard Reporting API", version="1.0")

# --- Prometheus metrics (the RED method: Rate, Errors, Duration) ---
REQUESTS = Counter("http_requests_total", "Total HTTP requests", ["method", "path", "status"])
LATENCY = Histogram("http_request_duration_seconds", "Request latency", ["path"])

DB = dict(
    host=os.getenv("DB_HOST", "reports-db"),
    port=os.getenv("DB_PORT", "5432"),
    dbname=os.getenv("DB_NAME", "ratingsboard"),
    user=os.getenv("DB_USER", "ratingsboard"),
    password=os.getenv("DB_PASSWORD", ""),
)


@contextmanager
def db_cursor():
    conn = psycopg2.connect(**DB)
    try:
        yield conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)
    finally:
        conn.close()


@app.middleware("http")
async def observe(request, call_next):
    start = time.time()
    response = await call_next(request)
    elapsed = time.time() - start
    path = request.url.path
    LATENCY.labels(path=path).observe(elapsed)
    REQUESTS.labels(method=request.method, path=path, status=response.status_code).inc()
    return response


@app.get("/healthz")
def healthz():
    return {"status": "ok"}


@app.get("/readyz")
def readyz():
    try:
        with db_cursor() as cur:
            cur.execute("SELECT 1")
            cur.fetchone()
        return {"status": "ready"}
    except Exception as e:
        log.warning("readiness failed: %s", e)
        return JSONResponse({"status": "db unreachable"}, status_code=503)


@app.get("/metrics")
def metrics():
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)


@app.get("/api/ratings")
def ratings(channel: str = Query(None), daypart: str = Query(None)):
    """Latest ratings, optionally filtered by channel/daypart."""
    q = "SELECT channel_id, daypart, rating, share, impressions, reach, calculated_at FROM reports WHERE 1=1"
    params = []
    if channel:
        q += " AND channel_id = %s"; params.append(channel)
    if daypart:
        q += " AND daypart = %s"; params.append(daypart)
    q += " ORDER BY calculated_at DESC, rating DESC LIMIT 200"
    try:
        with db_cursor() as cur:
            cur.execute(q, params)
            return {"ratings": cur.fetchall()}
    except Exception as e:
        log.error("query failed: %s", e)
        return JSONResponse({"error": "query_failed"}, status_code=500)


@app.get("/api/ratings/top")
def top(limit: int = Query(10, ge=1, le=100)):
    """Top channels by most recent rating."""
    try:
        with db_cursor() as cur:
            cur.execute(
                "SELECT channel_id, daypart, rating, share FROM reports "
                "ORDER BY calculated_at DESC, rating DESC LIMIT %s", [limit])
            return {"top": cur.fetchall()}
    except Exception as e:
        log.error("query failed: %s", e)
        return JSONResponse({"error": "query_failed"}, status_code=500)
