import React, { useEffect, useState } from 'react';
import { IRootDocument } from '../../interfaces/root-document.interface';
import { RootContext } from './root.context';
import { fetchRootDocument } from '../../services/root.service';
import { Loading } from '../../components/loading/loading.component';
import { Fatality } from '../../components/fatality/fatality.component';
import { IStructuredError } from '../../interfaces/structured-error.interface';

export function RootDocumentProvider(props: any): JSX.Element {
  const [rootDocument, setRootDocument] = useState<IRootDocument | null>(null);
  const [rootDocumentError, setRootDocumentError] = useState<IStructuredError | undefined>();

  useEffect(() => {
    const getRootDocument = async () => {
      await fetchRootDocument()
        .then((result: IRootDocument) => {
          setRootDocument(result);
        })
        .catch((error: IStructuredError) => {
          setRootDocumentError(error);
        });
    };

    getRootDocument();
  }, []);

  if (rootDocumentError) {
    return <Fatality error={rootDocumentError} />;
  }

  if (rootDocument) {
    return <RootContext.Provider value={rootDocument}>{props.children}</RootContext.Provider>;
  }

  return <Loading />;
}
