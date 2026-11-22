package io.mimir.metadata;

import java.util.Comparator;
import java.util.List;
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

    public MetadataStore(JdbcAggregateTemplate template) {
        this.template = template;
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

    public <T extends Metadata.Identified> T create(T value) {
        return template.insert(value);
    }

    public <T extends Metadata.Identified> T update(String id, T value, Class<T> type) {
        if (!id.equals(value.id())) {
            throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "Primary keys cannot be changed");
        }
        get(id, type);
        return template.update(value);
    }

    public <T extends Metadata.Identified> void delete(String id, Class<T> type) {
        get(id, type);
        template.deleteById(id, type);
    }
}
