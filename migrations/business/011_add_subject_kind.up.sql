ALTER TABLE subjects ADD COLUMN subject_kind TEXT NOT NULL DEFAULT 'dwelling';

CREATE INDEX idx_subjects_subject_kind ON subjects(subject_kind);
