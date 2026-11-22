package io.mimir.metadata;

import java.util.List;
import java.util.Map;
import java.util.stream.Stream;
import java.sql.DriverManager;

import com.fasterxml.jackson.databind.ObjectMapper;
import liquibase.Contexts;
import liquibase.LabelExpression;
import liquibase.Liquibase;
import liquibase.database.jvm.JdbcConnection;
import liquibase.resource.ClassLoaderResourceAccessor;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.Arguments;
import org.junit.jupiter.params.provider.MethodSource;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.transaction.annotation.Transactional;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@SpringBootTest(properties = {
        "spring.datasource.url=jdbc:h2:mem:metadata;MODE=MariaDB;DATABASE_TO_LOWER=TRUE;DB_CLOSE_DELAY=-1",
        "spring.datasource.username=sa",
        "spring.datasource.password="
})
@AutoConfigureMockMvc
@Transactional
class MetadataApiTest {
    @Autowired MockMvc mvc;
    @Autowired ObjectMapper json;
    @Autowired JdbcTemplate jdbc;

    private static final Map<String, Object> SOURCE = Map.of(
            "sourceName", "source", "sourceFunctionalName", "Source", "sourceType", "database",
            "sourceOwner", "Owner", "sourceApplication", "CRM", "sourceLocation", "https://source",
            "isActive", true);
    private static final Map<String, Object> AUTH = Map.of(
            "authenticationId", "auth", "authenticationType", "vault", "authenticationLocation", "vault://mimir",
            "isActive", true);
    private static final Map<String, Object> INTERFACE = interfaceSample();
    private static final Map<String, Object> OBJECT = Map.of(
            "objectName", "object", "interfaceName", "interface", "objectFunctionalName", "Object",
            "objectDescription", "Description", "isConfidential", false, "isBusinessCritical", true, "isActive", true);
    private static final Map<String, Object> COLUMN = columnSample();
    private static Map<String, Object> interfaceSample() {
        var value = new java.util.HashMap<String, Object>();
        value.put("interfaceName", "interface");
        value.put("sourceName", "source");
        value.put("authenticationId", "auth");
        value.put("interfaceFunctionalName", "Interface");
        value.put("interfaceType", "jdbc");
        value.put("interfaceLocation", "jdbc:mariadb://source");
        value.put("decimalSeparator", ".");
        value.put("thousandsSeparator", ",");
        value.put("delimiter", ";");
        value.put("hasHeaders", true);
        value.put("isActive", true);
        return value;
    }
    private static Map<String, Object> columnSample() {
        var value = new java.util.HashMap<String, Object>();
        value.put("columnName", "column");
        value.put("objectName", "object");
        value.put("columnAlternateName", "col");
        value.put("columnFunctionalName", "Column");
        value.put("columnDescription", "Description");
        value.put("columnOrdinalPosition", 1);
        value.put("needsEncryption", false);
        value.put("isPartitionField", false);
        value.put("partitionOrdinalPosition", 0);
        value.put("isKey", true);
        value.put("keyOrdinalPosition", 1);
        value.put("isNullable", false);
        value.put("testDatatype", true);
        value.put("datatype", "varchar");
        value.put("characterMaximumLength", 255);
        value.put("numericScale", null);
        value.put("numericPrecision", null);
        value.put("nullValueRegexPattern", null);
        value.put("isActive", true);
        return value;
    }

    static Stream<Arguments> entities() {
        return Stream.of(
                Arguments.of("source-systems", "sourceName", SOURCE, "sourceFunctionalName"),
                Arguments.of("authentications", "authenticationId", AUTH, "authenticationType"),
                Arguments.of("interfaces", "interfaceName", INTERFACE, "interfaceFunctionalName"),
                Arguments.of("objects", "objectName", OBJECT, "objectFunctionalName"),
                Arguments.of("columns", "columnName", COLUMN, "datatype"));
    }

    private String path(String resource) { return "/api/metadata/" + resource.replace("/", "?id="); }

