import React from 'react';
import { IRootDocument } from '../../interfaces/root-document.interface';

export const RootContext = React.createContext<IRootDocument>({} as IRootDocument);
