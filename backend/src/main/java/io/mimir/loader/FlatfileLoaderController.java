package io.mimir.loader;

import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/loader/interfaces")
public class FlatfileLoaderController {
    private final FlatfileLoaderService loader;

    public FlatfileLoaderController(FlatfileLoaderService loader) {
        this.loader = loader;
    }

    @PostMapping("/{interfaceId}/metadata")
    public FlatfileLoaderService.MetadataResult deriveMetadata(@PathVariable String interfaceId) {
        return loader.deriveMetadata(interfaceId);
    }

    @PostMapping("/{interfaceId}/load")
    public FlatfileLoaderService.LoadResult load(@PathVariable String interfaceId) {
        return loader.load(interfaceId);
    }
}