    private void create(String resource, Map<String, ?> value) throws Exception {
        mvc.perform(post(path(resource)).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(value)))
                .andExpect(status().isCreated())
                .andExpect(content().json(json.writeValueAsString(value)));
    }

    private void parents() throws Exception {
        create("source-systems", SOURCE);
        create("authentications", AUTH);
        create("interfaces", INTERFACE);
        create("objects", OBJECT);
    }

    @ParameterizedTest
    @MethodSource("entities")
    void crudAndImmutableKeys(String resource, String key, Map<String, ?> sample, String editable) throws Exception {
        if (List.of("interfaces", "objects", "columns").contains(resource)) {
            create("source-systems", SOURCE);
            create("authentications", AUTH);
        }
        if (List.of("objects", "columns").contains(resource)) create("interfaces", INTERFACE);
        if (resource.equals("columns")) create("objects", OBJECT);

        String id = String.valueOf(sample.get(key));
        String item = path(resource) + "?id=" + id;
        mvc.perform(get(path(resource))).andExpect(status().isOk()).andExpect(content().json("[]"));
        mvc.perform(post(path(resource)).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(sample)))
                .andExpect(status().isCreated()).andExpect(header().string("Location", "http://localhost" + item));
        mvc.perform(get(item)).andExpect(status().isOk()).andExpect(content().json(json.writeValueAsString(sample)));
        mvc.perform(get(path(resource))).andExpect(status().isOk()).andExpect(jsonPath("$[0]." + key).value(id));
        mvc.perform(post(path(resource)).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(sample)))
                .andExpect(status().isConflict()).andExpect(jsonPath("$.detail").isNotEmpty());

        var updated = new java.util.HashMap<String, Object>();
        updated.putAll(sample);
        updated.put(editable, "Updated");
        mvc.perform(put(item).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(updated)))
                .andExpect(status().isOk()).andExpect(content().json(json.writeValueAsString(updated)));
        mvc.perform(get(item)).andExpect(content().json(json.writeValueAsString(updated)));
        updated.put(key, "renamed");
        mvc.perform(put(item).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(updated)))
                .andExpect(status().isBadRequest()).andExpect(jsonPath("$.detail").value("Primary keys cannot be changed"));
        mvc.perform(delete(item)).andExpect(status().isNoContent());
        mvc.perform(get(item)).andExpect(status().isNotFound());
        mvc.perform(delete(item)).andExpect(status().isNotFound());
        mvc.perform(put(item).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(sample)))
                .andExpect(status().isNotFound());
    }

    @ParameterizedTest
    @MethodSource("entities")
    void validatesRequiredFieldsAndLengths(String resource, String key, Map<String, ?> sample, String editable) throws Exception {
        if (List.of("interfaces", "objects", "columns").contains(resource)) {
            create("source-systems", SOURCE);
            create("authentications", AUTH);
        }
        if (List.of("objects", "columns").contains(resource)) create("interfaces", INTERFACE);
        if (resource.equals("columns")) create("objects", OBJECT);
        for (String field : sample.keySet()) {
            if (!(sample.get(field) instanceof String) && sample.get(field) != null) continue;
            boolean optional = List.of(
                    "sourceFunctionalName", "sourceOwner", "sourceLocation",
                    "authenticationId", "interfaceFunctionalName", "delimiter", "hasHeaders",
                    "objectFunctionalName", "objectDescription", "columnAlternateName",
                    "columnFunctionalName", "columnDescription", "characterMaximumLength",
                    "numericScale", "numericPrecision", "nullValueRegexPattern").contains(field);
            if (optional) continue;
            var invalid = new java.util.HashMap<String, Object>();
            invalid.putAll(sample);
            invalid.put(field, " ");
            mvc.perform(post(path(resource)).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(invalid)))
                    .andExpect(status().isBadRequest());
            invalid.put(field, "x".repeat(field.endsWith("Location") ? 2049 : 256));
            mvc.perform(post(path(resource)).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(invalid)))
                    .andExpect(status().isBadRequest());
            if (!optional) {
                invalid.remove(field);
                mvc.perform(post(path(resource)).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(invalid)))
                        .andExpect(status().isBadRequest());
            }
        }
        var invalid = new java.util.HashMap<String, Object>();
        invalid.putAll(sample);
        invalid.put("unexpected", "value");
        mvc.perform(post(path(resource)).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(invalid)))
                .andExpect(status().isBadRequest());
    }

    @Test
    void enforcesAllForeignKeysAndRestrictsDeletion() throws Exception {
        for (var entry : Map.of("interfaces", INTERFACE, "objects", OBJECT, "columns", COLUMN).entrySet()) {
            mvc.perform(post(path(entry.getKey())).contentType(MediaType.APPLICATION_JSON)
                    .content(json.writeValueAsBytes(entry.getValue()))).andExpect(status().isConflict());
        }
        parents();
        create("columns", COLUMN);
        for (String item : List.of("source-systems/source", "authentications/auth", "interfaces/interface", "objects/object")) {
            mvc.perform(delete(path(item))).andExpect(status().isConflict());
        }
        var invalidAuth = new java.util.HashMap<>(INTERFACE);
        invalidAuth.put("authenticationId", "missing");
        mvc.perform(put(path("interfaces/interface")).contentType(MediaType.APPLICATION_JSON)
                .content(json.writeValueAsBytes(invalidAuth))).andExpect(status().isConflict());
        mvc.perform(get(path("interfaces/interface"))).andExpect(jsonPath("$.authenticationId").value("auth"));
    }

    @Test
    void supportsOptionalSharedAuthenticationAndReparenting() throws Exception {
        parents();
        var second = new java.util.HashMap<>(INTERFACE);
        second.put("interfaceName", "second");
        create("interfaces", second);
        second.remove("authenticationId");
        mvc.perform(put(path("interfaces/second")).contentType(MediaType.APPLICATION_JSON)
                .content(json.writeValueAsBytes(second))).andExpect(status().isOk()).andExpect(jsonPath("$.authenticationId").isEmpty());
        var moved = new java.util.HashMap<>(OBJECT);
        moved.put("interfaceName", "second");
        mvc.perform(put(path("objects/object")).contentType(MediaType.APPLICATION_JSON)
                .content(json.writeValueAsBytes(moved))).andExpect(status().isOk());
        mvc.perform(delete(path("interfaces/interface"))).andExpect(status().isNoContent());
        mvc.perform(delete(path("authentications/auth"))).andExpect(status().isNoContent());
    }

    @Test
    void acceptsMaximumLengthsAndSortsListsByKey() throws Exception {
        create("source-systems", Map.of("sourceName", "z".repeat(255),
                "sourceFunctionalName", "x".repeat(255), "sourceType", "x".repeat(255),
                "sourceOwner", "Owner", "sourceApplication", "x".repeat(255),
                "sourceLocation", "x".repeat(255), "isActive", true));
        create("source-systems", SOURCE);
        create("authentications", Map.of("authenticationId", "a".repeat(255),
                "authenticationType", "x".repeat(255), "authenticationLocation", "x".repeat(2048),
                "isActive", true));
        mvc.perform(get(path("source-systems"))).andExpect(jsonPath("$[0].sourceName").value("source"))
                .andExpect(jsonPath("$[1].sourceName").value("z".repeat(255)));
    }

    @Test
    void liquibaseOwnsSchemaAndMigrationHistory() {
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM databasechangelog WHERE id = '001-ingestion-metadata'", Integer.class));
        assertEquals(0, jdbc.queryForObject("SELECT COUNT(*) FROM source_column", Integer.class));
    }

    @Test
    void namesWithReservedUrlCharactersRemainEditable() throws Exception {
        var record = Map.of("sourceName", "crm/table & name+?#", "sourceFunctionalName", "CRM", "sourceType", "database",
                "sourceOwner", "Owner", "sourceApplication", "CRM", "sourceLocation", "https://crm", "isActive", true);
        mvc.perform(post(path("source-systems")).queryParam("ignored", "value")
                .contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(record)))
                .andExpect(status().isCreated())
                .andExpect(header().string("Location",
                        "http://localhost/api/metadata/source-systems?id=crm%2Ftable%20%26%20name%2B%3F%23"));
        mvc.perform(get(path("source-systems")).param("id", String.valueOf(record.get("sourceName"))))
                .andExpect(status().isOk()).andExpect(content().json(json.writeValueAsString(record)));
        mvc.perform(put(path("source-systems")).param("id", String.valueOf(record.get("sourceName")))
                .contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(record)))
                .andExpect(status().isOk());
        mvc.perform(delete(path("source-systems")).param("id", String.valueOf(record.get("sourceName"))))
                .andExpect(status().isNoContent());
    }

    @Test
    void migrationCanBeReappliedAndRolledBackInAnIsolatedDatabase() throws Exception {
        try (var connection = DriverManager.getConnection(
                "jdbc:h2:mem:migration-rollback;MODE=MariaDB", "sa", "");
             var resources = new ClassLoaderResourceAccessor();
             var liquibase = new Liquibase("db/changelog/db.changelog-master.xml", resources,
                     new JdbcConnection(connection))) {
            liquibase.update(new Contexts(), new LabelExpression());
            try (var statement = connection.createStatement()) {
                statement.executeUpdate("INSERT INTO source_system VALUES ('source', 'Source', 'database', NULL, 'CRM', NULL, TRUE)");
                liquibase.update(new Contexts(), new LabelExpression());
                try (var rows = statement.executeQuery("SELECT COUNT(*) FROM source_system")) {
                    rows.next();
                    assertEquals(1, rows.getInt(1));
                }
                liquibase.rollback(10, new Contexts(), new LabelExpression());
                try (var tables = connection.getMetaData().getTables(null, null, "SOURCE_SYSTEM", null)) {
                    org.junit.jupiter.api.Assertions.assertFalse(tables.next());
                }
            }
        }
    }
}
