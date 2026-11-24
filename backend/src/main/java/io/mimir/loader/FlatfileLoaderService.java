package io.mimir.loader;

import static io.mimir.metadata.Metadata.*;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;

import io.mimir.logging.LoggingService;
import io.mimir.metadata.MetadataStore;
import org.springframework.dao.DataAccessException;
import org.springframework.data.relational.core.conversion.DbActionExecutionException;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.transaction.PlatformTransactionManager;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.web.server.ResponseStatusException;

@Service
public class FlatfileLoaderService {
    private final MetadataStore metadata;
    private final CsvFileService csvFiles;
    private final LoadedDataWriter dataWriter;
    private final LoggingService logging;
    private final TransactionTemplate metadataTransaction;
    private final ConcurrentMap<String, Boolean> activeLoads = new ConcurrentHashMap<>();

    public FlatfileLoaderService(
            MetadataStore metadata,
            CsvFileService csvFiles,
            LoadedDataWriter dataWriter,
            LoggingService logging,
            @Qualifier("transactionManager") PlatformTransactionManager metadataTransactionManager) {
        this.metadata = metadata;
        this.csvFiles = csvFiles;
        this.dataWriter = dataWriter;
        this.logging = logging;
        this.metadataTransaction = new TransactionTemplate(metadataTransactionManager);
    }

    public MetadataResult deriveMetadata(String interfaceId) {
        SourceSystemInterface sourceInterface = getInterface(interfaceId);
        CsvFileService.CsvSchema schema = csvFiles.analyze(sourceInterface);
        return Objects.requireNonNull(metadataTransaction.execute(
                status -> persistMetadata(interfaceId, sourceInterface, schema)));
    }

    private MetadataResult persistMetadata(
            String interfaceId,
            SourceSystemInterface sourceInterface,
            CsvFileService.CsvSchema schema) {
        List<SourceObject> interfaceObjects = metadata.list(SourceObject.class).stream()
                .filter(object -> object.interfaceId().equals(interfaceId))
                .toList();

        SourceObject sourceObject = interfaceObjects.stream()
                .filter(object -> object.objectName().equals(sourceInterface.interfaceName()))
                .findFirst()
                .orElse(null);
        if (sourceObject == null && interfaceObjects.size() > 1) {
            throw new LoaderException(
                    HttpStatus.CONFLICT,
                    "This interface has multiple Objects; select or remove the extra Object before deriving CSV metadata.");
        }
        boolean updated = !interfaceObjects.isEmpty();
        if (sourceObject == null && interfaceObjects.size() == 1) {
            SourceObject existing = interfaceObjects.get(0);
            sourceObject = metadata.update(
                    existing.objectId(),
                    new SourceObject(
                            existing.objectId(),
                            sourceInterface.interfaceName(),
                            interfaceId,
                            existing.objectFunctionalName(),
                            existing.objectDescription(),
                            existing.isConfidential(),
                            existing.isBusinessCritical(),
                            existing.isActive()),
                    SourceObject.class);
        } else if (sourceObject == null) {
            sourceObject = metadata.create(
                    new SourceObject(
                            null,
                            sourceInterface.interfaceName(),
                            interfaceId,
                            null,
                            null,
                            false,
                            false,
                            true),
                    SourceObject.class);
        }

        SourceObject targetObject = sourceObject;
        List<SourceColumn> currentColumns = metadata.list(SourceColumn.class).stream()
                .filter(column -> column.objectId().equals(targetObject.objectId()))
                .toList();
        Map<String, SourceColumn> byName = new HashMap<>();
        for (SourceColumn column : currentColumns) {
            String key = column.columnName().toLowerCase(Locale.ROOT);
            if (byName.putIfAbsent(key, column) != null) {
                throw new LoaderException(HttpStatus.CONFLICT, "Existing Columns have conflicting names.");
            }
        }

        List<SourceColumn> savedColumns = new ArrayList<>();
        for (int index = 0; index < schema.headers().size(); index++) {
            String name = schema.headers().get(index);
            String key = name.toLowerCase(Locale.ROOT);
            CsvTypeInference.InferredColumn inferred = schema.columns().get(index);
            SourceColumn current = byName.remove(key);
            SourceColumn value = current == null
                    ? new SourceColumn(
                            null,
                            name,
                            targetObject.objectId(),
                            null,
                            null,
                            null,
                            index + 1,
                            false,
                            false,
                            0,
                            false,
                            0,
                            true,
                            true,
                            inferred.datatype(),
                            inferred.characterMaximumLength(),
                            inferred.numericScale(),
                            inferred.numericPrecision(),
                            null,
                            true)
                    : new SourceColumn(
                            current.columnId(),
                            name,
                            targetObject.objectId(),
                            current.columnAlternateName(),
                            current.columnFunctionalName(),
                            current.columnDescription(),
                            index + 1,
                            current.needsEncryption(),
                            current.isPartitionField(),
                            current.partitionOrdinalPosition(),
                            current.isKey(),
                            current.keyOrdinalPosition(),
                            current.isNullable(),
                            true,
                            inferred.datatype(),
                            inferred.characterMaximumLength(),
                            inferred.numericScale(),
                            inferred.numericPrecision(),
                            current.nullValueRegexPattern(),
                            current.isActive());
            savedColumns.add(current == null
                    ? metadata.create(value, SourceColumn.class)
                    : metadata.update(current.columnId(), value, SourceColumn.class));
        }

        List<SourceColumn> removedColumns = new ArrayList<>(byName.values());
        removedColumns.forEach(column -> metadata.delete(column.columnId(), SourceColumn.class));
        logging.info(FlatfileLoaderService.class, "CSV metadata derivation completed");
        return new MetadataResult(targetObject, savedColumns, schema.rowCount(), removedColumns.size(), updated);
    }

