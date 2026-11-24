import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

export interface MetadataRecord {
  [field: string]: string | number | boolean | null;
}

export interface SourceSystem extends MetadataRecord {
  sourceName: string;
  sourceFunctionalName: string;
  sourceType: string;
  sourceOwner: string | null;
  sourceApplication: string;
  sourceLocation: string | null;
  isActive: boolean;
}
export interface Authentication extends MetadataRecord {
  authenticationId: string;
  authenticationType: string;
  authenticationLocation: string;
  authenticationUser: string | null;
  authenticationCredential: string | null;
  isActive: boolean;
}
export interface ApiParameter extends MetadataRecord {
  parameterId: string;
  interfaceId: string;
  parameterName: string;
  parameterValue: string | null;
}
export interface SourceSystemInterface extends MetadataRecord {
  interfaceId: string;
  interfaceName: string;
  sourceName: string;
  authenticationId: string | null;
  interfaceFunctionalName: string;
  interfaceType: string;
  interfaceLocation: string;
  decimalSeparator: string;
  thousandsSeparator: string;
  delimiter: string | null;
  hasHeaders: boolean | null;
  isActive: boolean;
}
export interface SourceObject extends MetadataRecord {
  objectId: string;
  objectName: string;
  interfaceId: string;
  objectFunctionalName: string;
  objectDescription: string | null;
  isConfidential: boolean;
  isBusinessCritical: boolean;
  isActive: boolean;
}
export interface SourceColumn extends MetadataRecord {
  columnId: string;
  columnName: string;
  objectId: string;
  columnFunctionalName: string;
  columnAlternateName: string | null;
  columnDescription: string | null;
  columnOrdinalPosition: number;
  needsEncryption: boolean;
  isPartitionField: boolean;
  partitionOrdinalPosition: number;
  isKey: boolean;
  keyOrdinalPosition: number;
  isNullable: boolean;
  testDatatype: boolean;
  datatype: string;
  characterMaximumLength: number | null;
  numericScale: number | null;
  numericPrecision: number | null;
  nullValueRegexPattern: string | null;
  isActive: boolean;
}

export interface MetadataTypes {
  'source-systems': SourceSystem;
  authentications: Authentication;
  'api-parameters': ApiParameter;
  interfaces: SourceSystemInterface;
  objects: SourceObject;
  columns: SourceColumn;
}
export type Resource = keyof MetadataTypes;

export interface Field {
  name: string;
  label: string;
  optional?: boolean;
  kind?: 'text' | 'boolean' | 'number' | 'password';
  reference?: Resource;
  referenceKey?: string;
  maxLength?: number;
  sensitive?: boolean;
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
      { name: 'sourceFunctionalName', label: 'Source functional name', optional: true },
      { name: 'sourceType', label: 'Source type' },
      { name: 'sourceOwner', label: 'Source owner', optional: true },
      { name: 'sourceApplication', label: 'Source application' },
      { name: 'sourceLocation', label: 'Source location', optional: true },
      { name: 'isActive', label: 'Is active', kind: 'boolean' },
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
      { name: 'authenticationUser', label: 'Authentication user', optional: true },
      {
        name: 'authenticationCredential',
        label: 'Authentication credential',
        optional: true,
        kind: 'password',
        sensitive: true,
      },
      { name: 'isActive', label: 'Is active', kind: 'boolean' },
    ],
  },
  {
    resource: 'api-parameters',
    label: 'API parameters',
    key: 'parameterId',
    fields: [
      { name: 'parameterId', label: 'Parameter id' },
      {
        name: 'interfaceId',
        label: 'Interface',
        reference: 'interfaces',
        referenceKey: 'interfaceId',
      },
      { name: 'parameterName', label: 'Parameter name' },
      { name: 'parameterValue', label: 'Parameter value', optional: true },
    ],
  },
  {
    resource: 'interfaces',
    label: 'Interfaces',
    key: 'interfaceId',
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
      { name: 'interfaceFunctionalName', label: 'Interface functional name', optional: true },
      { name: 'interfaceType', label: 'Interface type' },
      { name: 'interfaceLocation', label: 'Interface location', maxLength: 2048 },
      { name: 'decimalSeparator', label: 'Decimal separator', maxLength: 1 },
      { name: 'thousandsSeparator', label: 'Thousands separator', maxLength: 1 },
      { name: 'delimiter', label: 'Delimiter', optional: true, maxLength: 1 },
      { name: 'hasHeaders', label: 'Has headers', optional: true, kind: 'boolean' },
      { name: 'isActive', label: 'Is active', kind: 'boolean' },
    ],
  },
  {
    resource: 'objects',
    label: 'Objects',
    key: 'objectId',
    fields: [
      { name: 'objectName', label: 'Object name' },
      {
        name: 'interfaceId',
        label: 'Interface',
        reference: 'interfaces',
        referenceKey: 'interfaceId',
      },
      { name: 'objectFunctionalName', label: 'Object functional name', optional: true },
      { name: 'objectDescription', label: 'Object description', optional: true },
      { name: 'isConfidential', label: 'Is confidential', kind: 'boolean' },
      { name: 'isBusinessCritical', label: 'Is business critical', kind: 'boolean' },
      { name: 'isActive', label: 'Is active', kind: 'boolean' },
    ],
  },
  {
    resource: 'columns',
    label: 'Columns',
    key: 'columnId',
    fields: [
      { name: 'columnName', label: 'Column name' },
      { name: 'objectId', label: 'Object', reference: 'objects', referenceKey: 'objectId' },
      { name: 'columnAlternateName', label: 'Column alternate name', optional: true },
      { name: 'columnFunctionalName', label: 'Column functional name', optional: true },
      { name: 'columnDescription', label: 'Column description', optional: true },
      { name: 'columnOrdinalPosition', label: 'Column ordinal position', kind: 'number' },
      { name: 'needsEncryption', label: 'Needs encryption', kind: 'boolean' },
      { name: 'isPartitionField', label: 'Is partition field', kind: 'boolean' },
      { name: 'partitionOrdinalPosition', label: 'Partition ordinal position', kind: 'number' },
      { name: 'isKey', label: 'Is key', kind: 'boolean' },
      { name: 'keyOrdinalPosition', label: 'Key ordinal position', kind: 'number' },
      { name: 'isNullable', label: 'Is nullable', kind: 'boolean' },
      { name: 'testDatatype', label: 'Test datatype', kind: 'boolean' },
      { name: 'datatype', label: 'Datatype' },
      { name: 'characterMaximumLength', label: 'Character maximum length', optional: true, kind: 'number' },
      { name: 'numericScale', label: 'Numeric scale', optional: true, kind: 'number' },
      { name: 'numericPrecision', label: 'Numeric precision', optional: true, kind: 'number' },
      { name: 'nullValueRegexPattern', label: 'Null value regex pattern', optional: true },
      { name: 'isActive', label: 'Is active', kind: 'boolean' },
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
