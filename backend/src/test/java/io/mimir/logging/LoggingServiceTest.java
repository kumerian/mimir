package io.mimir.logging;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.springframework.boot.test.system.CapturedOutput;
import org.springframework.boot.test.system.OutputCaptureExtension;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.dao.DataAccessResourceFailureException;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(OutputCaptureExtension.class)
class LoggingServiceTest {
    @Test
    void appliesOneThresholdToTerminalAndDatabase(CapturedOutput output) {
        var jdbc = mock(JdbcTemplate.class);
        var logging = new LoggingService(jdbc, "WARN");

        logging.debug(LoggingServiceTest.class, "debug event");
        logging.info(LoggingServiceTest.class, "info event");
        logging.warn(LoggingServiceTest.class, "warning event");
        logging.error(LoggingServiceTest.class, "error event");

        verify(jdbc, times(2)).update(
                startsWith("INSERT INTO application_log"), any(), any(), any(), any());
        assertTrue(output.getOut().contains("WARN"));
        assertTrue(output.getOut().contains("LoggingServiceTest"));
        assertTrue(output.getOut().contains("warning event"));
        assertTrue(output.getOut().contains("ERROR"));
        assertFalse(output.getOut().contains("debug event"));
        assertFalse(output.getOut().contains("info event"));
    }

    @Test
    void rejectsUnsupportedThresholds() {
        assertThrows(IllegalArgumentException.class, () -> new LoggingService(mock(JdbcTemplate.class), "TRACE"));
    }

    @Test
    void surfacesDatabaseWriteFailures(CapturedOutput output) {
        var jdbc = mock(JdbcTemplate.class);
        when(jdbc.update(anyString(), any(), any(), any(), any()))
                .thenThrow(new DataAccessResourceFailureException("database unavailable"));
        var logging = new LoggingService(jdbc, "INFO");

        assertThrows( DataAccessResourceFailureException.class,
                () -> logging.info(LoggingServiceTest.class, "visible terminal event"));
        assertTrue(output.getOut().contains("visible terminal event"));
    }
}
