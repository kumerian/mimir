package io.mimir.loader;

import java.math.BigDecimal;
import java.sql.Connection;
import java.sql.DatabaseMetaData;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Types;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.format.DateTimeParseException;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;
import java.util.UUID;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import io.mimir.metadata.Metadata.SourceColumn;
import io.mimir.metadata.Metadata.SourceObject;
import io.mimir.metadata.Metadata.SourceSystemInterface;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.jdbc.core.ConnectionCallback;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.core.ParameterizedPreparedStatementSetter;
import org.springframework.stereotype.Service;
import org.springframework.transaction.PlatformTransactionManager;
import org.springframework.transaction.support.TransactionTemplate;

@Service
public class LoadedDataWriter {
    private static final int BATCH_SIZE = 500;
    private static final Pattern VARCHAR = Pattern.compile("(?i)^VARCHAR\\((\\d+)\\)$");
    private static final Pattern DECIMAL = Pattern.compile("(?i)^DECIMAL\\((\\d+),(\\d+)\\)$");

    private final JdbcTemplate jdbc;
    private final CsvFileService csvFiles;
    private final TransactionTemplate transaction;

    public LoadedDataWriter(
            @Qualifier("loadedDataJdbcTemplate") JdbcTemplate jdbc,
            CsvFileService csvFiles,
            @Qualifier("loadedDataTransactionManager") PlatformTransactionManager transactionManager) {
        this.jdbc = jdbc;
        this.csvFiles = csvFiles;
        this.transaction = new TransactionTemplate(transactionManager);
    }

    public long load(SourceSystemInterface sourceInterface, SourceObject sourceObject, List<SourceColumn> columns) {
        List<ColumnDefinition> definitions = columns.stream().map(LoadedDataWriter::definition).toList();
        String tableName = FlatfileLoaderService.tableName(sourceObject.objectId());
        ensureTable(tableName, definitions);
        String insert = "INSERT INTO " + quote(tableName) + " ("
                + String.join(", ", columns.stream().map(column -> quote(column.columnName())).toList())
                + ") VALUES (" + String.join(", ", columns.stream().map(column -> "?").toList()) + ")";

        Long rows = transaction.execute(status -> {
            long[] inserted = {0};
            List<List<String>> batch = new ArrayList<>(BATCH_SIZE);
            try {
                long rowCount = csvFiles.forEachRow(sourceInterface, columns.stream()
                        .map(SourceColumn::columnName)
                        .toList(), row -> {
                            batch.add(row);
                            if (batch.size() == BATCH_SIZE) {
                                insertBatch(insert, batch, definitions, sourceInterface, inserted[0] + 2);
                                inserted[0] += batch.size();
                                batch.clear();
                            }
                        });
                if (!batch.isEmpty()) {
                    insertBatch(insert, batch, definitions, sourceInterface, inserted[0] + 2);
                    inserted[0] += batch.size();
                }
                return rowCount;
            } catch (LoaderException exception) {
                throw new LoaderException(
                        exception.status(),
                        exception.getMessage()
                                + " Data rows were rolled back; the destination table may already exist or have been created.",
                        exception);
            }
        });
        return rows == null ? 0 : rows;
    }

    public String tableName(String objectId) {
        return FlatfileLoaderService.tableName(objectId);
    }

    private void ensureTable(String tableName, List<ColumnDefinition> definitions) {
        List<TableColumn> existing = columns(tableName);
        if (existing.isEmpty()) {
            String columns = String.join(", ", definitions.stream()
                    .map(column -> quote(column.name()) + " " + column.type().sql()
                            + (column.nullable() ? " NULL" : " NOT NULL"))
                    .toList());
            jdbc.execute("CREATE TABLE IF NOT EXISTS " + quote(tableName) + " (" + columns + ")");
            existing = columns(tableName);
        }
        if (existing.size() != definitions.size()) {
            throw schemaMismatch();
        }
        for (int index = 0; index < definitions.size(); index++) {
            if (!definitions.get(index).matches(existing.get(index))) {
                throw schemaMismatch();
            }
        }
    }

