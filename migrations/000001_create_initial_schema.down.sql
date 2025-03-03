-- Drop indexes
DROP VIEW IF EXISTS document_activity;
DROP VIEW IF EXISTS user_activity;

DROP INDEX IF EXISTS idx_document_edits_edited_at;
DROP INDEX IF EXISTS idx_document_edits_user_id;
DROP INDEX IF EXISTS idx_document_edits_document_id;
DROP INDEX IF EXISTS idx_document_views_viewed_at;
DROP INDEX IF EXISTS idx_document_views_user_id;
DROP INDEX IF EXISTS idx_document_views_document_id;
DROP INDEX IF EXISTS idx_collaborators_user_id;
DROP INDEX IF EXISTS idx_collaborators_document_id;
DROP INDEX IF EXISTS idx_document_history_document_id;
DROP INDEX IF EXISTS idx_documents_deleted_at;
DROP INDEX IF EXISTS idx_documents_owner_id;
DROP INDEX IF EXISTS idx_users_email;

-- Drop tables
DROP TABLE IF EXISTS document_edits;
DROP TABLE IF EXISTS document_views;
DROP TABLE IF EXISTS collaborators;
DROP TABLE IF EXISTS document_histories;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS users;


