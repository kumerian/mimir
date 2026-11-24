package io.mimir.loader;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.web.servlet.MockMvc;

@SpringBootTest(properties = {
        "spring.datasource.url=jdbc:h2:mem:flatfile_metadata;MODE=MariaDB;DATABASE_TO_LOWER=TRUE;DB_CLOSE_DELAY=-1",
        "spring.datasource.username=sa",
        "spring.datasource.password=",
        "mimir.loaded-data.datasource.url=jdbc:h2:mem:flatfile_loaded;MODE=MariaDB;DATABASE_TO_LOWER=TRUE;DB_CLOSE_DELAY=-1",
        "mimir.loaded-data.datasource.username=sa",
        "mimir.loaded-data.datasource.password="
})
@AutoConfigureMockMvc
class FlatfileLoaderIntegrationTest {
    @Autowired MockMvc mvc;
    @Autowired ObjectMapper json;
    @Autowired @Qualifier("loadedDataJdbcTemplate") JdbcTemplate loadedData;
    @Autowired @Qualifier("metadataJdbcTemplate") JdbcTemplate metadata;
    @TempDir Path temporaryDirectory;

    @Test
    void derivesAndUpdatesObjectAndColumnsFromCsvHeaderAndValues() throws Exception {
        Path file = temporaryDirectory.resolve("metadata.csv");
        Files.writeString(file, "name;amount;active\n\"Alice; Inc.\";1,234.50;true\nBob;2.50;false\n");
        String interfaceId = createInterface(file);
        int previousColumns = getJson("/api/metadata/columns").size();

        JsonNode first = postJson("/api/loader/interfaces/" + interfaceId + "/metadata", Map.of());
        assertEquals("sample", first.path("object").path("objectName").asText());
        String objectId = first.path("object").path("objectId").asText();
        assertEquals(2, first.path("rowsScanned").asInt());
        assertEquals(List.of("name", "amount", "active"), columnNames(first.path("columns")));
        assertEquals("DECIMAL(6,2)", first.path("columns").get(1).path("datatype").asText());
        assertEquals("BOOLEAN", first.path("columns").get(2).path("datatype").asText());
        assertEquals(1, first.path("columns").get(0).path("columnOrdinalPosition").asInt());

        Files.writeString(file, "code;amount\nA;10.25\n");
        JsonNode updated = postJson("/api/loader/interfaces/" + interfaceId + "/metadata", Map.of());
        assertTrue(updated.path("updated").asBoolean());
        assertEquals(objectId, updated.path("object").path("objectId").asText());
        assertEquals(2, updated.path("removedColumns").asInt());
        assertEquals(List.of("code", "amount"), columnNames(updated.path("columns")));
        assertEquals(previousColumns + 2, getJson("/api/metadata/columns").size());
    }

    @Test
    void rejectsMissingAndInvalidHeadersWithoutCreatingMetadata() throws Exception {
        String missingInterface = createInterface(temporaryDirectory.resolve("missing.csv"));
        int previousObjects = getJson("/api/metadata/objects").size();
        mvc.perform(post("/api/loader/interfaces/" + missingInterface + "/metadata"))
                .andExpect(status().isBadRequest());
        assertEquals(previousObjects, getJson("/api/metadata/objects").size());

        Path duplicate = temporaryDirectory.resolve("duplicate.csv");
        Files.writeString(duplicate, "name;NAME\nA;B\n");
        String duplicateInterface = createInterface(duplicate);
        JsonNode failure = postFailure("/api/loader/interfaces/" + duplicateInterface + "/metadata");
        assertTrue(failure.path("detail").asText().contains("unique"));
        assertEquals(previousObjects, getJson("/api/metadata/objects").size());
    }

    @Test
    void preservesExistingObjectsWhenMetadataDerivationIsAmbiguous() throws Exception {
        Path file = temporaryDirectory.resolve("ambiguous.csv");
        Files.writeString(file, "name\nAlice\n");
        String interfaceId = createInterface(file);
        postCreatedJson("/api/metadata/objects", object("existing-a", interfaceId));
        postCreatedJson("/api/metadata/objects", object("existing-b", interfaceId));

        JsonNode failure = postFailure("/api/loader/interfaces/" + interfaceId + "/metadata");
        assertTrue(failure.path("detail").asText().contains("multiple Objects"));
        assertEquals(
                List.of("existing-a", "existing-b"),
                java.util.stream.StreamSupport.stream(
                                getJson("/api/metadata/objects").spliterator(), false)
                        .filter(objectRecord -> interfaceId.equals(objectRecord.path("interfaceId").asText()))
                        .map(objectRecord -> objectRecord.path("objectName").asText())
                        .sorted()
                        .toList());
    }

