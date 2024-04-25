import React, { useEffect, useState } from 'react';
import { Button } from '../button/button.component';
import { IResourceResponse } from '../../services/responses/resource-response.interface';

interface Props {
  response: IResourceResponse<any>;
  pageChanged: (url: string) => void;
}

/**
 * Provides Prev and Next buttons for paging through resource responses
 */
export function Pagination(props: Props): JSX.Element {
  const { response, pageChanged } = props;
  const [disableButtons, setDisableButtons] = useState(true);

  useEffect(() => {
    // Re-enable the buttons when another page of results comes in
    setDisableButtons(false);
  }, [response]);

  const buttonClicked = (url: string): void => {
    // Disable the buttons when we request another page of results
    setDisableButtons(true);
    pageChanged(url);
  };

  const button = (label: string, url: string): JSX.Element => {
    if (url === '') {
      // Always render something so buttons are spaced apart
      return <div></div>;
    }

    return <Button disabled={disableButtons} label={label} size="small" type="secondary" onClick={() => buttonClicked(url)} />;
  };

  return (
    <div className="flex items-center justify-between">
      {button('Prev', response.prev_url)}
      {button('Next', response.next_url)}
    </div>
  );
}
