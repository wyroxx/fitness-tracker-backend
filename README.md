## How to run
### With docker compose
```bash
docker compose up
```
### Locally
- Set up PostgreSQL locally or run it from docker compose, and set `GROQ_API_KEY` for workout suggestions
- Run main.go

## API Documentation
- Swagger available in `./docs/`
- Or by `/swagger/index.html` endpoint


## Database schema

The server does not run migrations on startup. Provision the database schema separately before starting the API.