    @Test
    void loadsRowsIntoLoadedDatabaseAndRollsBackMalformedRows() throws Exception {
        Path file = temporaryDirectory.resolve("load.csv");
        Files.writeString(file, "name;amount;active\nAlice;12.50;true\nBob;3.25;false\n");
        String interfaceId = createInterface(file);
        JsonNode metadataResult = postJson("/api/loader/interfaces/" + interfaceId + "/metadata", Map.of());
        String objectId = metadataResult.path("object").path("objectId").asText();
        String table = FlatfileLoaderService.tableName(objectId);

        Files.writeString(file, "name;amount;active\nAlice;12.50;true\nMalformed;4.50\n");
        JsonNode malformed = postFailure("/api/loader/interfaces/" + interfaceId + "/load");
        assertTrue(malformed.path("detail").asText().contains("expected 3"));
        assertTrue(malformed.path("detail").asText().contains("destination table may already exist"));
        assertEquals(0, loadedData.queryForObject("SELECT COUNT(*) FROM `" + table + "`", Integer.class));

        Files.writeString(file, "name;amount;active\nAlice;12.50;true\nBob;3.25;false\n");
        JsonNode loaded = postJson("/api/loader/interfaces/" + interfaceId + "/load", Map.of());
        assertEquals(2, loaded.path("rowsLoaded").asInt());
        assertEquals("sample", loaded.path("objectName").asText());
        assertEquals(2, loadedData.queryForObject("SELECT COUNT(*) FROM `" + table + "`", Integer.class));
        assertEquals(
                12.50,
                loadedData.queryForObject(
                        "SELECT amount FROM `" + table + "` WHERE name = 'Alice'", Double.class));
        assertEquals(
                0,
                metadata.queryForObject(
                        "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?",
                        Integer.class,
                        table));

        Files.writeString(file, "name;amount\nAlice;12.50\n");
        JsonNode drift = postFailure("/api/loader/interfaces/" + interfaceId + "/load");
        assertTrue(drift.path("detail").asText().contains("header"));
        assertEquals(2, loadedData.queryForObject("SELECT COUNT(*) FROM `" + table + "`", Integer.class));
    }

    @Test
    void rejectsUnsavedAndNonCsvInterfaces() throws Exception {
        mvc.perform(post("/api/loader/interfaces/not-saved/metadata"))
                .andExpect(status().isNotFound());

        Path file = temporaryDirectory.resolve("jdbc.csv");
        Files.writeString(file, "name\nvalue\n");
        String interfaceId = createInterface(file, "jdbc");
        JsonNode failure = postFailure("/api/loader/interfaces/" + interfaceId + "/metadata");
        assertTrue(failure.path("detail").asText().contains("Only CSV"));
    }

    private String createInterface(Path file) throws Exception {
        return createInterface(file, "csv");
    }

    private String createInterface(Path file, String type) throws Exception {
        String sourceName = "source-" + UUID.randomUUID();
        postCreatedJson("/api/metadata/source-systems", Map.of(
                "sourceName", sourceName,
                "sourceFunctionalName", "Source",
                "sourceType", "flatfile",
                "sourceOwner", "Owner",
                "sourceApplication", "Tests",
                "sourceLocation", "local",
                "isActive", true));
        JsonNode result = postCreatedJson("/api/metadata/interfaces", Map.of(
                "interfaceName", "sample",
                "sourceName", sourceName,
                "interfaceFunctionalName", "Sample file",
                "interfaceType", type,
                "interfaceLocation", file.toString(),
                "decimalSeparator", ".",
                "thousandsSeparator", ",",
                "delimiter", ";",
                "hasHeaders", true,
                "isActive", true));
        return result.path("interfaceId").asText();
    }

    private Map<String, Object> object(String name, String interfaceId) {
        return Map.of(
                "objectName", name,
                "interfaceId", interfaceId,
                "objectFunctionalName", name,
                "objectDescription", "",
                "isConfidential", false,
                "isBusinessCritical", false,
                "isActive", true);
    }

    private JsonNode postJson(String path, Object body) throws Exception {
        var response = mvc.perform(post(path).contentType(MediaType.APPLICATION_JSON)
                        .content(json.writeValueAsBytes(body)))
                .andReturn().getResponse();
        assertEquals(200, response.getStatus(), response.getContentAsString());
        return json.readTree(response.getContentAsString());
    }

    private JsonNode postCreatedJson(String path, Object body) throws Exception {
        var response = mvc.perform(post(path).contentType(MediaType.APPLICATION_JSON)
                        .content(json.writeValueAsBytes(body)))
                .andReturn().getResponse();
        assertEquals(201, response.getStatus(), response.getContentAsString());
        return json.readTree(response.getContentAsString());
    }

    private JsonNode postFailure(String path) throws Exception {
        var response = mvc.perform(post(path).contentType(MediaType.APPLICATION_JSON).content("{}"))
                .andReturn().getResponse();
        assertFalse(response.getStatus() < 400, response.getContentAsString());
        return json.readTree(response.getContentAsString());
    }

    private JsonNode getJson(String path) throws Exception {
        var response = mvc.perform(org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get(path))
                .andReturn().getResponse();
        assertEquals(200, response.getStatus(), response.getContentAsString());
        return json.readTree(response.getContentAsString());
    }

    private List<String> columnNames(JsonNode columns) {
        List<String> names = new java.util.ArrayList<>();
        columns.forEach(column -> names.add(column.path("columnName").asText()));
        return names;
    }

}
