package io.mimir.metadata;

import io.mimir.logging.LoggingService;
import org.junit.jupiter.api.Test;
import org.springframework.dao.DataAccessResourceFailureException;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.data.relational.core.conversion.DbActionExecutionException;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class MetadataErrorsTest {
    @Test
    void wrappedConstraintViolationsAreConflictsWithoutDatabaseDetails() {
        var exception = mock(DbActionExecutionException.class);
        var logging = mock(LoggingService.class);
        when(exception.getCause()).thenReturn(new DataIntegrityViolationException("private SQL details"));
        var response = new MetadataErrors(logging).persistence(exception);
        assertEquals(409, response.getStatusCode().value());
        assertNotNull(response.getBody());
        assertFalse(response.getBody().getDetail().contains("private SQL"));
    }

    @Test
    void otherPersistenceFailuresAreExplicitServerErrors() {
        var exception = mock(DbActionExecutionException.class);
        var logging = mock(LoggingService.class);
        when(exception.getCause()).thenReturn(new DataAccessResourceFailureException("connection lost"));
        var response = new MetadataErrors(logging).persistence(exception);
        assertEquals(500, response.getStatusCode().value());
        assertNotNull(response.getBody());
        assertFalse(response.getBody().getDetail().contains("connection lost"));
        verify(logging).error(MetadataErrors.class, "Metadata persistence failed");
    }
}
