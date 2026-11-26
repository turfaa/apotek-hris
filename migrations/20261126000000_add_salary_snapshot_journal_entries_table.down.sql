-- Refuse to forget entries that exist in Vmedis. The drop is in the same
-- block, so it never runs after the refusal, whatever runs this file.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM salary_snapshot_journal_entries WHERE status IN ('posted', 'pending')) THEN
        RAISE EXCEPTION 'salary_snapshot_journal_entries has posted entries; delete them in Vmedis and here first';
    END IF;

    DROP TABLE salary_snapshot_journal_entries;
END $$;
