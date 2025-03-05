-- Document API Database Schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create index on users email
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- Create documents table
CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    content TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    owner_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for documents
CREATE INDEX IF NOT EXISTS idx_documents_owner_id ON documents(owner_id);
CREATE INDEX IF NOT EXISTS idx_documents_deleted_at ON documents(deleted_at);
CREATE INDEX IF NOT EXISTS idx_documents_title ON documents(title);
CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents(created_at);
CREATE INDEX IF NOT EXISTS idx_documents_updated_at ON documents(updated_at);
CREATE INDEX IF NOT EXISTS idx_documents_is_public ON documents(is_public);

-- Create document_history table
CREATE TABLE IF NOT EXISTS document_histories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    content TEXT,
    updated_by_id UUID NOT NULL REFERENCES users(id),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (document_id, version)
);

-- Create indexes for document_history
CREATE INDEX IF NOT EXISTS idx_document_history_document_id ON document_histories(document_id);
CREATE INDEX IF NOT EXISTS idx_document_history_updated_by_id ON document_histories(updated_by_id);
CREATE INDEX IF NOT EXISTS idx_document_history_updated_at ON document_histories(updated_at);

-- Create collaborators table
CREATE TABLE IF NOT EXISTS collaborators (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    permission VARCHAR(20) NOT NULL CHECK (permission IN ('read', 'write')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (document_id, user_id)
);

-- Create indexes for collaborators
CREATE INDEX IF NOT EXISTS idx_collaborators_document_id ON collaborators(document_id);
CREATE INDEX IF NOT EXISTS idx_collaborators_user_id ON collaborators(user_id);
CREATE INDEX IF NOT EXISTS idx_collaborators_permission ON collaborators(permission);

-- Create document_views table for analytics
CREATE TABLE IF NOT EXISTS document_views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),
    ip_address VARCHAR(45),
    user_agent VARCHAR(255),
    viewed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for document_views
CREATE INDEX IF NOT EXISTS idx_document_views_document_id ON document_views(document_id);
CREATE INDEX IF NOT EXISTS idx_document_views_user_id ON document_views(user_id);
CREATE INDEX IF NOT EXISTS idx_document_views_viewed_at ON document_views(viewed_at);
CREATE INDEX IF NOT EXISTS idx_document_views_ip_address ON document_views(ip_address);

-- Create document_edits table for analytics
CREATE TABLE IF NOT EXISTS document_edits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    version INTEGER NOT NULL,
    edited_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for document_edits
CREATE INDEX IF NOT EXISTS idx_document_edits_document_id ON document_edits(document_id);
CREATE INDEX IF NOT EXISTS idx_document_edits_user_id ON document_edits(user_id);
CREATE INDEX IF NOT EXISTS idx_document_edits_edited_at ON document_edits(edited_at);

-- Create full-text search index for document content (PostgreSQL specific)
-- This enables efficient searching within documents
ALTER TABLE documents ADD COLUMN IF NOT EXISTS content_tsv TSVECTOR;
CREATE INDEX IF NOT EXISTS idx_documents_content_tsv ON documents USING GIN(content_tsv);

-- Create trigger function to update content_tsv on document insert/update
CREATE OR REPLACE FUNCTION documents_search_trigger() RETURNS trigger AS $$
BEGIN
    NEW.content_tsv :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.content, '')), 'B');
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- Apply trigger to documents table
DROP TRIGGER IF EXISTS tsvector_update_trigger ON documents;
CREATE TRIGGER tsvector_update_trigger
    BEFORE INSERT OR UPDATE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION documents_search_trigger();

-- Create views for common analytics queries

-- View for document activity (last 30 days)
CREATE OR REPLACE VIEW document_activity AS
SELECT
    d.id AS document_id,
    d.title,
    d.owner_id,
    COUNT(DISTINCT dv.id) AS view_count,
    COUNT(DISTINCT de.id) AS edit_count,
    COUNT(DISTINCT CASE WHEN dv.user_id IS NOT NULL THEN dv.user_id END) AS unique_viewers,
    MAX(dv.viewed_at) AS last_viewed,
    MAX(de.edited_at) AS last_edited
