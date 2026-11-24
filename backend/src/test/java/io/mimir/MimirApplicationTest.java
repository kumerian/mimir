package io.mimir;

import io.mimir.logging.LoggingService;
import com.zaxxer.hikari.HikariDataSource;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.dao.DataAccessException;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.system.CapturedOutput;
import org.springframework.boot.test.system.OutputCaptureExtension;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.web.servlet.MockMvc;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@ExtendWith(OutputCaptureExtension.class)
@SpringBootTest(properties = {
        "spring.datasource.url=jdbc:h2:mem:mimir;MODE=MariaDB;DB_CLOSE_DELAY=-1",
        "spring.datasource.username=sa",
        "spring.datasource.password=",
        "mimir.loaded-data.datasource.url=jdbc:h2:mem:loaded_data;MODE=MariaDB;DB_CLOSE_DELAY=-1",
        "mimir.loaded-data.datasource.username=sa",
        "mimir.loaded-data.datasource.password="
})
@AutoConfigureMockMvc
class MimirApplicationTest {
    @Autowired
    private MockMvc mvc;

    @Autowired
    private LoggingService logging;

    @Autowired
    @Qualifier("loadedDataJdbcTemplate")
    private JdbcTemplate loadedDataJdbc;

    @Autowired
    @Qualifier("metadataJdbcTemplate")
    private JdbcTemplate metadataJdbc;

    @Autowired
    @Qualifier("loadedDataDataSource")
    private HikariDataSource loadedDataDataSource;

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
    void loadedDataJdbcTemplateUsesASeparateDatabase() {
        loadedDataJdbc.execute("CREATE TABLE loaded_probe (loaded_value varchar(20))");
        loadedDataJdbc.update("INSERT INTO loaded_probe (loaded_value) VALUES (?)", "loaded");

        org.junit.jupiter.api.Assertions.assertEquals(
                "loaded", loadedDataJdbc.queryForObject(
                        "SELECT loaded_value FROM loaded_probe", String.class));
        org.junit.jupiter.api.Assertions.assertEquals(
                0,
                metadataJdbc.queryForObject(
                        "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'loaded_probe'",
                        Integer.class));
    }

    @Test
    @DirtiesContext
    void surfacesLoadedDataConnectionFailures() {
        loadedDataDataSource.close();

        org.junit.jupiter.api.Assertions.assertThrows(
                DataAccessException.class,
                () -> loadedDataJdbc.queryForObject("SELECT 1", Integer.class));
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
