package io.mimir.metadata;

import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.List;

import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/audit")
public class MetadataAuditController {
    private final JdbcTemplate jdbc;

    public MetadataAuditController(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    @GetMapping
    public List<MetadataAuditEntry> list() {
        return jdbc.query(
                "SELECT audit_id, occurred_at, operation, entity_type, record_id, details " +
                        "FROM metadata_audit ORDER BY occurred_at DESC, audit_id DESC",
                this::mapEntry);
    }

    private MetadataAuditEntry mapEntry(ResultSet row, int rowNumber) throws SQLException {
        return new MetadataAuditEntry(
                row.getLong("audit_id"),
                row.getTimestamp("occurred_at").toInstant(),
                row.getString("operation"),
                row.getString("entity_type"),
                row.getString("record_id"),
                row.getString("details"));
    }
}
