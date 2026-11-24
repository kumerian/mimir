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
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@SpringBootTest(properties = {
        "spring.datasource.url=jdbc:h2:mem:metadata;MODE=MariaDB;DATABASE_TO_LOWER=TRUE;DB_CLOSE_DELAY=-1",
        "spring.datasource.username=sa",
        "spring.datasource.password=",
        "mimir.loaded-data.datasource.url=jdbc:h2:mem:loaded_api_test;MODE=MariaDB;DB_CLOSE_DELAY=-1",
        "mimir.loaded-data.datasource.username=sa",
        "mimir.loaded-data.datasource.password="
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
            "authenticationUser", "service-user", "authenticationCredential", "credential-reference",
            "isActive", true);
    private static final Map<String, Object> INTERFACE = interfaceSample();
    private static final Map<String, Object> API_PARAMETER = Map.of(
            "parameterId", "parameter", "interfaceId", "pending-interface", "parameterName", "api-version",
            "parameterValue", "v1");
    private static final Map<String, Object> OBJECT = Map.of(
            "objectName", "object", "interfaceId", "pending-interface", "objectFunctionalName", "Object",
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
        value.put("objectId", "pending-object");
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
                Arguments.of("api-parameters", "parameterId", API_PARAMETER, "parameterName"),
                Arguments.of("interfaces", "interfaceId", INTERFACE, "interfaceFunctionalName"),
                Arguments.of("objects", "objectId", OBJECT, "objectFunctionalName"),
                Arguments.of("columns", "columnId", COLUMN, "datatype"));
    }

    private String path(String resource) { return "/api/metadata/" + resource.replace("/", "?id="); }

    private Map<String, Object> create(String resource, Map<String, ?> value) throws Exception {
        var response = mvc.perform(post(path(resource)).contentType(MediaType.APPLICATION_JSON)
                        .content(json.writeValueAsBytes(value)))
                .andReturn().getResponse();
        assertEquals(201, response.getStatus(), response.getContentAsString());
        Map<String, Object> saved = json.readValue(response.getContentAsString(), json.getTypeFactory()
                .constructMapType(java.util.HashMap.class, String.class, Object.class));
        value.forEach((field, expected) -> assertEquals(expected, saved.get(field), field));
        return saved;
    }

    private Map<String, Object> parents() throws Exception {
        create("source-systems", SOURCE);
        create("authentications", AUTH);
        return create("interfaces", INTERFACE);
    }

    private Map<String, Object> prepare(String resource) throws Exception {
        Map<String, Object> sample = new java.util.HashMap<>(switch (resource) {
            case "api-parameters" -> API_PARAMETER;
            case "interfaces" -> INTERFACE;
            case "objects" -> OBJECT;
            case "columns" -> COLUMN;
            case "source-systems" -> SOURCE;
            case "authentications" -> AUTH;
            default -> throw new IllegalArgumentException(resource);
        });
        if (List.of("interfaces", "objects", "columns", "api-parameters").contains(resource)) {
            create("source-systems", SOURCE);
            create("authentications", AUTH);
            Map<String, Object> sourceInterface = null;
            if (!resource.equals("interfaces")) sourceInterface = create("interfaces", INTERFACE);
            if (resource.equals("objects") || resource.equals("columns")) {
                var sourceObject = new java.util.HashMap<>(OBJECT);
                sourceObject.put("objectName", "parent-object");
                sourceObject.put("interfaceId", sourceInterface.get("interfaceId"));
                Map<String, Object> savedObject = create("objects", sourceObject);
                if (resource.equals("columns")) sample.put("objectId", savedObject.get("objectId"));
            }
            if (resource.equals("objects") || resource.equals("api-parameters"))
                sample.put("interfaceId", sourceInterface.get("interfaceId"));
        }
        return sample;
    }

    @ParameterizedTest
    @MethodSource("entities")
    void crudAndImmutableKeys(String resource, String key, Map<String, ?> ignored, String editable) throws Exception {
        mvc.perform(get(path(resource))).andExpect(status().isOk()).andExpect(content().json("[]"));
        Map<String, Object> sample = prepare(resource);
        Map<String, Object> saved = create(resource, sample);
        String id = String.valueOf(saved.get(key));
        String item = path(resource) + "?id=" + id;
        mvc.perform(get(item)).andExpect(status().isOk()).andExpect(content().json(json.writeValueAsString(saved)));
        mvc.perform(get(path(resource))).andExpect(status().isOk())
                .andExpect(jsonPath("$[*]." + key, org.hamcrest.Matchers.hasItem(id)));
        mvc.perform(post(path(resource)).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(sample)))
                .andExpect(status().isConflict()).andExpect(jsonPath("$.detail").isNotEmpty());

        var updated = new java.util.HashMap<String, Object>();
        updated.putAll(saved);
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
        mvc.perform(put(item).contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(saved)))
                .andExpect(status().isNotFound());
    }

    @ParameterizedTest
    @MethodSource("entities")
    void validatesRequiredFieldsAndLengths(String resource, String key, Map<String, ?> ignored, String editable) throws Exception {
        Map<String, Object> sample = prepare(resource);
        for (String field : sample.keySet()) {
            if (!(sample.get(field) instanceof String)) continue;
            boolean optional = List.of(
                    "sourceFunctionalName", "sourceOwner", "sourceLocation",
                    "authenticationUser", "authenticationCredential", "parameterValue",
                    "authenticationId", "interfaceFunctionalName", "delimiter", "hasHeaders",
                    "objectFunctionalName", "objectDescription", "columnAlternateName",
                    "columnFunctionalName", "columnDescription", "characterMaximumLength",
                    "numericScale", "numericPrecision", "nullValueRegexPattern").contains(field);
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
        for (var entry : Map.of("interfaces", INTERFACE, "objects", OBJECT, "columns", COLUMN,
                "api-parameters", API_PARAMETER).entrySet()) {
            mvc.perform(post(path(entry.getKey())).contentType(MediaType.APPLICATION_JSON)
                    .content(json.writeValueAsBytes(entry.getValue()))).andExpect(status().isConflict());
        }
        create("source-systems", SOURCE);
        create("authentications", AUTH);
        Map<String, Object> sourceInterface = create("interfaces", INTERFACE);
        var sourceObject = new java.util.HashMap<>(OBJECT);
        sourceObject.put("interfaceId", sourceInterface.get("interfaceId"));
        Map<String, Object> savedObject = create("objects", sourceObject);
        var sourceColumn = new java.util.HashMap<>(COLUMN);
        sourceColumn.put("objectId", savedObject.get("objectId"));
        create("columns", sourceColumn);
        var parameter = new java.util.HashMap<>(API_PARAMETER);
        parameter.put("interfaceId", sourceInterface.get("interfaceId"));
        create("api-parameters", parameter);
        for (String item : List.of("source-systems/source", "authentications/auth")) {
            mvc.perform(delete(path(item))).andExpect(status().isConflict());
        }
        mvc.perform(delete(path("interfaces")).param("id", String.valueOf(sourceInterface.get("interfaceId"))))
                .andExpect(status().isConflict());
        mvc.perform(delete(path("objects")).param("id", String.valueOf(savedObject.get("objectId"))))
                .andExpect(status().isConflict());
        var invalidAuth = new java.util.HashMap<>(INTERFACE);
        invalidAuth.put("interfaceId", sourceInterface.get("interfaceId"));
        invalidAuth.put("authenticationId", "missing");
        mvc.perform(put(path("interfaces")).param("id", String.valueOf(sourceInterface.get("interfaceId")))
                .contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(invalidAuth)))
                .andExpect(status().isConflict());
        mvc.perform(get(path("interfaces")).param("id", String.valueOf(sourceInterface.get("interfaceId"))))
                .andExpect(jsonPath("$.authenticationId").value("auth"));
        var invalidInterface = new java.util.HashMap<>(parameter);
        invalidInterface.put("interfaceId", "missing");
        mvc.perform(post(path("api-parameters")).contentType(MediaType.APPLICATION_JSON)
                .content(json.writeValueAsBytes(invalidInterface))).andExpect(status().isConflict());
    }

    @Test
    void supportsOptionalSharedAuthenticationAndReparenting() throws Exception {
        Map<String, Object> first = parents();
        var second = new java.util.HashMap<>(INTERFACE);
        second.put("interfaceName", "second");
        Map<String, Object> savedSecond = create("interfaces", second);
        var parameter = new java.util.HashMap<>(API_PARAMETER);
        parameter.put("interfaceId", first.get("interfaceId"));
        create("api-parameters", parameter);
        second.remove("authenticationId");
        second.put("interfaceId", savedSecond.get("interfaceId"));
        var secondUpdate = mvc.perform(put(path("interfaces")).param("id", String.valueOf(savedSecond.get("interfaceId")))
                .contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(second)))
                .andReturn().getResponse();
        assertEquals(200, secondUpdate.getStatus(), secondUpdate.getContentAsString());
        assertTrue(secondUpdate.getContentAsString().contains("\"authenticationId\":null"));
        var moved = new java.util.HashMap<>(OBJECT);
        moved.put("interfaceId", first.get("interfaceId"));
        Map<String, Object> savedObject = create("objects", moved);
        moved.put("objectId", savedObject.get("objectId"));
        moved.put("interfaceId", savedSecond.get("interfaceId"));
        mvc.perform(put(path("objects")).param("id", String.valueOf(savedObject.get("objectId")))
                .contentType(MediaType.APPLICATION_JSON)
                .content(json.writeValueAsBytes(moved))).andExpect(status().isOk());
        mvc.perform(delete(path("api-parameters")).param("id", "parameter")).andExpect(status().isNoContent());
        mvc.perform(delete(path("interfaces")).param("id", String.valueOf(first.get("interfaceId"))))
                .andExpect(status().isNoContent());
        mvc.perform(delete(path("objects")).param("id", String.valueOf(savedObject.get("objectId"))))
                .andExpect(status().isNoContent());
        mvc.perform(delete(path("authentications/auth"))).andExpect(status().isNoContent());
    }

    @Test
    void scopesNamesByHierarchyAndKeepsRelationshipsOnParentChanges() throws Exception {
        create("source-systems", SOURCE);
        var otherSource = new java.util.HashMap<>(SOURCE);
        otherSource.put("sourceName", "other-source");
        create("source-systems", otherSource);
        create("authentications", AUTH);

        Map<String, Object> first = create("interfaces", INTERFACE);
        var sameNameOtherSource = new java.util.HashMap<>(INTERFACE);
        sameNameOtherSource.put("sourceName", "other-source");
        Map<String, Object> second = create("interfaces", sameNameOtherSource);
        var sameSourceOtherName = new java.util.HashMap<>(INTERFACE);
        sameSourceOtherName.put("interfaceName", "another-interface");
        Map<String, Object> third = create("interfaces", sameSourceOtherName);
        assertEquals(3, jdbc.queryForObject("SELECT COUNT(*) FROM source_system_interface", Integer.class));
        mvc.perform(post(path("interfaces")).contentType(MediaType.APPLICATION_JSON)
                .content(json.writeValueAsBytes(INTERFACE))).andExpect(status().isConflict());

        Map<String, Object> firstObject = create("objects", object("first", first));
        Map<String, Object> secondObject = create("objects", object("second", second));
        Map<String, Object> thirdObject = create("objects", object("third", third));
        Map<String, Object> fourthObject = create("objects", object("first", third));
        mvc.perform(post(path("objects")).contentType(MediaType.APPLICATION_JSON)
                .content(json.writeValueAsBytes(object("first", first)))).andExpect(status().isConflict());

        Map<String, Object> firstColumn = create("columns", column("shared", firstObject));
        Map<String, Object> secondColumn = create("columns", column("shared", secondObject));
        create("columns", column("shared", thirdObject));
        mvc.perform(post(path("columns")).contentType(MediaType.APPLICATION_JSON)
                .content(json.writeValueAsBytes(column("shared", firstObject)))).andExpect(status().isConflict());

        var firstParameter = new java.util.HashMap<>(API_PARAMETER);
        firstParameter.put("parameterId", "first-parameter");
        firstParameter.put("interfaceId", first.get("interfaceId"));
        Map<String, Object> savedParameter = create("api-parameters", firstParameter);
        var secondParameter = new java.util.HashMap<>(API_PARAMETER);
        secondParameter.put("parameterId", "second-parameter");
        secondParameter.put("interfaceId", second.get("interfaceId"));
        create("api-parameters", secondParameter);

        mvc.perform(get(path("columns")).param("id", String.valueOf(firstColumn.get("columnId"))))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.objectId").value(firstObject.get("objectId")));
        mvc.perform(get(path("api-parameters")).param("id", "first-parameter"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.interfaceId").value(first.get("interfaceId")));

        var conflictingMove = new java.util.HashMap<>(firstObject);
        conflictingMove.put("interfaceId", third.get("interfaceId"));
        mvc.perform(put(path("objects")).param("id", String.valueOf(firstObject.get("objectId")))
                .contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(conflictingMove)))
                .andExpect(status().isConflict());
        mvc.perform(get(path("objects")).param("id", String.valueOf(firstObject.get("objectId"))))
                .andExpect(jsonPath("$.interfaceId").value(first.get("interfaceId")));

        var movedInterface = new java.util.HashMap<>(third);
        movedInterface.put("sourceName", "other-source");
        mvc.perform(put(path("interfaces")).param("id", String.valueOf(third.get("interfaceId")))
                .contentType(MediaType.APPLICATION_JSON).content(json.writeValueAsBytes(movedInterface)))
                .andExpect(status().isOk());
        mvc.perform(get(path("objects")).param("id", String.valueOf(fourthObject.get("objectId"))))
                .andExpect(jsonPath("$.interfaceId").value(third.get("interfaceId")));
        mvc.perform(get(path("api-parameters")).param("id", String.valueOf(savedParameter.get("parameterId"))))
                .andExpect(jsonPath("$.interfaceId").value(first.get("interfaceId")));

        String audit = mvc.perform(get("/api/audit")).andReturn().getResponse().getContentAsString();
        assertTrue(audit.contains(String.valueOf(first.get("interfaceId"))));
        assertTrue(audit.contains(String.valueOf(second.get("interfaceId"))));
        assertFalse(first.get("interfaceId").equals(second.get("interfaceId")));
    }

    private Map<String, Object> object(String name, Map<String, Object> sourceInterface) {
        var value = new java.util.HashMap<>(OBJECT);
        value.put("objectName", name);
        value.put("interfaceId", sourceInterface.get("interfaceId"));
        return value;
    }

    private Map<String, Object> column(String name, Map<String, Object> sourceObject) {
        var value = new java.util.HashMap<>(COLUMN);
        value.put("columnName", name);
        value.put("objectId", sourceObject.get("objectId"));
        return value;
    }

    @Test
    void allowsNullAuthenticationCredentialsAndApiParameterValues() throws Exception {
        create("source-systems", SOURCE);
        var authentication = new java.util.HashMap<>(AUTH);
        authentication.put("authenticationId", "auth-without-credentials");
        authentication.put("authenticationUser", null);
        authentication.put("authenticationCredential", null);
        create("authentications", authentication);
        var sourceInterface = new java.util.HashMap<>(INTERFACE);
        sourceInterface.put("authenticationId", "auth-without-credentials");
        Map<String, Object> savedInterface = create("interfaces", sourceInterface);
        var parameter = new java.util.HashMap<>(API_PARAMETER);
        parameter.put("parameterValue", null);
        parameter.put("interfaceId", savedInterface.get("interfaceId"));
        mvc.perform(post(path("api-parameters")).contentType(MediaType.APPLICATION_JSON)
                .content(json.writeValueAsBytes(parameter)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.interfaceId").value(savedInterface.get("interfaceId")))
                .andExpect(jsonPath("$.parameterValue").isEmpty());
        mvc.perform(get(path("authentications/auth-without-credentials")))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.authenticationUser").isEmpty())
                .andExpect(jsonPath("$.authenticationCredential").isEmpty());
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
                "authenticationUser", "x".repeat(255), "authenticationCredential", "x".repeat(255),
                "isActive", true));
        mvc.perform(get(path("source-systems"))).andExpect(jsonPath("$[0].sourceName").value("source"))
                .andExpect(jsonPath("$[1].sourceName").value("z".repeat(255)));
    }

    @Test
    void liquibaseOwnsSchemaAndMigrationHistory() {
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM databasechangelog WHERE id = '001-ingestion-metadata'", Integer.class));
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM databasechangelog WHERE id = '003-authentication-credentials-and-api-parameters'", Integer.class));
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM databasechangelog WHERE id = '004-application-logging-and-metadata-audit'", Integer.class));
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM databasechangelog WHERE id = '005-scoped-metadata-names'", Integer.class));
        assertEquals(0, jdbc.queryForObject("SELECT COUNT(*) FROM source_column", Integer.class));
        assertEquals(0, jdbc.queryForObject("SELECT COUNT(*) FROM api_parameter", Integer.class));
    }

    @Test
    void auditsSuccessfulChangesAndRedactsSensitiveValues() throws Exception {
        create("source-systems", SOURCE);
        mvc.perform(post(path("source-systems")).contentType(MediaType.APPLICATION_JSON)
                .content(json.writeValueAsBytes(SOURCE))).andExpect(status().isConflict());
        var updated = new java.util.HashMap<>(SOURCE);
        updated.put("sourceFunctionalName", "Updated source");
        mvc.perform(put(path("source-systems/source")).contentType(MediaType.APPLICATION_JSON)
                .content(json.writeValueAsBytes(updated))).andExpect(status().isOk());
        mvc.perform(delete(path("source-systems/source"))).andExpect(status().isNoContent());

        var auth = new java.util.HashMap<>(AUTH);
        auth.put("authenticationUser", "audit-user-secret");
        auth.put("authenticationCredential", "audit-credential-secret");
        create("source-systems", SOURCE);
        create("authentications", auth);
        Map<String, Object> savedInterface = create("interfaces", INTERFACE);
        var parameter = new java.util.HashMap<>(API_PARAMETER);
        parameter.put("parameterValue", "audit-parameter-secret");
        parameter.put("interfaceId", savedInterface.get("interfaceId"));
        create("api-parameters", parameter);

        String audit = mvc.perform(get("/api/audit"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0].operation").value("CREATE"))
                .andExpect(jsonPath("$[0].entityType").value("ApiParameter"))
                .andExpect(jsonPath("$[0].recordId").value("parameter"))
                .andExpect(jsonPath("$[0].details").value(org.hamcrest.Matchers.containsString("[REDACTED]")))
                .andExpect(jsonPath("$[5].details").value(org.hamcrest.Matchers.containsString("\"before\":\"Source\"")))
                .andExpect(jsonPath("$[5].details").value(org.hamcrest.Matchers.containsString("\"after\":\"Updated source\"")))
                .andExpect(jsonPath("$[4].operation").value("DELETE"))
                .andExpect(jsonPath("$[5].operation").value("UPDATE"))
                .andExpect(jsonPath("$[6].operation").value("CREATE"))
                .andReturn().getResponse().getContentAsString();
        assertFalse(audit.contains("audit-user-secret"));
        assertFalse(audit.contains("audit-credential-secret"));
        assertFalse(audit.contains("audit-parameter-secret"));

        var logMessages = jdbc.queryForList("SELECT message FROM application_log", String.class);
        assertTrue(logMessages.stream().anyMatch(message -> message.equals("Application started")));
        assertTrue(logMessages.stream().anyMatch(message -> message.equals("Metadata SourceSystem created")));
        assertTrue(logMessages.stream().noneMatch(message -> message.contains("audit-")));
        assertEquals(0, jdbc.queryForObject(
                "SELECT COUNT(*) FROM metadata_audit WHERE operation NOT IN ('CREATE', 'UPDATE', 'DELETE')",
                Integer.class));
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
    void migrationCanBeReappliedInAnIsolatedDatabase() throws Exception {
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
            }
        }
    }

    @Test
    void migratesExistingHierarchyAndAuditHistoryWithoutLosingReferences() throws Exception {
        try (var connection = DriverManager.getConnection(
                "jdbc:h2:mem:scoped-migration;MODE=MariaDB;DB_CLOSE_DELAY=-1", "sa", "");
             var resources = new ClassLoaderResourceAccessor();
             var oldSchema = new Liquibase("db/changelog/pre-scoped-metadata.xml", resources,
                     new JdbcConnection(connection))) {
            oldSchema.update(new Contexts(), new LabelExpression());
            try (var statement = connection.createStatement()) {
                statement.executeUpdate("INSERT INTO source_system (source_name, source_functional_name, source_type, source_owner, source_application, source_location, is_active) VALUES ('source', 'Source', 'database', NULL, 'CRM', NULL, TRUE)");
                statement.executeUpdate("INSERT INTO source_system_interface (interface_name, source_name, authentication_id, interface_functional_name, interface_type, interface_location, decimal_separator, thousands_separator, delimiter, has_headers, is_active) VALUES ('interface', 'source', NULL, 'Interface', 'jdbc', 'jdbc:source', '.', ',', NULL, TRUE, TRUE)");
                statement.executeUpdate("INSERT INTO source_object (object_name, interface_name, object_functional_name, object_description, is_confidential, is_business_critical, is_active) VALUES ('object', 'interface', 'Object', NULL, FALSE, TRUE, TRUE)");
                statement.executeUpdate("INSERT INTO source_column (column_name, object_name, column_functional_name, datatype, column_alternate_name, column_description, column_ordinal_position, needs_encryption, is_partition_field, partition_ordinal_position, is_key, key_ordinal_position, is_nullable, test_datatype, character_maximum_length, numeric_scale, numeric_precision, null_value_regex_pattern, is_active) VALUES ('column', 'object', 'Column', 'varchar', NULL, NULL, 1, FALSE, FALSE, 0, TRUE, 1, FALSE, TRUE, 255, NULL, NULL, NULL, TRUE)");
                statement.executeUpdate("INSERT INTO api_parameter (parameter_id, interface_id, parameter_name, parameter_value) VALUES ('parameter', 'interface', 'api-version', 'v1')");
                statement.executeUpdate("INSERT INTO metadata_audit (occurred_at, operation, entity_type, record_id, details) VALUES (CURRENT_TIMESTAMP, 'CREATE', 'SourceObject', 'object', '{\"objectName\":\"object\"}')");
            }
        }

        try (var connection = DriverManager.getConnection(
                "jdbc:h2:mem:scoped-migration;MODE=MariaDB;DB_CLOSE_DELAY=-1", "sa", "");
             var resources = new ClassLoaderResourceAccessor();
             var liquibase = new Liquibase("db/changelog/db.changelog-master.xml", resources,
                     new JdbcConnection(connection))) {
            liquibase.update(new Contexts(), new LabelExpression());
            assertEquals(1, jdbcCount(connection, "SELECT COUNT(*) FROM source_system_interface WHERE interface_id = 'interface' AND interface_name = 'interface'"), "interface migration");
            assertEquals(1, jdbcCount(connection, "SELECT COUNT(*) FROM source_object WHERE object_id = 'object' AND interface_id = 'interface' AND object_name = 'object'"), "object migration");
            assertEquals(1, jdbcCount(connection, "SELECT COUNT(*) FROM source_column WHERE column_id = 'column' AND object_id = 'object' AND column_name = 'column'"), "column migration");
            assertEquals(1, jdbcCount(connection, "SELECT COUNT(*) FROM api_parameter WHERE parameter_id = 'parameter' AND interface_id = 'interface'"), "API parameter migration");
            assertEquals(1, jdbcCount(connection, "SELECT COUNT(*) FROM metadata_audit WHERE record_id = 'object'"), "audit retention");
            liquibase.update(new Contexts(), new LabelExpression());
            assertEquals(1, jdbcCount(connection, "SELECT COUNT(*) FROM source_column"));
            assertEquals(1, jdbcCount(connection, "SELECT COUNT(*) FROM api_parameter"));
        }
    }

    private int jdbcCount(java.sql.Connection connection, String query) throws Exception {
        try (var statement = connection.createStatement(); var rows = statement.executeQuery(query)) {
            rows.next();
            return rows.getInt(1);
        }
    }
}