FROM
    documents d
LEFT JOIN document_views dv ON d.id = dv.document_id AND dv.viewed_at >= NOW() - INTERVAL '30 days'
LEFT JOIN document_edits de ON d.id = de.document_id AND de.edited_at >= NOW() - INTERVAL '30 days'
WHERE
    d.deleted_at IS NULL
GROUP BY
    d.id, d.title, d.owner_id;

-- View for user activity (last 30 days)
CREATE OR REPLACE VIEW user_activity AS
SELECT
    u.id AS user_id,
    u.name,
    u.email,
    COUNT(DISTINCT d.id) AS owned_documents,
    COUNT(DISTINCT c.document_id) AS collaborated_documents,
    COUNT(DISTINCT dv.id) AS document_views,
    COUNT(DISTINCT de.id) AS document_edits,
    MAX(dv.viewed_at) AS last_activity_view,
    MAX(de.edited_at) AS last_activity_edit
FROM
    users u
LEFT JOIN documents d ON u.id = d.owner_id AND d.deleted_at IS NULL
LEFT JOIN collaborators c ON u.id = c.user_id
LEFT JOIN document_views dv ON u.id = dv.user_id AND dv.viewed_at >= NOW() - INTERVAL '30 days'
LEFT JOIN document_edits de ON u.id = de.user_id AND de.edited_at >= NOW() - INTERVAL '30 days'
WHERE
    u.deleted_at IS NULL
GROUP BY
    u.id, u.name, u.email;

CREATE OR REPLACE FUNCTION record_document_view(
    doc_id UUID,
    usr_id UUID,
    ip VARCHAR(45),
    agent VARCHAR(255)
) RETURNS VOID AS $$
BEGIN
    INSERT INTO document_views (document_id, user_id, ip_address, user_agent, viewed_at)
    VALUES (doc_id, usr_id, ip, agent, NOW());
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION record_document_edit(
    doc_id UUID,
    usr_id UUID,
    ver INTEGER
) RETURNS VOID AS $$
BEGIN
    INSERT INTO document_edits (document_id, user_id, version, edited_at)
    VALUES (doc_id, usr_id, ver, NOW());
END;
$$ LANGUAGE plpgsql;

-- Function to check if a user can access a document
CREATE OR REPLACE FUNCTION can_user_access_document(
    doc_id UUID,
    usr_id UUID,
    required_permission VARCHAR
) RETURNS BOOLEAN AS $$
DECLARE
    is_owner BOOLEAN;
    is_public BOOLEAN;
    collab_permission VARCHAR;
BEGIN
    -- Check if user is the owner
    SELECT EXISTS(
        SELECT 1 FROM documents
        WHERE id = doc_id AND owner_id = usr_id AND deleted_at IS NULL
    ) INTO is_owner;
    
    IF is_owner THEN
        RETURN TRUE;
    END IF;
    
    -- If read access is required, check if document is public
    IF required_permission = 'read' THEN
        SELECT is_public FROM documents
        WHERE id = doc_id AND deleted_at IS NULL
        INTO is_public;
        
        IF is_public THEN
            RETURN TRUE;
        END IF;
    END IF;
    
    -- Check collaboration permission
    SELECT permission FROM collaborators
    WHERE document_id = doc_id AND user_id = usr_id
    INTO collab_permission;
    
    IF collab_permission IS NULL THEN
        RETURN FALSE;
    END IF;
    
    -- For read access, both read and write permissions are sufficient
    IF required_permission = 'read' THEN
        RETURN TRUE;
    END IF;
    
    -- For write access, only write permission is sufficient
    RETURN collab_permission = 'write';
END;
$$ LANGUAGE plpgsql;