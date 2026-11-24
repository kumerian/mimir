package io.mimir.logging;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.Locale;

import org.slf4j.LoggerFactory;
import org.slf4j.event.Level;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;

@Service
public class LoggingService {
    private final JdbcTemplate jdbc;
    private final Severity threshold;

    public LoggingService(JdbcTemplate jdbc, @Value("${mimir.logging.level:INFO}") String configuredLevel) {
        this.jdbc = jdbc;
        try {
            this.threshold = Severity.valueOf(configuredLevel.trim().toUpperCase(Locale.ROOT));
        } catch (IllegalArgumentException exception) {
            throw new IllegalArgumentException("LOG_LEVEL must be DEBUG, INFO, WARN, or ERROR", exception);
        }
    }

    public void debug(Class<?> source, String message) {
        write(Severity.DEBUG, source, message);
    }

    public void info(Class<?> source, String message) {
        write(Severity.INFO, source, message);
    }

    public void warn(Class<?> source, String message) {
        write(Severity.WARN, source, message);
    }

    public void error(Class<?> source, String message) {
        write(Severity.ERROR, source, message);
    }

    private void write(Severity severity, Class<?> source, String message) {
        if (severity.ordinal() < threshold.ordinal()) {
            return;
        }

        LoggerFactory.getLogger(source).atLevel(severity.slf4jLevel).log(message);
        jdbc.update(
                "INSERT INTO application_log (occurred_at, log_level, source_name, message) VALUES (?, ?, ?, ?)",
                Timestamp.from(Instant.now()), severity.name(), source.getSimpleName(), message);
    }

    private enum Severity {
        DEBUG(Level.DEBUG),
        INFO(Level.INFO),
        WARN(Level.WARN),
        ERROR(Level.ERROR);

        private final Level slf4jLevel;

        Severity(Level slf4jLevel) {
            this.slf4jLevel = slf4jLevel;
        }
    }
}
