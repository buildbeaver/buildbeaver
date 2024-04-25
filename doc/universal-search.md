# Universal Search - Design

# Introduction

Universal search provides a single unified query API for searching across all resources in the system.

A search query consists of a term and/or one or more qualifiers. The term is arbitrary text that will be fuzzy matched
on all fields of all resources by default. Qualifiers modify the search, typically filtering it to a specific resource
kind, or limiting the term to match on a specific field instead of all.

There are three main phases in the execution of a search. The GUI first receives the search query as plaintext via an
input box, the query is then converted to a structured document and submitted to the API, and the backend finally
unpacks the structured document and executes the search. Each resource kind is searched independently and the search
results contain an inner set of results for each kind.

This design is a simplified approximation of GitHub's universal search. See [GitHub](https://docs.github.com/en/search-github/getting-started-with-searching-on-github/about-searching-on-github).

All examples in this document use the plaintext query format that would be entered in the GUI.

# Qualifiers

Qualifiers are a tuple made of _command_ and _constraint_ components. The _command_ determines how the corresponding
constraint is applied and the _constraint_ provides data to the command.

## Commands

**kind**:

The `kind` command filters the search results to a specific resource kind specified in `constraint` e.g `kind:repo`. 
`kind` may be specified more than once, building up a set of kinds in a logical OR.

**in**:

`in` instructs the search to match `term` against the field specified in `constraint` (as opposed to the default behaviour of
matching all fields) e.g. `hello in:description` will match the term `hello` in all resources that have a `description` field.
Each resource has a different set of supported fields to match on. If a resource does not have a field with a matching 
name it is excluded from the search. `in` may be specified more than once, building up a set of fields to match on via logical OR.

**fields**:

Each resource may nominate zero or more fields that they support being filtered on via an exact match. These field names
are then able to be targeted via corresponding `command` names e.g. `enabled:true` will filter search results to resources
that have an `enabled` field with the value `true`. Each resource is free to infer the value of the `constraint` field based
on its data model. e.g. hypothetically if two resources both had an `enabled` field, where one was a boolean and the other a string,
they would interpret the `constraint` value as bool=true and string="true" respectively. field commands may be specified
more than once, building up a set of fields to match on via logical AND.

**sort**:

`sort` modifies the sort order of the search results by sorting on the field specified in the `constraint`. Fields
are sorted in ascending order by default, and this can be modified by suffixing the field name with `-asc` or `-desc`.
If a given resource does not contain the specified field the sort is ignored for that resource kind. `sort` can only
be specified once.

## Constraints

### Equality

Field commands can begin with `!=`, `>`, `>=`, `<`, or `<=` to change the equality rules of the match e.g. `kind:artifact size:>1024`
to find artifacts larger than 1KB. `=` is inferred by default.

### Bools

Field commands can match boolean values using `true` or `false`, or `1` or `0` e.g. `enabled:true` or `enabled:1` to find resources
with an `enabled` field set to true.

### Shortcuts

A constraint that exactly matches `@me` expands to the currently authenticated user's name.

### Deferred

The constraint syntax can evolve over time to support additional data types and equality checks. In particular, support
for dates and ranges may be added.

# Resources

## Repos

**kind**: `repo`

**in**:

* `name`
* `description`

**fields**:

* `user`
* `org`
* `enabled`

## Builds

**kind**: `build`

**in**:

* `commit-message`

**fields**:

* `repo`
* `user`
* `org`
* `status`
* `ref`
* `hash`
* `author`
* `author-name`
* `author-email`
* `committer`
* `committer-name`
* `committer-email`

# Plaintext Syntax

The GUI will be responsible for taking the query's text representation and mapping it to a structured document to submit
to the search API. The query contains at least a term, or one qualifier. The term may come before or after the qualifier(s).
Qualifier tuples are separated by a colon. The term may optionally be quoted to disambiguate it from a qualifier (e.g. if
the term itself happens to be foo:bar). 

## Examples

* `` is an invalid query
* `kind:repo hello enabled:true` is invalid, because the term does not come before or after the qualifier(s)
* `hello kind:repo` is equivalent to `kind:repo hello`, and both are valid
* `foobar` contains a term but no qualifiers, and is valid
* `"foo:bar"` contains a term but no qualifiers, and is valid
* `foo:bar` contains a qualifier but no term, and is valid
* `buildbeaver kind:repo` searches all repos with any fields beginning with `buildbeaver`
* `hash:ab342e9` searches all resources with a `hash` property matching `ab342e9`
* `repo:buildbeaver/buildbeaver-ng hash:ab342e9` searches all resources owned by the repo `buildbeaver/buildbeaver-ng` with a `hash`  property matching `ab342e9`
* `ticket-54 in:commit-message kind:build` searches all builds that ran for a commit that contains `ticket-54` in the message field
* `kind:build sort:created_at-desc` searches all builds, sorting them by `created_at`, ordered by newest to oldest

# APIs

## Create Search

### Request

**Path**: `/api/v1/search`

**Method**: `POST`

**Auth Required**: Yes

**Query**: None

**Body**:

```json
{
  "term": "[term]",
  "qualifiers": {
    "[command]": "[constraint]",
    "[command]": "[constraint]",
    ...
  }
}
```

Where:

* `term` - Free form text
* `qualifiers` - A map of zero or more command/constraint tuples

### Responses

#### Success

`303` Redirects to `GET` `/api/v1/search?q=[query]`

#### Errors

##### ValidationFailed

Returned if the search document does not contain a term or a qualifier.

**Status Code**: `400`

**Body**:

```json
{
  "code": "ValidationFailed",
  "http_status_code": 400,
  "message": "At least term or one qualifier must be specified"
}
```

## Get Search Results

**Path**: `/api/v1/search?q=[query]`

**Method**: `GET`

**Auth Required**: Yes

**Query**:

All query parameters must be url encoded.

* `q` - The full search query

**Body**: None

### Responses

#### Success

**Status Code**: `200`

**Body**:

The response body contains a list of lists. Each sub list contains results for a specific resource kind. A sub list
will only be present for a given kind if there was at least one result for that kind. Each list contains the standard
list metadata (results + next/prev urls). For now, next/prev urls will never be populated, but in future each sub list
may be independently paged if needed (to support e.g. a dedicated search results page). The outermost list will never
be paged. The server will determine a suitable limit (it is not configurable via query params).

```json
{
  "results": [
    {
      "kind": "build",
      "results": [
        {
          "kind": "build",
          ...
        },
        {
          "kind": "build",
          ...
        }
      ],
      "prev_url": "<always empty>",
      "next_url": "<always empty>"
    },
    {
      "kind": "repo",
      "results": [
        {
          "kind": "repo",
          ...
        },
        {
          "kind": "repo",
          ...
        }
      ],
      "prev_url": "<always empty>",
      "next_url": "<always empty>"
    }
  ],
  "prev_url": "<always empty>",
  "next_url": "<always empty>"
}
```

#### Errors

##### ValidationFailed

Returned if the search document does not contain a valid query.

**Status Code**: `400`

**Body**:

```json
{
  "code": "ValidationFailed",
  "http_status_code": 400,
  "message": "Invalid query"
}
```