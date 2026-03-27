BEGIN;

INSERT INTO users (
    id,
    email,
    password_hash,
    preferred_language,
    created_at,
    updated_at
)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'demo@example.com',
    crypt('demo12345', gen_salt('bf', 10)),
    'en',
    NOW(),
    NOW()
)
ON CONFLICT (email) DO UPDATE
SET
    password_hash = EXCLUDED.password_hash,
    preferred_language = EXCLUDED.preferred_language,
    updated_at = NOW();

INSERT INTO decks (
    id,
    user_id,
    title,
    description,
    language_code,
    created_at,
    updated_at
)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    '11111111-1111-1111-1111-111111111111',
    'Go Concurrency Basics',
    'Starter deck for local development.',
    'en',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE
SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    language_code = EXCLUDED.language_code,
    updated_at = NOW();

INSERT INTO info_objects (
    id,
    deck_id,
    title,
    content,
    discipline,
    content_type,
    created_at,
    updated_at
)
VALUES (
    '33333333-3333-3333-3333-333333333333',
    '22222222-2222-2222-2222-222222222222',
    'Launching a goroutine',
    $$func worker() {
    fmt.Println("working")
}

go worker()$$,
    'programming',
    'code_go',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE
SET
    title = EXCLUDED.title,
    content = EXCLUDED.content,
    discipline = EXCLUDED.discipline,
    content_type = EXCLUDED.content_type,
    updated_at = NOW();

INSERT INTO cards (
    id,
    info_object_id,
    front,
    step,
    correct_answers,
    distractors,
    highlight_lines,
    created_at,
    updated_at
)
VALUES
    (
        '44444444-4444-4444-4444-444444444444',
        '33333333-3333-3333-3333-333333333333',
        'Which expression starts the goroutine?',
        0,
        '[["go", "worker()"]]'::jsonb,
        '["defer", "func", "chan"]'::jsonb,
        '[5]'::jsonb,
        NOW(),
        NOW()
    ),
    (
        '55555555-5555-5555-5555-555555555555',
        '33333333-3333-3333-3333-333333333333',
        'Which function name is called by the goroutine?',
        1,
        '[["worker()"]]'::jsonb,
        '["main()", "run()", "job()"]'::jsonb,
        '[5]'::jsonb,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO UPDATE
SET
    front = EXCLUDED.front,
    step = EXCLUDED.step,
    correct_answers = EXCLUDED.correct_answers,
    distractors = EXCLUDED.distractors,
    highlight_lines = EXCLUDED.highlight_lines,
    updated_at = NOW();

INSERT INTO card_states (
    id,
    card_id,
    user_id,
    stability,
    difficulty,
    retrievability,
    due_date,
    last_review,
    interval_days,
    status,
    reps,
    lapses
)
VALUES
    (
        '66666666-6666-6666-6666-666666666666',
        '44444444-4444-4444-4444-444444444444',
        '11111111-1111-1111-1111-111111111111',
        0,
        5,
        0,
        NOW(),
        NULL,
        0,
        'new',
        0,
        0
    ),
    (
        '77777777-7777-7777-7777-777777777777',
        '55555555-5555-5555-5555-555555555555',
        '11111111-1111-1111-1111-111111111111',
        0,
        5,
        0,
        NULL,
        NULL,
        0,
        'locked',
        0,
        0
    )
ON CONFLICT (card_id, user_id) DO UPDATE
SET
    stability = EXCLUDED.stability,
    difficulty = EXCLUDED.difficulty,
    retrievability = EXCLUDED.retrievability,
    due_date = EXCLUDED.due_date,
    last_review = EXCLUDED.last_review,
    interval_days = EXCLUDED.interval_days,
    status = EXCLUDED.status,
    reps = EXCLUDED.reps,
    lapses = EXCLUDED.lapses;

COMMIT;
