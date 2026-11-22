package io.mimir.status;

import org.junit.jupiter.api.Test;
import org.springframework.dao.DataAccessResourceFailureException;
import org.springframework.jdbc.core.JdbcTemplate;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class StatusControllerTest {
    private final JdbcTemplate jdbc = mock(JdbcTemplate.class);
    private final StatusController controller = new StatusController(jdbc);

    @Test
    void reportsAConnectedDatabaseAfterAQuery() {
        when(jdbc.queryForObject("SELECT 1", Integer.class)).thenReturn(1);

        assertThat(controller.status())
                .isEqualTo(new StatusController.Status("mimir", "connected"));
    }

    @Test
    void doesNotReportSuccessWhenTheDatabaseIsUnavailable() {
        when(jdbc.queryForObject("SELECT 1", Integer.class))
                .thenThrow(new DataAccessResourceFailureException("Database unavailable"));

        assertThatThrownBy(controller::status)
                .isInstanceOf(DataAccessResourceFailureException.class);
    }
}
