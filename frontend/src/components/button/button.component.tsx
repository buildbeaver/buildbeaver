import React from 'react';
import ContentLoader from 'react-content-loader';

type ButtonSize = 'full' | 'large' | 'regular' | 'small' | 'x-small';
type ButtonType = 'primary' | 'secondary' | 'danger';

interface Props {
  children?: React.ReactNode;
  label?: string;
  loading?: boolean;
  loadingLabel?: string;
  onClick?: () => unknown;
  size?: ButtonSize;
  type?: ButtonType;
  disabled?: boolean;
}

export function Button(props: Props): JSX.Element {
  const { children, label, loading, loadingLabel, onClick, size, type, disabled } = props;

  const buttonSize = `btn-${size ?? 'regular'}`;
  const buttonType = `btn-${type ?? 'primary'}`;
  const buttonText = loading && loadingLabel ? loadingLabel : label;

  const onButtonClick = (event: React.MouseEvent): void => {
    if (!loading && !disabled && onClick) {
      onClick();
      event.preventDefault();
    }
  };

  return (
    <button
      disabled={disabled || loading}
      className={`${buttonSize} ${buttonType} btn relative h-8 min-w-0 text-sm`}
      onClick={onButtonClick}
      title={buttonText}
    >
      {loading && (
        <ContentLoader className="absolute top-0" speed={1} height="100%" width="100%">
          <rect height="100%" x="0" y="0" rx="3" ry="3" width="100%" />
        </ContentLoader>
      )}
      <div className="relative flex h-full w-full items-center justify-center px-2">
        <span className="... truncate">{buttonText}</span>
        {children}
      </div>
    </button>
  );
}
