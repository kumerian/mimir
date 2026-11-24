package io.mimir.logging;

import io.mimir.MimirApplication;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.context.event.EventListener;
import org.springframework.stereotype.Component;

@Component
public class ApplicationStartupLogger {
    private final LoggingService logging;

    public ApplicationStartupLogger(LoggingService logging) {
        this.logging = logging;
    }

    @EventListener(ApplicationReadyEvent.class)
    public void applicationReady() {
        logging.info(MimirApplication.class, "Application started");
    }
}
