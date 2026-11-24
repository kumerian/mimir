package io.mimir.metadata;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;
import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Column;
import org.springframework.data.relational.core.mapping.Table;

public final class Metadata {
    private Metadata() {}

    public interface Identified {
        String id();
    }

    @Table("source_system")
    public record SourceSystem(
            @Id @Column("source_name") @NotBlank @Size(max = 255) String sourceName,
            @Column("source_functional_name") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String sourceFunctionalName,
            @Column("source_type") @NotBlank @Size(max = 255) String sourceType,
            @Column("source_owner") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String sourceOwner,
            @Column("source_application") @NotBlank @Size(max = 255) String sourceApplication,
            @Column("source_location") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String sourceLocation,
            @Column("is_active") @NotNull Boolean isActive) implements Identified {
        public String id() { return sourceName; }
    }

    @Table("authentication")
    public record Authentication(
            @Id @Column("authentication_id") @NotBlank @Size(max = 255) String authenticationId,
            @Column("authentication_type") @NotBlank @Size(max = 255) String authenticationType,
            @Column("authentication_location") @NotBlank @Size(max = 2048) String authenticationLocation,
            @Column("authentication_user") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String authenticationUser,
            @Column("authentication_credential") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String authenticationCredential,
            @Column("is_active") @NotNull Boolean isActive) implements Identified {
        public String id() { return authenticationId; }
    }

    @Table("api_parameter")
    public record ApiParameter(
            @Id @Column("parameter_id") @NotBlank @Size(max = 255) String parameterId,
            @Column("interface_id") @NotBlank @Size(max = 255) String interfaceId,
            @Column("parameter_name") @NotBlank @Size(max = 255) String parameterName,
            @Column("parameter_value") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String parameterValue) implements Identified {
        public String id() { return parameterId; }
    }

    @Table("source_system_interface")
    public record SourceSystemInterface(
            @Id @Column("interface_name") @NotBlank @Size(max = 255) String interfaceName,
            @Column("source_name") @NotBlank @Size(max = 255) String sourceName,
            @Column("authentication_id") @Size(max = 255) @Pattern(regexp = "(?s).*\\S.*") String authenticationId,
            @Column("interface_functional_name") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String interfaceFunctionalName,
            @Column("interface_type") @NotBlank @Size(max = 255) String interfaceType,
            @Column("interface_location") @NotBlank @Size(max = 2048) String interfaceLocation,
            @Column("decimal_separator") @NotBlank @Size(min = 1, max = 1) String decimalSeparator,
            @Column("thousands_separator") @NotBlank @Size(min = 1, max = 1) String thousandsSeparator,
            @Column("delimiter") @Size(min = 1, max = 1) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String delimiter,
            @Column("has_headers") Boolean hasHeaders,
            @Column("is_active") @NotNull Boolean isActive) implements Identified {
        public String id() { return interfaceName; }
    }

    @Table("source_object")
    public record SourceObject(
            @Id @Column("object_name") @NotBlank @Size(max = 255) String objectName,
            @Column("interface_name") @NotBlank @Size(max = 255) String interfaceName,
            @Column("object_functional_name") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String objectFunctionalName,
            @Column("object_description") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String objectDescription,
            @Column("is_confidential") @NotNull Boolean isConfidential,
            @Column("is_business_critical") @NotNull Boolean isBusinessCritical,
            @Column("is_active") @NotNull Boolean isActive) implements Identified {
        public String id() { return objectName; }
    }

    @Table("source_column")
    public record SourceColumn(
            @Id @Column("column_name") @NotBlank @Size(max = 255) String columnName,
            @Column("object_name") @NotBlank @Size(max = 255) String objectName,
            @Column("column_alternate_name") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String columnAlternateName,
            @Column("column_functional_name") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String columnFunctionalName,
            @Column("column_description") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String columnDescription,
            @Column("column_ordinal_position") @NotNull Integer columnOrdinalPosition,
            @Column("needs_encryption") @NotNull Boolean needsEncryption,
            @Column("is_partition_field") @NotNull Boolean isPartitionField,
            @Column("partition_ordinal_position") @NotNull Integer partitionOrdinalPosition,
            @Column("is_key") @NotNull Boolean isKey,
            @Column("key_ordinal_position") @NotNull Integer keyOrdinalPosition,
            @Column("is_nullable") @NotNull Boolean isNullable,
            @Column("test_datatype") @NotNull Boolean testDatatype,
            @Column("datatype") @NotBlank @Size(max = 255) String datatype,
            @Column("character_maximum_length") Integer characterMaximumLength,
            @Column("numeric_scale") Integer numericScale,
            @Column("numeric_precision") Integer numericPrecision,
            @Column("null_value_regex_pattern") @Size(max = 255) @Pattern(regexp = "(?s)(?:.*\\S.*)?") String nullValueRegexPattern,
            @Column("is_active") @NotNull Boolean isActive) implements Identified {
        public String id() { return columnName; }
    }
}
