/**
 * Represents a single piece of a breadcrumb title.
 */
export interface IBreadcrumb {
  /**
   * The label for this particular part of the overall breadcrumb title.
   */
  label: string;

  /**
   * The path to redirect to when clicked.
   */
  path: string;
}
