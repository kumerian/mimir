import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

export interface MetadataRecord {
  [field: string]: string | null;
}

export interface SourceSystem extends MetadataRecord {
  sourceName: string;
  sourceFunctionalName: string;
  sourceType: string;
}
export interface Authentication extends MetadataRecord {
  authenticationId: string;
  authenticationType: string;
  authenticationLocation: string;
}
export interface SourceSystemInterface extends MetadataRecord {
  interfaceName: string;
  sourceName: string;
  authenticationId: string | null;
  interfaceFunctionalName: string;
  interfaceType: string;
  interfaceLocation: string;
}
export interface SourceObject extends MetadataRecord {
  objectName: string;
  interfaceName: string;
  objectFunctionalName: string;
}
export interface SourceColumn extends MetadataRecord {
  columnName: string;
  objectName: string;
  columnFunctionalName: string;
  datatype: string;
}

export interface MetadataTypes {
  'source-systems': SourceSystem;
  authentications: Authentication;
  interfaces: SourceSystemInterface;
  objects: SourceObject;
  columns: SourceColumn;
}
export type Resource = keyof MetadataTypes;

export interface Field {
  name: string;
  label: string;
  optional?: boolean;
  reference?: Resource;
  referenceKey?: string;
  maxLength?: number;
}
export interface Entity {
  resource: Resource;
  label: string;
  key: string;
  fields: readonly Field[];
}

export const ENTITIES: readonly Entity[] = [
  {
    resource: 'source-systems',
    label: 'Source systems',
    key: 'sourceName',
    fields: [
      { name: 'sourceName', label: 'Source name' },
      { name: 'sourceFunctionalName', label: 'Source functional name' },
      { name: 'sourceType', label: 'Source type' },
    ],
  },
  {
    resource: 'authentications',
    label: 'Authentications',
    key: 'authenticationId',
    fields: [
      { name: 'authenticationId', label: 'Authentication id' },
      { name: 'authenticationType', label: 'Authentication type' },
      { name: 'authenticationLocation', label: 'Authentication location', maxLength: 2048 },
    ],
  },
  {
    resource: 'interfaces',
    label: 'Interfaces',
    key: 'interfaceName',
    fields: [
      { name: 'interfaceName', label: 'Interface name' },
      {
        name: 'sourceName',
        label: 'Source system',
        reference: 'source-systems',
        referenceKey: 'sourceName',
      },
      {
        name: 'authenticationId',
        label: 'Authentication',
        optional: true,
        reference: 'authentications',
        referenceKey: 'authenticationId',
      },
      { name: 'interfaceFunctionalName', label: 'Interface functional name' },
      { name: 'interfaceType', label: 'Interface type' },
      { name: 'interfaceLocation', label: 'Interface location', maxLength: 2048 },
    ],
  },
  {
    resource: 'objects',
    label: 'Objects',
    key: 'objectName',
    fields: [
      { name: 'objectName', label: 'Object name' },
      {
        name: 'interfaceName',
        label: 'Interface',
        reference: 'interfaces',
        referenceKey: 'interfaceName',
      },
      { name: 'objectFunctionalName', label: 'Object functional name' },
    ],
  },
  {
    resource: 'columns',
    label: 'Columns',
    key: 'columnName',
    fields: [
      { name: 'columnName', label: 'Column name' },
      { name: 'objectName', label: 'Object', reference: 'objects', referenceKey: 'objectName' },
      { name: 'columnFunctionalName', label: 'Column functional name' },
      { name: 'datatype', label: 'Datatype' },
    ],
  },
];

@Injectable({ providedIn: 'root' })
export class MetadataApi {
  private readonly http = inject(HttpClient);
  private url(resource: Resource, id?: string) {
    return `/api/metadata/${resource}${id === undefined ? '' : '?id=' + encodeURIComponent(id)}`;
  }
  list<R extends Resource>(resource: R) {
    return this.http.get<MetadataTypes[R][]>(this.url(resource));
  }
  create(resource: Resource, value: MetadataRecord) {
    return this.http.post<MetadataRecord>(this.url(resource), value);
  }
  update(resource: Resource, id: string, value: MetadataRecord) {
    return this.http.put<MetadataRecord>(this.url(resource, id), value);
  }
  delete(resource: Resource, id: string) {
    return this.http.delete<void>(this.url(resource, id));
  }
}
