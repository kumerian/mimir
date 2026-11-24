package io.mimir.metadata;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Objects;
import java.util.Set;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;

@Service
public class MetadataAuditService {
    private static final Set<String> SENSITIVE_FIELDS = Set.of(
            "authenticationUser", "authenticationCredential", "parameterValue");

    private final JdbcTemplate jdbc;
    private final ObjectMapper json;

    public MetadataAuditService(JdbcTemplate jdbc, ObjectMapper json) {
        this.jdbc = jdbc;
        this.json = json;
    }

    public void created(Metadata.Identified value) {
        Map<String, Object> after = fields(value);
        Map<String, Map<String, Object>> changes = new LinkedHashMap<>();
        after.forEach((field, current) -> changes.put(field, change(null, current)));
        insert("CREATE", value, changes);
    }

    public void updated(Metadata.Identified before, Metadata.Identified after) {
        Map<String, Object> previous = fields(before);
        Map<String, Object> current = fields(after);
        Map<String, Map<String, Object>> changes = new LinkedHashMap<>();
        current.forEach((field, value) -> {
            Object oldValue = previous.get(field);
            if (!Objects.equals(oldValue, value)) {
                changes.put(field, change(oldValue, value));
            }
        });
        insert("UPDATE", after, changes);
    }

    public void deleted(Metadata.Identified value) {
        Map<String, Object> before = fields(value);
        Map<String, Map<String, Object>> changes = new LinkedHashMap<>();
        before.forEach((field, previous) -> changes.put(field, change(previous, null)));
        insert("DELETE", value, changes);
    }

    private Map<String, Object> fields(Metadata.Identified value) {
        return json.convertValue(value, new TypeReference<>() {});
    }

    private Map<String, Object> change(Object before, Object after) {
        Map<String, Object> result = new LinkedHashMap<>();
        result.put("before", before);
        result.put("after", after);
        return result;
    }

    private Object redact(String field, Object value) {
        return value != null && SENSITIVE_FIELDS.contains(field) ? "[REDACTED]" : value;
    }

    private void insert(
            String operation, Metadata.Identified value, Map<String, Map<String, Object>> changes) {
        Map<String, Object> safeChanges = new LinkedHashMap<>();
        changes.forEach((field, values) -> {
            safeChanges.put(field, change(
                    redact(field, values.get("before")),
                    redact(field, values.get("after"))));
        });
        String details;
        try {
            details = json.writeValueAsString(safeChanges);
        } catch (JsonProcessingException exception) {
            throw new IllegalStateException("Could not serialize metadata audit details", exception);
        }
        jdbc.update(
                "INSERT INTO metadata_audit (occurred_at, operation, entity_type, record_id, details) " +
                        "VALUES (?, ?, ?, ?, ?)",
                Timestamp.from(Instant.now()), operation, value.getClass().getSimpleName(), value.id(), details);
    }
}
