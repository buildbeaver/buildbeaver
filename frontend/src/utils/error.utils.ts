import { IStructuredError } from '../interfaces/structured-error.interface';

/**
 * Retrieves a human-readable error message from a structured error, if available. Falls back to an optionally provided
 * message.
 */
export function getStructuredErrorMessage(error: IStructuredError, fallback = 'Something went wrong'): string {
  return error?.serverError?.message ?? fallback;
}
