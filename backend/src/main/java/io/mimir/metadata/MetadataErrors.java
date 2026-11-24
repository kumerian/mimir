package io.mimir.metadata;

import io.mimir.logging.LoggingService;
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
    private final LoggingService logging;

    public MetadataErrors(LoggingService logging) {
        this.logging = logging;
    }

    @ExceptionHandler(DataIntegrityViolationException.class)
    ProblemDetail conflict(DataIntegrityViolationException exception) {
        logging.warn(MetadataErrors.class, "Metadata constraint violation");
        return ProblemDetail.forStatusAndDetail(HttpStatus.CONFLICT,
                "A name or key already exists, a referenced record is missing, or this record is still referenced.");
    }

    @ExceptionHandler(DbActionExecutionException.class)
    ResponseEntity<ProblemDetail> persistence(DbActionExecutionException exception) {
        for (Throwable cause = exception.getCause(); cause != null; cause = cause.getCause()) {
            if (cause instanceof DataIntegrityViolationException integrity) {
                return ResponseEntity.status(HttpStatus.CONFLICT).body(conflict(integrity));
            }
        }
        logging.error(MetadataErrors.class, "Metadata persistence failed");
        return ResponseEntity.internalServerError().body(
                ProblemDetail.forStatusAndDetail(HttpStatus.INTERNAL_SERVER_ERROR, "Metadata could not be saved."));
    }
}
