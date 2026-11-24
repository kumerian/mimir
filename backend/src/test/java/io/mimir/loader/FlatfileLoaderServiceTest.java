package io.mimir.loader;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;

import io.mimir.logging.LoggingService;
import io.mimir.metadata.Metadata.SourceColumn;
import io.mimir.metadata.Metadata.SourceObject;
import io.mimir.metadata.Metadata.SourceSystemInterface;
import io.mimir.metadata.MetadataStore;
import org.junit.jupiter.api.Test;
import org.springframework.http.HttpStatus;
import org.springframework.transaction.PlatformTransactionManager;

class FlatfileLoaderServiceTest {
    @Test
    void rejectsAnotherLoadForAnInterfaceWhileTheFirstLoadIsRunning() throws Exception {
        MetadataStore metadata = mock(MetadataStore.class);
        CsvFileService csvFiles = mock(CsvFileService.class);
        LoadedDataWriter dataWriter = mock(LoadedDataWriter.class);
        SourceSystemInterface sourceInterface = new SourceSystemInterface(
                "interface-id", "sample", "source", null, null, "csv", "/data/sample.csv",
                ".", ",", ";", true, true);
        SourceObject sourceObject = new SourceObject(
                "object-id", "sample", "interface-id", null, null, false, false, true);
        SourceColumn column = new SourceColumn(
                "column-id", "name", "object-id", null, null, null,
                1, false, false, 0, false, 0, true, true,
                "VARCHAR(20)", 20, null, null, null, true);
        when(metadata.get("interface-id", SourceSystemInterface.class)).thenReturn(sourceInterface);
        when(metadata.list(SourceObject.class)).thenReturn(List.of(sourceObject));
        when(metadata.list(SourceColumn.class)).thenReturn(List.of(column));
        when(csvFiles.readHeaders(sourceInterface)).thenReturn(List.of("name"));

        CountDownLatch started = new CountDownLatch(1);
        CountDownLatch finish = new CountDownLatch(1);
        doAnswer(invocation -> {
            started.countDown();
            if (!finish.await(5, TimeUnit.SECONDS)) {
                throw new IllegalStateException("Timed out waiting for the test load to finish");
            }
            return 1L;
        }).when(dataWriter).load(eq(sourceInterface), eq(sourceObject), any());
        when(dataWriter.tableName("object-id")).thenReturn("object-table");

        FlatfileLoaderService loader = new FlatfileLoaderService(
                metadata,
                csvFiles,
                dataWriter,
                mock(LoggingService.class),
                mock(PlatformTransactionManager.class));
        ExecutorService executor = Executors.newSingleThreadExecutor();
        try {
            var firstLoad = executor.submit(() -> loader.load("interface-id"));
            org.junit.jupiter.api.Assertions.assertTrue(started.await(5, TimeUnit.SECONDS));

            LoaderException duplicate = assertThrows(
                    LoaderException.class,
                    () -> loader.load("interface-id"));
            assertEquals(HttpStatus.CONFLICT, duplicate.status());

            finish.countDown();
            assertEquals(1, firstLoad.get(5, TimeUnit.SECONDS).rowsLoaded());
        } finally {
            finish.countDown();
            executor.shutdownNow();
        }
    }
}
