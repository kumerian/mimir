package io.mimir.metadata;

import org.junit.jupiter.api.Test;
import org.springframework.dao.DataAccessResourceFailureException;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.data.relational.core.conversion.DbActionExecutionException;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class MetadataErrorsTest {
    @Test
    void wrappedConstraintViolationsAreConflictsWithoutDatabaseDetails() {
        var exception = mock(DbActionExecutionException.class);
        when(exception.getCause()).thenReturn(new DataIntegrityViolationException("private SQL details"));
        var response = new MetadataErrors().persistence(exception);
        assertEquals(409, response.getStatusCode().value());
        assertNotNull(response.getBody());
        assertFalse(response.getBody().getDetail().contains("private SQL"));
    }

    @Test
    void otherPersistenceFailuresAreExplicitServerErrors() {
        var exception = mock(DbActionExecutionException.class);
        when(exception.getCause()).thenReturn(new DataAccessResourceFailureException("connection lost"));
        var response = new MetadataErrors().persistence(exception);
        assertEquals(500, response.getStatusCode().value());
        assertNotNull(response.getBody());
        assertFalse(response.getBody().getDetail().contains("connection lost"));
    }
}
