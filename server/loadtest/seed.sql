CREATE TABLE IF NOT EXISTS scores (
    id SERIAL PRIMARY KEY,
    player_name VARCHAR(50) NOT NULL,
    stage_id INT NOT NULL,
    clear_time DOUBLE PRECISION NOT NULL,
    death_count INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO scores (
    player_name,
    stage_id,
    clear_time,
    death_count,
    created_at
)
SELECT
    'LOAD_PLAYER_' || LPAD(player_no::text, 4, '0'),
    1,
    20.0
        + ((player_no % 300) * 0.5)
        + (attempt_no * 0.123),
    (player_no + attempt_no) % 20,
    CURRENT_TIMESTAMP
        - ((player_no * 10 + attempt_no) * INTERVAL '1 second')
FROM generate_series(1, 1000) AS player_no
CROSS JOIN generate_series(1, 10) AS attempt_no;