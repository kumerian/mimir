import { Routes } from '@angular/router';
import { InterfaceDetails } from './loader/interface-details';
import { MetadataEditor } from './metadata/metadata-editor';

export const routes: Routes = [
  { path: '', component: MetadataEditor },
  { path: 'interfaces/:id', component: InterfaceDetails },
  { path: '**', redirectTo: '' },
];
