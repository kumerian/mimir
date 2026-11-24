package io.mimir.loader;

import io.mimir.logging.LoggingService;
import org.springframework.dao.DataAccessException;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.data.relational.core.conversion.DbActionExecutionException;
import org.springframework.http.HttpStatus;
import org.springframework.http.ProblemDetail;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.servlet.mvc.method.annotation.ResponseEntityExceptionHandler;

@RestControllerAdvice(assignableTypes = FlatfileLoaderController.class)
public class FlatfileLoaderErrors extends ResponseEntityExceptionHandler {
    private final LoggingService logging;

    public FlatfileLoaderErrors(LoggingService logging) {
        this.logging = logging;
    }

    @ExceptionHandler(LoaderException.class)
    ResponseEntity<ProblemDetail> loader(LoaderException exception) {
        if (exception.status().is5xxServerError()) {
            logging.error(FlatfileLoaderErrors.class, "CSV loader operation failed");
        } else {
            logging.warn(FlatfileLoaderErrors.class, "CSV loader operation was rejected");
        }
        return ResponseEntity.status(exception.status())
                .body(ProblemDetail.forStatusAndDetail(exception.status(), exception.getMessage()));
    }

    @ExceptionHandler({DataIntegrityViolationException.class, DbActionExecutionException.class})
    ResponseEntity<ProblemDetail> metadataConflict(RuntimeException exception) {
        logging.warn(FlatfileLoaderErrors.class, "CSV metadata constraint violation");
        return ResponseEntity.status(HttpStatus.CONFLICT).body(
                ProblemDetail.forStatusAndDetail(
                        HttpStatus.CONFLICT,
                        "A metadata name conflicts with an existing record; no metadata changes were saved."));
    }

    @ExceptionHandler(DataAccessException.class)
    ResponseEntity<ProblemDetail> database(DataAccessException exception) {
        logging.error(FlatfileLoaderErrors.class, "CSV loader database operation failed");
        return ResponseEntity.internalServerError().body(
                ProblemDetail.forStatusAndDetail(
                        HttpStatus.INTERNAL_SERVER_ERROR,
                        "The database operation failed; no success was reported. A destination table may have been created."));
    }
}
