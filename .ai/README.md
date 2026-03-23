# AI workspace guide

Read in this order:

1. `.ai/project.md`
2. Relevant architecture spec under `specs/architecture/`
3. `specs/database.md` for schema-backed work
4. `specs/scheduler_algorithm.md` for review scheduling and rating logic
5. Relevant guideline file under `.ai/guidelines/`

Use:

- `.ai/guidelines/backend.md` for Go backend tasks
- `.ai/guidelines/frontend.md` for Qwik frontend tasks

Do not:

- invent new patterns when an existing one already exists
- change the review/rating flow without updating both backend and frontend specs
- bypass tests for touched areas