    private List<TableColumn> columns(String tableName) {
        return jdbc.execute((ConnectionCallback<List<TableColumn>>) connection -> {
            List<TableColumn> result = new ArrayList<>();
            DatabaseMetaData metadata = connection.getMetaData();
            try (ResultSet rows = metadata.getColumns(connection.getCatalog(), null, tableName, null)) {
                while (rows.next()) {
                    if (tableName.equalsIgnoreCase(rows.getString("TABLE_NAME"))) {
                        result.add(new TableColumn(
                                rows.getString("COLUMN_NAME"),
                                rows.getString("TYPE_NAME"),
                                rows.getInt("COLUMN_SIZE"),
                                rows.getInt("DECIMAL_DIGITS"),
                                rows.getInt("ORDINAL_POSITION"),
                                rows.getInt("NULLABLE") == DatabaseMetaData.columnNullable));
                    }
                }
            }
            return result.stream().sorted(java.util.Comparator.comparingInt(TableColumn::ordinal)).toList();
        });
    }

    private void insertBatch(
            String insert,
            List<List<String>> batch,
            List<ColumnDefinition> definitions,
            SourceSystemInterface sourceInterface,
            long firstRowNumber) {
        int[] batchIndex = {0};
        jdbc.batchUpdate(
                insert,
                batch,
                batch.size(),
                (ParameterizedPreparedStatementSetter<List<String>>) (statement, row) -> {
                    long rowNumber = firstRowNumber + batchIndex[0]++;
                    for (int index = 0; index < row.size(); index++) {
                        setValue(statement, index + 1, row.get(index), definitions.get(index), sourceInterface, rowNumber);
                    }
                });
    }

    private void setValue(
            PreparedStatement statement,
            int parameter,
            String value,
            ColumnDefinition column,
            SourceSystemInterface sourceInterface,
            long rowNumber) throws SQLException {
        if (value.isEmpty()) {
            if (!column.nullable()) {
                throw invalidValue(rowNumber, column.name());
            }
            statement.setNull(parameter, column.type().jdbcType());
            return;
        }
        try {
            switch (column.type().family()) {
                case "VARCHAR", "TEXT" -> {
                    if (column.type().family().equals("VARCHAR")
                            && value.codePointCount(0, value.length()) > column.type().size()) {
                        throw invalidValue(rowNumber, column.name());
                    }
                    statement.setString(parameter, value);
                }
                case "BIGINT" -> {
                    String normalized = CsvTypeInference.normalizeNumber(value, sourceInterface);
                    if (normalized == null) throw invalidValue(rowNumber, column.name());
                    statement.setLong(parameter, Long.parseLong(normalized));
                }
                case "DECIMAL" -> {
                    String normalized = CsvTypeInference.normalizeNumber(value, sourceInterface);
                    if (normalized == null) throw invalidValue(rowNumber, column.name());
                    statement.setBigDecimal(parameter, new BigDecimal(normalized));
                }
                case "DATE" -> statement.setDate(parameter, java.sql.Date.valueOf(LocalDate.parse(value.strip())));
                case "DATETIME" -> statement.setTimestamp(
                        parameter, java.sql.Timestamp.valueOf(LocalDateTime.parse(value.strip())));
                case "BOOLEAN" -> {
                    if (!value.equalsIgnoreCase("true") && !value.equalsIgnoreCase("false")) {
                        throw invalidValue(rowNumber, column.name());
                    }
                    statement.setBoolean(parameter, Boolean.parseBoolean(value));
                }
                default -> throw new IllegalStateException("Unsupported loaded-data column type");
            }
        } catch (NumberFormatException | DateTimeParseException exception) {
            throw invalidValue(rowNumber, column.name());
        }
    }

