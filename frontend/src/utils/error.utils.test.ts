import { getStructuredErrorMessage } from './error.utils';
import { IStructuredError } from '../interfaces/structured-error.interface';

describe('error.utils', () => {
  describe('getStructuredErrorMessage', () => {
    it('should return a server error message', () => {
      const error: IStructuredError = {
        serverError: {
          code: '1234',
          details: undefined,
          http_status_code: 400,
          message: 'Validation failed'
        },
        statusCode: 400,
        statusText: 'Bad request'
      };

      expect(getStructuredErrorMessage(error)).toEqual('Validation failed');
    });

    it('should return a specified fallback message', () => {
      const error: IStructuredError = {
        statusCode: 400,
        statusText: 'Bad request'
      };

      expect(getStructuredErrorMessage(error, 'Creation of thing failed')).toEqual('Creation of thing failed');
    });

    it('should return the default fallback message', () => {
      const error: IStructuredError = {
        statusCode: 400,
        statusText: 'Bad request'
      };

      expect(getStructuredErrorMessage(error)).toEqual('Something went wrong');
    });
  });
});
