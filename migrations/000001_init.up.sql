CREATE TABLE IF NOT EXISTS images (
    id VARCHAR(255) PRIMARY KEY,
    original_path VARCHAR(500) NOT NULL,
    processed_path VARCHAR(500),
    thumbnail_path VARCHAR(500),
    status VARCHAR(50) NOT NULL,
    format VARCHAR(10) NOT NULL,
    original_width INTEGER NOT NULL,
    original_height INTEGER NOT NULL,
    processed_width INTEGER,
    processed_height INTEGER,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_images_status ON images(status);
CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at);
