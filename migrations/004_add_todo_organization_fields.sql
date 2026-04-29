ALTER TABLE todos
ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'normal',
ADD COLUMN IF NOT EXISTS notes TEXT NOT NULL DEFAULT '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'todos_priority_check'
    ) THEN
        ALTER TABLE todos
        ADD CONSTRAINT todos_priority_check
        CHECK (priority IN ('low', 'normal', 'high'));
    END IF;
END $$;
