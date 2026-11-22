package io.mimir.metadata;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.data.relational.core.conversion.DbActionExecutionException;
import org.springframework.http.HttpStatus;
import org.springframework.http.ProblemDetail;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.servlet.mvc.method.annotation.ResponseEntityExceptionHandler;

@RestControllerAdvice(assignableTypes = MetadataController.class)
public class MetadataErrors extends ResponseEntityExceptionHandler {
    private static final Logger LOG = LoggerFactory.getLogger(MetadataErrors.class);

    @ExceptionHandler(DataIntegrityViolationException.class)
    ProblemDetail conflict(DataIntegrityViolationException exception) {
        LOG.warn("Metadata constraint violation: {}", exception.getClass().getSimpleName());
        return ProblemDetail.forStatusAndDetail(HttpStatus.CONFLICT,
                "A key already exists, a referenced record is missing, or this record is still referenced.");
    }

    @ExceptionHandler(DbActionExecutionException.class)
    ResponseEntity<ProblemDetail> persistence(DbActionExecutionException exception) {
        for (Throwable cause = exception.getCause(); cause != null; cause = cause.getCause()) {
            if (cause instanceof DataIntegrityViolationException integrity) {
                return ResponseEntity.status(HttpStatus.CONFLICT).body(conflict(integrity));
            }
        }
        LOG.error("Metadata persistence failed", exception);
        return ResponseEntity.internalServerError().body(
                ProblemDetail.forStatusAndDetail(HttpStatus.INTERNAL_SERVER_ERROR, "Metadata could not be saved."));
    }
}
