## How to run
### With docker compose
Copy `.env.example` to `.env` and fill in `ACCESS_SECRET` and `REFRESH_SECRET`.
Generate each secret separately with `openssl rand -hex 32`.

```bash
docker compose up
```
### Locally
- Set up PostgreSQL locally or run it from docker compose, and set `GROQ_API_KEY` for workout suggestions
- Export `ACCESS_SECRET` and `REFRESH_SECRET` in the server environment (a local `.env` is not loaded automatically by Go).
- Run main.go

The server refuses to start when either JWT secret is missing, blank, or uses one of the old published defaults. Keep secrets stable across restarts. Replacing existing secrets invalidates tokens signed with those secrets.

## API Documentation
- Swagger available in `./docs/`
- Or by `/swagger/index.html` endpoint


## Database schema

The server does not run migrations on startup. Provision the database schema separately before starting the API.

## Profile API

All profile routes require a Bearer access token. The legacy `/profile/stats`
and `/suggest-workout` endpoints remain available.

- `GET /profile/overview`: rolling last seven days in UTC, excluding future
  workouts. Returns `from`, `to` (exclusive), `workouts`, `sets`, `volume_kg`,
  and `muscle_balance`. A counted set has positive reps. Volume sums positive
  weight times reps; bodyweight sets count but add no external load. Muscle
  balance uses the primary muscle group and returns its percentage of counted
  sets, not a recommended target. Missing groups are reported as Unknown.
- `GET /profile/strength-progress?workout_type_id=1&from=2026-10-01&to=2026-11-13`:
  UTC calendar dates, inclusive input end date, capped at the current time.
  Defaults to the last 90 days; maximum interval is 366 days. Returns available
  `exercises` in the period, `workout_type_id`, `points`, and `change_percent`.
  Omit the exercise ID to select the lowest available ID. Each point is the
  best estimated 1RM in one training: actual weight for 1 rep, Epley
  `weight * (1 + reps / 30)` for 2–10 reps. Other sets are excluded.
  Change compares the first and last points; fewer than two points yields null.
  Empty history returns empty arrays, with no fabricated progress.
- `POST /profile/coach/messages`: send `{"question":"How can I improve?"}`
  or `{}` for an automatic overview. Each request is independent; there is no
  conversation ID or stored dialogue. The model receives the user's seven-day
  summary. Returns `{"answer":"..."}`. Questions are limited to 2000 characters
  and request bodies to 16 KiB. Missing LLM configuration returns 503; provider
  failures return 502. Set `GROQ_API_KEY` to enable the coach.

Overview and strength progress are computed by the backend without LLM calls.
