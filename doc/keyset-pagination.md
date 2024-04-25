!! This isn't meant to be a comprehensive doc.

We do keyset aka cursor aka seek pagination when listing resources in the DB. The logic is all encapsulated in `resource_table.go`.

Useful resources for understanding how this works:

* https://vladmihalcea.com/sql-seek-keyset-pagination/
* https://stackoverflow.com/questions/56719611/generic-sql-predicate-to-use-for-keyset-pagination-on-multiple-fields
* https://ask.use-the-index-luke.com/questions/205/how-to-query-for-previous-page-with-keyset-pagination
* https://pganalyze.com/blog/pagination-django-postgres
* https://welcometosoftware.com/cursor-pagination-previous-page/