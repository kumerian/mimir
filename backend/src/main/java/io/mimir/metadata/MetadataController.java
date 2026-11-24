package io.mimir.metadata;

import java.net.URI;
import java.util.List;

import jakarta.validation.Valid;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.servlet.support.ServletUriComponentsBuilder;

import static io.mimir.metadata.Metadata.*;

@RestController
@RequestMapping("/api/metadata")
public class MetadataController {
    private final MetadataStore store;

    public MetadataController(MetadataStore store) {
        this.store = store;
    }

    private <T extends Identified> ResponseEntity<T> created(T value) {
        T saved = store.create(value);
        URI location = ServletUriComponentsBuilder.fromCurrentRequest()
                .replaceQuery(null).queryParam("id", "{id}").encode().buildAndExpand(saved.id()).toUri();
        return ResponseEntity.created(location).body(saved);
    }

    @GetMapping(value = "/source-systems", params = "!id")
    public List<SourceSystem> sources() { return store.list(SourceSystem.class); }
    @GetMapping(value = "/source-systems", params = "id")
    public SourceSystem source(@RequestParam String id) { return store.get(id, SourceSystem.class); }
    @PostMapping("/source-systems")
    public ResponseEntity<SourceSystem> createSource(@Valid @RequestBody SourceSystem value) { return created(value); }
    @PutMapping("/source-systems")
    public SourceSystem updateSource(@RequestParam String id, @Valid @RequestBody SourceSystem value) { return store.update(id, value, SourceSystem.class); }
    @DeleteMapping("/source-systems")
    public ResponseEntity<Void> deleteSource(@RequestParam String id) {
        store.delete(id, SourceSystem.class);
        return ResponseEntity.noContent().build();
    }

    @GetMapping(value = "/authentications", params = "!id")
    public List<Authentication> authentications() { return store.list(Authentication.class); }
    @GetMapping(value = "/authentications", params = "id")
    public Authentication authentication(@RequestParam String id) { return store.get(id, Authentication.class); }
    @PostMapping("/authentications")
    public ResponseEntity<Authentication> createAuthentication(@Valid @RequestBody Authentication value) { return created(value); }
    @PutMapping("/authentications")
    public Authentication updateAuthentication(@RequestParam String id, @Valid @RequestBody Authentication value) { return store.update(id, value, Authentication.class); }
    @DeleteMapping("/authentications")
    public ResponseEntity<Void> deleteAuthentication(@RequestParam String id) {
        store.delete(id, Authentication.class);
        return ResponseEntity.noContent().build();
    }

    @GetMapping(value = "/api-parameters", params = "!id")
    public List<ApiParameter> apiParameters() { return store.list(ApiParameter.class); }
    @GetMapping(value = "/api-parameters", params = "id")
    public ApiParameter apiParameter(@RequestParam String id) { return store.get(id, ApiParameter.class); }
    @PostMapping("/api-parameters")
    public ResponseEntity<ApiParameter> createApiParameter(@Valid @RequestBody ApiParameter value) { return created(value); }
    @PutMapping("/api-parameters")
    public ApiParameter updateApiParameter(@RequestParam String id, @Valid @RequestBody ApiParameter value) { return store.update(id, value, ApiParameter.class); }
    @DeleteMapping("/api-parameters")
    public ResponseEntity<Void> deleteApiParameter(@RequestParam String id) {
        store.delete(id, ApiParameter.class);
        return ResponseEntity.noContent().build();
    }

    @GetMapping(value = "/interfaces", params = "!id")
    public List<SourceSystemInterface> interfaces() { return store.list(SourceSystemInterface.class); }
    @GetMapping(value = "/interfaces", params = "id")
    public SourceSystemInterface sourceInterface(@RequestParam String id) { return store.get(id, SourceSystemInterface.class); }
    @PostMapping("/interfaces")
    public ResponseEntity<SourceSystemInterface> createInterface(@Valid @RequestBody SourceSystemInterface value) { return created(value); }
    @PutMapping("/interfaces")
    public SourceSystemInterface updateInterface(@RequestParam String id, @Valid @RequestBody SourceSystemInterface value) { return store.update(id, value, SourceSystemInterface.class); }
    @DeleteMapping("/interfaces")
    public ResponseEntity<Void> deleteInterface(@RequestParam String id) {
        store.delete(id, SourceSystemInterface.class);
        return ResponseEntity.noContent().build();
    }

    @GetMapping(value = "/objects", params = "!id")
    public List<SourceObject> objects() { return store.list(SourceObject.class); }
    @GetMapping(value = "/objects", params = "id")
    public SourceObject object(@RequestParam String id) { return store.get(id, SourceObject.class); }
    @PostMapping("/objects")
    public ResponseEntity<SourceObject> createObject(@Valid @RequestBody SourceObject value) { return created(value); }
    @PutMapping("/objects")
    public SourceObject updateObject(@RequestParam String id, @Valid @RequestBody SourceObject value) { return store.update(id, value, SourceObject.class); }
    @DeleteMapping("/objects")
    public ResponseEntity<Void> deleteObject(@RequestParam String id) {
        store.delete(id, SourceObject.class);
        return ResponseEntity.noContent().build();
    }

    @GetMapping(value = "/columns", params = "!id")
    public List<SourceColumn> columns() { return store.list(SourceColumn.class); }
    @GetMapping(value = "/columns", params = "id")
    public SourceColumn column(@RequestParam String id) { return store.get(id, SourceColumn.class); }
    @PostMapping("/columns")
    public ResponseEntity<SourceColumn> createColumn(@Valid @RequestBody SourceColumn value) { return created(value); }
    @PutMapping("/columns")
    public SourceColumn updateColumn(@RequestParam String id, @Valid @RequestBody SourceColumn value) { return store.update(id, value, SourceColumn.class); }
    @DeleteMapping("/columns")
    public ResponseEntity<Void> deleteColumn(@RequestParam String id) {
        store.delete(id, SourceColumn.class);
        return ResponseEntity.noContent().build();
    }
}
