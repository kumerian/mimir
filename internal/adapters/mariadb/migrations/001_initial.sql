CREATE TABLE IF NOT EXISTS metadata_revisions (
    revision_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    revision_status VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    metadata_digest CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    document LONGTEXT NOT NULL,
    actor VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (revision_id, revision_status),
    KEY ix_metadata_revisions_digest (metadata_digest)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS metadata_active (
    singleton_id TINYINT NOT NULL PRIMARY KEY,
    revision_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    metadata_digest CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    activated_at TIMESTAMP(6) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

INSERT IGNORE INTO metadata_active
    (singleton_id, revision_id, metadata_digest, activated_at)
VALUES (1, '', '', NULL);

CREATE TABLE IF NOT EXISTS metadata_activation_history (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    revision_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    metadata_digest CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    previous_revision_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    activated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    KEY ix_metadata_activation_revision (revision_id),
    KEY ix_metadata_activation_time (activated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS metadata_validation_reports (
    input_digest CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL PRIMARY KEY,
    report_json LONGTEXT NOT NULL,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS metadata_discovery_proposals (
    proposal_id CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL PRIMARY KEY,
    proposal_json LONGTEXT NOT NULL,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
