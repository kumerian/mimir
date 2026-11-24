package io.mimir.metadata;

import io.mimir.logging.LoggingService;
import java.util.Comparator;
import java.util.List;
import java.util.UUID;
import java.util.stream.StreamSupport;

import org.springframework.data.jdbc.core.JdbcAggregateTemplate;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.server.ResponseStatusException;

@Service
@Transactional
public class MetadataStore {
    private final JdbcAggregateTemplate template;
    private final MetadataAuditService audit;
    private final LoggingService logging;

    public MetadataStore(JdbcAggregateTemplate template, MetadataAuditService audit, LoggingService logging) {
        this.template = template;
        this.audit = audit;
        this.logging = logging;
    }

    @Transactional(readOnly = true)
    public <T extends Metadata.Identified> List<T> list(Class<T> type) {
        return StreamSupport.stream(template.findAll(type).spliterator(), false)
                .sorted(Comparator.comparing(Metadata.Identified::id)).toList();
    }

    @Transactional(readOnly = true)
    public <T extends Metadata.Identified> T get(String id, Class<T> type) {
        T value = template.findById(id, type);
        if (value == null) {
            throw new ResponseStatusException(HttpStatus.NOT_FOUND, "Record not found");
        }
        return value;
    }

    public <T extends Metadata.Identified> T create(T value, Class<T> type) {
        if (value instanceof Metadata.GeneratedId<?> generated) {
            value = type.cast(generated.withId(UUID.randomUUID().toString()));
        }
        T saved = template.insert(value);
        audit.created(saved);
        logging.info(MetadataStore.class, "Metadata " + value.getClass().getSimpleName() + " created");
        return saved;
    }

    public <T extends Metadata.Identified> T update(String id, T value, Class<T> type) {
        if (!id.equals(value.id())) {
            throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "Primary keys cannot be changed");
        }
        T previous = get(id, type);
        T saved = template.update(value);
        audit.updated(previous, saved);
        logging.info(MetadataStore.class, "Metadata " + value.getClass().getSimpleName() + " updated");
        return saved;
    }

    public <T extends Metadata.Identified> void delete(String id, Class<T> type) {
        T previous = get(id, type);
        template.deleteById(id, type);
        audit.deleted(previous);
        logging.info(MetadataStore.class, "Metadata " + type.getSimpleName() + " deleted");
    }
}
