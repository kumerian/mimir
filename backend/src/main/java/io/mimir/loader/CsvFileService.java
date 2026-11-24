package io.mimir.loader;

import java.io.IOException;
import java.io.Reader;
import java.io.UncheckedIOException;
import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.Iterator;
import java.util.List;
import java.util.Locale;
import java.util.Set;
import java.util.function.Consumer;

import io.mimir.metadata.Metadata.SourceSystemInterface;
import org.apache.commons.csv.CSVFormat;
import org.apache.commons.csv.CSVParser;
import org.apache.commons.csv.CSVRecord;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;

@Service
public class CsvFileService {
    public CsvSchema analyze(SourceSystemInterface sourceInterface) {
        try (Reader reader = reader(sourceInterface); CSVParser parser = format(sourceInterface).parse(reader)) {
            Iterator<CSVRecord> records = parser.iterator();
            List<String> headers = readHeaders(records);
            CsvTypeInference inference = new CsvTypeInference(sourceInterface, headers.size());
            long rowCount = 0;
            while (records.hasNext()) {
                List<String> row = row(records.next(), headers.size(), rowCount + 2);
                inference.accept(row);
                rowCount++;
            }
            return new CsvSchema(headers, inference.finish(), rowCount);
        } catch (LoaderException exception) {
            throw exception;
        } catch (IOException | UncheckedIOException exception) {
            throw new LoaderException(HttpStatus.BAD_REQUEST, "The CSV file could not be read or parsed.", exception);
        }
    }

    public List<String> readHeaders(SourceSystemInterface sourceInterface) {
        try (Reader reader = reader(sourceInterface); CSVParser parser = format(sourceInterface).parse(reader)) {
            return readHeaders(parser.iterator());
        } catch (LoaderException exception) {
            throw exception;
        } catch (IOException | UncheckedIOException exception) {
            throw new LoaderException(HttpStatus.BAD_REQUEST, "The CSV file could not be read or parsed.", exception);
        }
    }

    public long forEachRow(
            SourceSystemInterface sourceInterface,
            List<String> expectedHeaders,
            Consumer<List<String>> consumer) {
        try (Reader reader = reader(sourceInterface); CSVParser parser = format(sourceInterface).parse(reader)) {
            Iterator<CSVRecord> records = parser.iterator();
            List<String> headers = readHeaders(records);
            if (!headers.equals(expectedHeaders)) {
                throw new LoaderException(HttpStatus.CONFLICT, "The CSV header changed after metadata was created.");
            }
            long rowCount = 0;
            while (records.hasNext()) {
                consumer.accept(row(records.next(), headers.size(), rowCount + 2));
                rowCount++;
            }
            return rowCount;
        } catch (LoaderException exception) {
            throw exception;
        } catch (IOException | UncheckedIOException exception) {
            throw new LoaderException(HttpStatus.BAD_REQUEST, "The CSV file could not be read or parsed.", exception);
        }
    }

    private Reader reader(SourceSystemInterface sourceInterface) throws IOException {
        validateInterface(sourceInterface);
        Path path;
        try {
            path = sourceInterface.interfaceLocation().startsWith("file:")
                    ? Paths.get(URI.create(sourceInterface.interfaceLocation()))
                    : Path.of(sourceInterface.interfaceLocation());
        } catch (IllegalArgumentException exception) {
            throw new LoaderException(HttpStatus.BAD_REQUEST, "The interface location is not a valid file path.", exception);
        }
        if (!Files.isRegularFile(path) || !Files.isReadable(path)) {
            throw new LoaderException(HttpStatus.BAD_REQUEST, "The CSV file does not exist or cannot be read.");
        }
        return Files.newBufferedReader(path, StandardCharsets.UTF_8);
    }

    private CSVFormat format(SourceSystemInterface sourceInterface) {
        validateInterface(sourceInterface);
        String configuredDelimiter = sourceInterface.delimiter();
        char delimiter = configuredDelimiter == null || configuredDelimiter.isBlank()
                ? ','
                : configuredDelimiter.charAt(0);
        if (delimiter == '"' || delimiter == '\r' || delimiter == '\n') {
            throw new LoaderException(HttpStatus.BAD_REQUEST, "The configured CSV delimiter is invalid.");
        }
        return CSVFormat.DEFAULT.builder()
                .setDelimiter(delimiter)
                .setIgnoreEmptyLines(true)
                .get();
    }

    private void validateInterface(SourceSystemInterface sourceInterface) {
        String type = sourceInterface.interfaceType().replaceAll("[\\s_-]", "").toLowerCase(Locale.ROOT);
        if (!type.equals("csv") && !type.equals("flatfile")) {
            throw new LoaderException(HttpStatus.BAD_REQUEST, "Only CSV flatfile interfaces are supported.");
        }
        if (Boolean.FALSE.equals(sourceInterface.hasHeaders())) {
            throw new LoaderException(HttpStatus.BAD_REQUEST, "CSV files without headers are not supported.");
        }
    }

    private List<String> readHeaders(Iterator<CSVRecord> records) {
        if (!records.hasNext()) {
            throw new LoaderException(HttpStatus.BAD_REQUEST, "The CSV file is empty and has no header.");
        }
        CSVRecord record = records.next();
        List<String> headers = new ArrayList<>(record.size());
        Set<String> unique = new HashSet<>();
        for (int index = 0; index < record.size(); index++) {
            String header = record.get(index);
            if (index == 0 && header.startsWith("\uFEFF")) {
                header = header.substring(1);
            }
            header = header.strip();
            if (header.isEmpty() || header.length() > 255 || !unique.add(header.toLowerCase(Locale.ROOT))) {
                throw new LoaderException(
                        HttpStatus.BAD_REQUEST,
                        "CSV headers must be nonblank, unique, and no longer than 255 characters.");
            }
            headers.add(header);
        }
        return List.copyOf(headers);
    }

    private List<String> row(CSVRecord record, int columnCount, long rowNumber) {
        if (record.size() != columnCount) {
            throw new LoaderException(
                    HttpStatus.BAD_REQUEST,
                    "CSV row " + rowNumber + " has " + record.size() + " fields; expected " + columnCount + ".");
        }
        List<String> values = new ArrayList<>(columnCount);
        for (int index = 0; index < columnCount; index++) {
            values.add(record.get(index));
        }
        return List.copyOf(values);
    }

    public record CsvSchema(List<String> headers, List<CsvTypeInference.InferredColumn> columns, long rowCount) {}
}