    private static ColumnDefinition definition(SourceColumn column) {
        String datatype = column.datatype().strip();
        Matcher varchar = VARCHAR.matcher(datatype);
        if (varchar.matches()) {
            int size = Integer.parseInt(varchar.group(1));
            if (size < 1 || size > 16_383) throw unsupportedType();
            return new ColumnDefinition(column.columnName(), new SqlType("VARCHAR(" + size + ")", "VARCHAR", size, 0,
                    Types.VARCHAR), column.isNullable());
        }
        Matcher decimal = DECIMAL.matcher(datatype);
        if (decimal.matches()) {
            int precision = Integer.parseInt(decimal.group(1));
            int scale = Integer.parseInt(decimal.group(2));
            if (precision < 1 || precision > 65 || scale < 0 || scale > 30 || scale > precision) {
                throw unsupportedType();
            }
            return new ColumnDefinition(column.columnName(),
                    new SqlType("DECIMAL(" + precision + "," + scale + ")", "DECIMAL", precision, scale, Types.DECIMAL),
                    column.isNullable());
        }
        return switch (datatype.toUpperCase(Locale.ROOT)) {
            case "TEXT", "MEDIUMTEXT" -> new ColumnDefinition(column.columnName(),
                    new SqlType(datatype.toUpperCase(Locale.ROOT), "TEXT", 0, 0, Types.LONGVARCHAR),
                    column.isNullable());
            case "BIGINT" -> new ColumnDefinition(column.columnName(),
                    new SqlType("BIGINT", "BIGINT", 0, 0, Types.BIGINT), column.isNullable());
            case "DATE" -> new ColumnDefinition(column.columnName(),
                    new SqlType("DATE", "DATE", 0, 0, Types.DATE), column.isNullable());
            case "DATETIME" -> new ColumnDefinition(column.columnName(),
                    new SqlType("DATETIME", "DATETIME", 0, 0, Types.TIMESTAMP), column.isNullable());
            case "BOOLEAN" -> new ColumnDefinition(column.columnName(),
                    new SqlType("BOOLEAN", "BOOLEAN", 0, 0, Types.BOOLEAN), column.isNullable());
            default -> throw unsupportedType();
        };
    }

    private static boolean compatible(SqlType expected, TableColumn actual) {
        String typeName = actual.typeName().toUpperCase(Locale.ROOT);
        return switch (expected.family()) {
            case "VARCHAR" -> typeName.contains("CHAR") && actual.size() == expected.size();
            case "TEXT" -> typeName.contains("TEXT") || typeName.contains("CLOB") || typeName.contains("LONGVARCHAR");
            case "BIGINT" -> typeName.contains("BIGINT");
            case "DECIMAL" -> (typeName.contains("DECIMAL") || typeName.contains("NUMERIC"))
                    && actual.size() == expected.size() && actual.scale() == expected.scale();
            case "DATE" -> typeName.equals("DATE");
            case "DATETIME" -> typeName.contains("DATETIME") || typeName.contains("TIMESTAMP");
            case "BOOLEAN" -> typeName.contains("BOOL") || typeName.equals("BIT") || typeName.equals("TINYINT");
            default -> false;
        };
    }

    private static LoaderException schemaMismatch() {
        return new LoaderException(
                org.springframework.http.HttpStatus.CONFLICT,
                "The loaded-data table schema differs from metadata. Schema drift is not supported.");
    }

    private static LoaderException unsupportedType() {
        return new LoaderException(
                org.springframework.http.HttpStatus.BAD_REQUEST,
                "A Column has an unsupported datatype for the CSV loader.");
    }

    private static LoaderException invalidValue(long rowNumber, String columnName) {
        return new LoaderException(
                org.springframework.http.HttpStatus.BAD_REQUEST,
                "CSV row " + rowNumber + " contains a value incompatible with Column '" + columnName + "'.");
    }

    private static String quote(String identifier) {
        return "`" + identifier.replace("`", "``") + "`";
    }

    private record SqlType(String sql, String family, int size, int scale, int jdbcType) {}

    private record ColumnDefinition(String name, SqlType type, boolean nullable) {
        boolean matches(TableColumn actual) {
            return name.equalsIgnoreCase(actual.name())
                    && compatible(type, actual)
                    && nullable == actual.nullable();
        }
    }

    private record TableColumn(String name, String typeName, int size, int scale, int ordinal, boolean nullable) {}
}
