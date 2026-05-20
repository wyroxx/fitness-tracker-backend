## How to run
### With docker compose
```bash
docker compose up
```
### Locally
- Set up Ollama and PostreSQL locally or run from docker compose
- Run main.go

## API Documentation
- Swagger available in `./docs/`
- Or by `/swagger/index.html` endpoint

## Flutter-facing workout flow
- `GET /muscle-groups` - muscle group grid.
- `GET /muscle-groups/{id}/workout-types` or `GET /workout-types?muscle_group_id={id}&query=bench` - exercise catalog.
- `GET /trainings?limit=20&cursor={next_cursor}` - "My Trainings" feed.
- `POST /trainings` - create a full training with nested exercises and sets in one request.
- `POST /workout-sessions` - add an exercise session to an existing training.
- `PUT /workout-sessions/{id}/sets` - replace all sets from the set editor bottom sheet.
- `GET /profile/stats` - profile summary and lightweight insight.

Example `POST /trainings` payload:
```json
{
  "title": "Chest & Triceps",
  "started_at": "2026-05-20T18:00:00Z",
  "sessions": [
    {
      "workout_type_id": 1,
      "sets": [
        { "set_number": 1, "weight_kg": 50, "reps": 12 },
        { "set_number": 2, "weight_kg": 55, "reps": 10 }
      ]
    }
  ]
}
```
