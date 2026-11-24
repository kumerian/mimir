package io.mimir.metadata;

import java.time.Instant;

public record MetadataAuditEntry(
        long id,
        Instant occurredAt,
        String operation,
        String entityType,
        String recordId,
        String details) {}
