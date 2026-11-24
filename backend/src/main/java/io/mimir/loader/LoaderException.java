package io.mimir.loader;

import org.springframework.http.HttpStatus;

public class LoaderException extends RuntimeException {
    private final HttpStatus status;

    public LoaderException(HttpStatus status, String message) {
        super(message);
        this.status = status;
    }

    public LoaderException(HttpStatus status, String message, Throwable cause) {
        super(message, cause);
        this.status = status;
    }

    public HttpStatus status() {
        return status;
    }
}
