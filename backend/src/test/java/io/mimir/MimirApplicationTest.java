package io.mimir;

import io.mimir.logging.LoggingService;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.system.CapturedOutput;
import org.springframework.boot.test.system.OutputCaptureExtension;
import org.springframework.test.web.servlet.MockMvc;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@ExtendWith(OutputCaptureExtension.class)
@SpringBootTest(properties = {
        "spring.datasource.url=jdbc:h2:mem:mimir;MODE=MariaDB;DB_CLOSE_DELAY=-1",
        "spring.datasource.username=sa",
        "spring.datasource.password="
})
@AutoConfigureMockMvc
class MimirApplicationTest {
    @Autowired
    private MockMvc mvc;

    @Autowired
    private LoggingService logging;

    @Test
    void exposesDatabaseBackedStatus() throws Exception {
        mvc.perform(get("/api/status"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.application").value("mimir"))
                .andExpect(jsonPath("$.database").value("connected"));
    }

    @Test
    void exposesHealthWithDatabaseConnectivity() throws Exception {
        mvc.perform(get("/actuator/health"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.status").value("UP"));
    }

    @Test
    void printsTimestampLevelSourceAndMessage(CapturedOutput output) {
        logging.info(MimirApplicationTest.class, "terminal format marker");

        org.junit.jupiter.api.Assertions.assertTrue(
                java.util.regex.Pattern.compile(
                        "(?m)^\\d{4}-\\d{2}-\\d{2}T[^ ]+ INFO  io\\.mimir\\.MimirApplicationTest - terminal format marker$")
                        .matcher(output.getOut())
                        .find());
    }
}
