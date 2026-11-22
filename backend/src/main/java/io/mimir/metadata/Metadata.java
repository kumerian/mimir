package io.mimir.metadata;

import jakarta.validation.constraints.NotBlank;
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
            @Column("source_functional_name") @NotBlank @Size(max = 255) String sourceFunctionalName,
            @Column("source_type") @NotBlank @Size(max = 255) String sourceType) implements Identified {
        public String id() { return sourceName; }
    }

    @Table("authentication")
    public record Authentication(
            @Id @Column("authentication_id") @NotBlank @Size(max = 255) String authenticationId,
            @Column("authentication_type") @NotBlank @Size(max = 255) String authenticationType,
            @Column("authentication_location") @NotBlank @Size(max = 2048) String authenticationLocation) implements Identified {
        public String id() { return authenticationId; }
    }

    @Table("source_system_interface")
    public record SourceSystemInterface(
            @Id @Column("interface_name") @NotBlank @Size(max = 255) String interfaceName,
            @Column("source_name") @NotBlank @Size(max = 255) String sourceName,
            @Column("authentication_id") @Size(max = 255) @Pattern(regexp = "(?s).*\\S.*") String authenticationId,
            @Column("interface_functional_name") @NotBlank @Size(max = 255) String interfaceFunctionalName,
            @Column("interface_type") @NotBlank @Size(max = 255) String interfaceType,
            @Column("interface_location") @NotBlank @Size(max = 2048) String interfaceLocation) implements Identified {
        public String id() { return interfaceName; }
    }

    @Table("source_object")
    public record SourceObject(
            @Id @Column("object_name") @NotBlank @Size(max = 255) String objectName,
            @Column("interface_name") @NotBlank @Size(max = 255) String interfaceName,
            @Column("object_functional_name") @NotBlank @Size(max = 255) String objectFunctionalName) implements Identified {
        public String id() { return objectName; }
    }

    @Table("source_column")
    public record SourceColumn(
            @Id @Column("column_name") @NotBlank @Size(max = 255) String columnName,
            @Column("object_name") @NotBlank @Size(max = 255) String objectName,
            @Column("column_functional_name") @NotBlank @Size(max = 255) String columnFunctionalName,
            @Column("datatype") @NotBlank @Size(max = 255) String datatype) implements Identified {
        public String id() { return columnName; }
    }
}