    public LoadResult load(String interfaceId) {
        if (activeLoads.putIfAbsent(interfaceId, true) != null) {
            throw new LoaderException(HttpStatus.CONFLICT, "A data load is already running for this interface.");
        }
        try {
            return performLoad(interfaceId);
        } finally {
            activeLoads.remove(interfaceId);
        }
    }

    private LoadResult performLoad(String interfaceId) {
        SourceSystemInterface sourceInterface = getInterface(interfaceId);
        List<SourceObject> objects = metadata.list(SourceObject.class).stream()
                .filter(object -> object.interfaceId().equals(interfaceId)
                        && object.objectName().equals(sourceInterface.interfaceName()))
                .toList();
        if (objects.size() != 1) {
            throw new LoaderException(HttpStatus.CONFLICT, "Derive CSV metadata before loading this interface.");
        }
        SourceObject sourceObject = objects.get(0);
        List<SourceColumn> columns = metadata.list(SourceColumn.class).stream()
                .filter(column -> column.objectId().equals(sourceObject.objectId()))
                .sorted(Comparator.comparing(SourceColumn::columnOrdinalPosition))
                .toList();
        if (columns.isEmpty()) {
            throw new LoaderException(HttpStatus.CONFLICT, "Derive CSV metadata before loading this interface.");
        }

        List<String> headers = csvFiles.readHeaders(sourceInterface);
        List<String> columnNames = columns.stream().map(SourceColumn::columnName).toList();
        if (!headers.equals(columnNames)) {
            throw new LoaderException(
                    HttpStatus.CONFLICT,
                    "The CSV header does not match the saved metadata. Derive metadata again before loading.");
        }
        try {
            long loadedRows = dataWriter.load(sourceInterface, sourceObject, columns);
            logging.info(FlatfileLoaderService.class, "CSV data load completed");
            return new LoadResult(sourceObject.objectName(), dataWriter.tableName(sourceObject.objectId()), loadedRows);
        } catch (DataAccessException | DbActionExecutionException exception) {
            logging.error(FlatfileLoaderService.class, "CSV data load failed");
            throw new LoaderException(
                    HttpStatus.INTERNAL_SERVER_ERROR,
                    "The data load failed. Data rows were rolled back; the destination table may already exist or have been created.",
                    exception);
        }
    }

    private SourceSystemInterface getInterface(String interfaceId) {
        try {
            SourceSystemInterface sourceInterface = metadata.get(interfaceId, SourceSystemInterface.class);
            String type = sourceInterface.interfaceType().replaceAll("[\\s_-]", "").toLowerCase(Locale.ROOT);
            if (!type.equals("csv") && !type.equals("flatfile")) {
                throw new LoaderException(HttpStatus.BAD_REQUEST, "Only CSV flatfile interfaces are supported.");
            }
            return sourceInterface;
        } catch (ResponseStatusException exception) {
            throw new LoaderException(HttpStatus.NOT_FOUND, "The saved interface was not found.", exception);
        }
    }

    public static String tableName(String objectId) {
        UUID stableName = UUID.nameUUIDFromBytes(objectId.getBytes(StandardCharsets.UTF_8));
        return "object_" + stableName.toString().replace("-", "");
    }

    public record MetadataResult(
            SourceObject object,
            List<SourceColumn> columns,
            long rowsScanned,
            int removedColumns,
            boolean updated) {}

    public record LoadResult(String objectName, String tableName, long rowsLoaded) {}
}
