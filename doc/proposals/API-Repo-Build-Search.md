# Repo Build Search

Used by authenticated Users to search Builds for a given Repo.

**URL** : `/api/v1/repos/{repo_id}/builds/search`

**Method** : `POST`

**Auth required** : YES

**Data constraints**

```json
{
  "ref": "[ref]",
  "commit_sha": "[commit_sha]",
  "commit_author_id": "[commit_author_id]",
  "status": "[status]",
  "limit": "[limit]"
}
```

Where:  
* ref: The fully formed git ref
* commit_sha: Prefix match (minimum 4 characters) of a commit SHA
* author: Legal Entity ID that pushed a commit that triggered the build
* status: comma-separated array of workflow statuses
* limit: optional integer limit of total number of builds to return per page. Defaults to 10 if not specified

**Data example(s)**

Return running builds for *main*:  

```json
{
  "ref": "refs/heads/main",
  "commit_sha": "",
  "commit_author_id": "",
  "status": "running"
}
```

Return builds that have run for commit with SHA *1d65315ee61f7e1cae5a144197608bd5c2088b38* where commit_sha is a prefix match for the SHA:  

```json
{
  "ref": "",
  "commit_sha": "1d65315ee61f7e1cae5a144197608bd5c2088b38",
  "commit_author_id": "",
  "status": ""
}
```

```json
{
  "ref": "",
  "commit_sha": "1d65",
  "commit_author_id": "",
  "status": ""
}
```

```json
{
  "ref": "",
  "commit_sha": "1d65315",
  "commit_author_id": "",
  "status": ""
}
```

Return running builds for a given commit author:  

```json
{
  "ref": "",
  "commit_sha": "",
  "commit_author_id": "legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
  "status": "running"
}
```

Return queued / running / succeeded builds for the current repo:  

```json
{
  "ref": "",
  "commit_sha": "",
  "commit_author_id": "",
  "status": "queued,running,succeeded"
}
```



## Success Response(s)

All responses are limited to 2 builds for document size purposes

### No filtering (low limit for docs)

**Data** :

```json
{
  "limit": 2
}
```

**Code** : `200 OK`

**Response example**

```json
{
  "results": [
    {
      "repo": {
        "url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "build_search_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/builds/search",
        "secrets_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/secrets",
        "id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "updated_at": "2022-05-11T09:40:39.035084Z",
        "etag": "\"466f440db6d4d3aa\"",
        "legal_entity_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "name": "buildbeaver",
        "description": "The most bestest CI",
        "ssh_url": "ssh://foobar.com/buildbeaver.git",
        "http_url": "https://foobar.com/buildbeaver.git",
        "link": "https://foobar.com/buildbeaver",
        "default_branch": "master",
        "enabled": false,
        "external_id": {
          "system_name": "github",
          "resource_id": "123:123"
        },
        "external_metadata": ""
      },
      "commit": {
        "committer_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "author_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "id": "commit:8a2e33c5-7a9d-4cd0-bea3-3702d0c2337c",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "config_type": "yaml",
        "sha": "kugqhgqz",
        "message": "Test commit",
        "author_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "author_name": "",
        "author_email": "",
        "committer_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "committer_name": "",
        "committer_email": "",
        "link": ""
      },
      "stages": null,
      "url": "http://localhost:3001/api/v1/builds/build:043136c5-7fc2-47b5-a101-a265658676aa",
      "id": "build:043136c5-7fc2-47b5-a101-a265658676aa",
      "build_name": "Queued: 4",
      "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
      "created_at": "2022-05-11T09:40:39.127084Z",
      "updated_at": "2022-05-11T09:40:39.127084Z",
      "etag": "\"516c72906f8aa634\"",
      "commit_id": "commit:8a2e33c5-7a9d-4cd0-bea3-3702d0c2337c",
      "log_descriptor_id": "log-descriptor:a6bcffdb-c954-4e20-961f-66ba735ace98",
      "ref": "refs/master/HEAD",
      "status": "queued",
      "timings": {
        "queued_at": "2022-04-07T08:41:11.468787Z",
        "submitted_at": "2022-04-07T08:41:11.468787Z",
        "running_at": null,
        "finished_at": null,
        "canceled_at": null
      },
      "error": null,
      "opts": {
        "nodes_to_run": []
      },
      "log_descriptor_url": "http://localhost:3001/api/v1/logs/log-descriptor:a6bcffdb-c954-4e20-961f-66ba735ace98"
    },
    {
      "repo": {
        "url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "build_search_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/builds/search",
        "secrets_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/secrets",
        "id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "updated_at": "2022-05-11T09:40:39.035084Z",
        "etag": "\"466f440db6d4d3aa\"",
        "legal_entity_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "name": "buildbeaver",
        "description": "The most bestest CI",
        "ssh_url": "ssh://foobar.com/buildbeaver.git",
        "http_url": "https://foobar.com/buildbeaver.git",
        "link": "https://foobar.com/buildbeaver",
        "default_branch": "master",
        "enabled": false,
        "external_id": {
          "system_name": "github",
          "resource_id": "123:123"
        },
        "external_metadata": ""
      },
      "commit": {
        "committer_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "author_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "config_type": "yaml",
        "sha": "aslpfzsv",
        "message": "Test commit",
        "author_id": "legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "author_name": "",
        "author_email": "",
        "committer_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "committer_name": "",
        "committer_email": "",
        "link": ""
      },
      "stages": null,
      "url": "http://localhost:3001/api/v1/builds/build:0c49b1c3-541c-48ad-8e1d-3e8c806e57d2",
      "id": "build:0c49b1c3-541c-48ad-8e1d-3e8c806e57d2",
      "build_name": "Running: 4",
      "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
      "created_at": "2022-05-11T09:40:39.224585Z",
      "updated_at": "2022-05-11T09:40:39.224585Z",
      "etag": "\"958022ebb082a64b\"",
      "commit_id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
      "log_descriptor_id": "log-descriptor:11f5bc2a-fe28-4fd9-9d74-d8ef96cffd4a",
      "ref": "refs/master/HEAD",
      "status": "running",
      "timings": {
        "queued_at": "2022-04-07T08:41:11.468787Z",
        "submitted_at": "2022-04-07T08:41:11.468787Z",
        "running_at": "2022-04-07T08:41:11.468787Z",
        "finished_at": null,
        "canceled_at": null
      },
      "error": null,
      "opts": {
        "nodes_to_run": []
      },
      "log_descriptor_url": "http://localhost:3001/api/v1/logs/log-descriptor:11f5bc2a-fe28-4fd9-9d74-d8ef96cffd4a"
    }
  ],
  "prev_url": "",
  "next_url": "http://localhost:3001/api/v1/repos/repo:38c09f9f-a56f-400e-8aa7-ae4e40191c78/builds/search?cursor=eyJkIjoibiIsIm0iOiJidWlsZDowYzQ5YjFjMy01NDFjLTQ4YWQtOGUxZC0zZThjODA2ZTU3ZDIifQ%253D%253D&limit=2"
}
```


### Status filtering

**Data** :

```json
{
  "status": "running",
  "limit": 2
}
```

**Code** : `200 OK`

**Response example**

```json
{
  "results": [
    {
      "repo": {
        "url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "build_search_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/builds/search",
        "secrets_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/secrets",
        "id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "updated_at": "2022-05-11T09:40:39.035084Z",
        "etag": "\"466f440db6d4d3aa\"",
        "legal_entity_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "name": "buildbeaver",
        "description": "The most bestest CI",
        "ssh_url": "ssh://foobar.com/buildbeaver.git",
        "http_url": "https://foobar.com/buildbeaver.git",
        "link": "https://foobar.com/buildbeaver",
        "default_branch": "master",
        "enabled": false,
        "external_id": {
          "system_name": "github",
          "resource_id": "123:123"
        },
        "external_metadata": ""
      },
      "commit": {
        "committer_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "author_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "config_type": "yaml",
        "sha": "aslpfzsv",
        "message": "Test commit",
        "author_id": "legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "author_name": "",
        "author_email": "",
        "committer_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "committer_name": "",
        "committer_email": "",
        "link": ""
      },
      "stages": null,
      "url": "http://localhost:3001/api/v1/builds/build:0c49b1c3-541c-48ad-8e1d-3e8c806e57d2",
      "id": "build:0c49b1c3-541c-48ad-8e1d-3e8c806e57d2",
      "build_name": "Running: 4",
      "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
      "created_at": "2022-05-11T09:40:39.224585Z",
      "updated_at": "2022-05-11T09:40:39.224585Z",
      "etag": "\"958022ebb082a64b\"",
      "commit_id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
      "log_descriptor_id": "log-descriptor:11f5bc2a-fe28-4fd9-9d74-d8ef96cffd4a",
      "ref": "refs/master/HEAD",
      "status": "running",
      "timings": {
        "queued_at": "2022-04-07T08:41:11.468787Z",
        "submitted_at": "2022-04-07T08:41:11.468787Z",
        "running_at": "2022-04-07T08:41:11.468787Z",
        "finished_at": null,
        "canceled_at": null
      },
      "error": null,
      "opts": {
        "nodes_to_run": []
      },
      "log_descriptor_url": "http://localhost:3001/api/v1/logs/log-descriptor:11f5bc2a-fe28-4fd9-9d74-d8ef96cffd4a"
    },
    {
      "repo": {
        "url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "build_search_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/builds/search",
        "secrets_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/secrets",
        "id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "updated_at": "2022-05-11T09:40:39.035084Z",
        "etag": "\"466f440db6d4d3aa\"",
        "legal_entity_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "name": "buildbeaver",
        "description": "The most bestest CI",
        "ssh_url": "ssh://foobar.com/buildbeaver.git",
        "http_url": "https://foobar.com/buildbeaver.git",
        "link": "https://foobar.com/buildbeaver",
        "default_branch": "master",
        "enabled": false,
        "external_id": {
          "system_name": "github",
          "resource_id": "123:123"
        },
        "external_metadata": ""
      },
      "commit": {
        "committer_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "author_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "config_type": "yaml",
        "sha": "aslpfzsv",
        "message": "Test commit",
        "author_id": "legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "author_name": "",
        "author_email": "",
        "committer_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "committer_name": "",
        "committer_email": "",
        "link": ""
      },
      "stages": null,
      "url": "http://localhost:3001/api/v1/builds/build:16ccc492-3526-4392-8f8b-8ef08fcd7ac1",
      "id": "build:16ccc492-3526-4392-8f8b-8ef08fcd7ac1",
      "build_name": "Running: 6",
      "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
      "created_at": "2022-05-11T09:40:39.243084Z",
      "updated_at": "2022-05-11T09:40:39.243084Z",
      "etag": "\"1bfebd08bc38a080\"",
      "commit_id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
      "log_descriptor_id": "log-descriptor:acc2f4a2-6fb2-4335-acbe-8134f415c810",
      "ref": "refs/master/HEAD",
      "status": "running",
      "timings": {
        "queued_at": "2022-04-07T08:41:11.468787Z",
        "submitted_at": "2022-04-07T08:41:11.468787Z",
        "running_at": "2022-04-07T08:41:11.468787Z",
        "finished_at": null,
        "canceled_at": null
      },
      "error": null,
      "opts": {
        "nodes_to_run": []
      },
      "log_descriptor_url": "http://localhost:3001/api/v1/logs/log-descriptor:acc2f4a2-6fb2-4335-acbe-8134f415c810"
    }
  ],
  "prev_url": "",
  "next_url": "http://localhost:3001/api/v1/repos/repo:38c09f9f-a56f-400e-8aa7-ae4e40191c78/builds/search?cursor=eyJkIjoibiIsIm0iOiJidWlsZDoxNmNjYzQ5Mi0zNTI2LTQzOTItOGY4Yi04ZWYwOGZjZDdhYzEifQ%253D%253D&limit=2&status=running"
}
```


### Commit SHA filtering

**Data** :

```json
{
  "commit_sha": "aslpfzsv",
  "limit": 2
}
```

**Code** : `200 OK`

**Response example**

```json
{
  "results": [
    {
      "repo": {
        "url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "build_search_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/builds/search",
        "secrets_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/secrets",
        "id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "updated_at": "2022-05-11T09:40:39.035084Z",
        "etag": "\"466f440db6d4d3aa\"",
        "legal_entity_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "name": "buildbeaver",
        "description": "The most bestest CI",
        "ssh_url": "ssh://foobar.com/buildbeaver.git",
        "http_url": "https://foobar.com/buildbeaver.git",
        "link": "https://foobar.com/buildbeaver",
        "default_branch": "master",
        "enabled": false,
        "external_id": {
          "system_name": "github",
          "resource_id": "123:123"
        },
        "external_metadata": ""
      },
      "commit": {
        "committer_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "author_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "config_type": "yaml",
        "sha": "aslpfzsv",
        "message": "Test commit",
        "author_id": "legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "author_name": "",
        "author_email": "",
        "committer_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "committer_name": "",
        "committer_email": "",
        "link": ""
      },
      "stages": null,
      "url": "http://localhost:3001/api/v1/builds/build:0c49b1c3-541c-48ad-8e1d-3e8c806e57d2",
      "id": "build:0c49b1c3-541c-48ad-8e1d-3e8c806e57d2",
      "build_name": "Running: 4",
      "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
      "created_at": "2022-05-11T09:40:39.224585Z",
      "updated_at": "2022-05-11T09:40:39.224585Z",
      "etag": "\"958022ebb082a64b\"",
      "commit_id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
      "log_descriptor_id": "log-descriptor:11f5bc2a-fe28-4fd9-9d74-d8ef96cffd4a",
      "ref": "refs/master/HEAD",
      "status": "running",
      "timings": {
        "queued_at": "2022-04-07T08:41:11.468787Z",
        "submitted_at": "2022-04-07T08:41:11.468787Z",
        "running_at": "2022-04-07T08:41:11.468787Z",
        "finished_at": null,
        "canceled_at": null
      },
      "error": null,
      "opts": {
        "nodes_to_run": []
      },
      "log_descriptor_url": "http://localhost:3001/api/v1/logs/log-descriptor:11f5bc2a-fe28-4fd9-9d74-d8ef96cffd4a"
    },
    {
      "repo": {
        "url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "build_search_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/builds/search",
        "secrets_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/secrets",
        "id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "updated_at": "2022-05-11T09:40:39.035084Z",
        "etag": "\"466f440db6d4d3aa\"",
        "legal_entity_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "name": "buildbeaver",
        "description": "The most bestest CI",
        "ssh_url": "ssh://foobar.com/buildbeaver.git",
        "http_url": "https://foobar.com/buildbeaver.git",
        "link": "https://foobar.com/buildbeaver",
        "default_branch": "master",
        "enabled": false,
        "external_id": {
          "system_name": "github",
          "resource_id": "123:123"
        },
        "external_metadata": ""
      },
      "commit": {
        "committer_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "author_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "config_type": "yaml",
        "sha": "aslpfzsv",
        "message": "Test commit",
        "author_id": "legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "author_name": "",
        "author_email": "",
        "committer_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "committer_name": "",
        "committer_email": "",
        "link": ""
      },
      "stages": null,
      "url": "http://localhost:3001/api/v1/builds/build:16ccc492-3526-4392-8f8b-8ef08fcd7ac1",
      "id": "build:16ccc492-3526-4392-8f8b-8ef08fcd7ac1",
      "build_name": "Running: 6",
      "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
      "created_at": "2022-05-11T09:40:39.243084Z",
      "updated_at": "2022-05-11T09:40:39.243084Z",
      "etag": "\"1bfebd08bc38a080\"",
      "commit_id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
      "log_descriptor_id": "log-descriptor:acc2f4a2-6fb2-4335-acbe-8134f415c810",
      "ref": "refs/master/HEAD",
      "status": "running",
      "timings": {
        "queued_at": "2022-04-07T08:41:11.468787Z",
        "submitted_at": "2022-04-07T08:41:11.468787Z",
        "running_at": "2022-04-07T08:41:11.468787Z",
        "finished_at": null,
        "canceled_at": null
      },
      "error": null,
      "opts": {
        "nodes_to_run": []
      },
      "log_descriptor_url": "http://localhost:3001/api/v1/logs/log-descriptor:acc2f4a2-6fb2-4335-acbe-8134f415c810"
    }
  ],
  "prev_url": "",
  "next_url": "http://localhost:3001/api/v1/repos/repo:38c09f9f-a56f-400e-8aa7-ae4e40191c78/builds/search?commit_sha=aslpfzsv&cursor=eyJkIjoibiIsIm0iOiJidWlsZDoxNmNjYzQ5Mi0zNTI2LTQzOTItOGY4Yi04ZWYwOGZjZDdhYzEifQ%253D%253D&limit=2"
}
```


### Commit Author Id filtering

**Data** :

```json
{
  "commit_author_id": "legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
  "limit": 2
}
```

**Code** : `200 OK`

**Response example**

```json
{
  "results": [
    {
      "repo": {
        "url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "build_search_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/builds/search",
        "secrets_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/secrets",
        "id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "updated_at": "2022-05-11T09:40:39.035084Z",
        "etag": "\"466f440db6d4d3aa\"",
        "legal_entity_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "name": "buildbeaver",
        "description": "The most bestest CI",
        "ssh_url": "ssh://foobar.com/buildbeaver.git",
        "http_url": "https://foobar.com/buildbeaver.git",
        "link": "https://foobar.com/buildbeaver",
        "default_branch": "master",
        "enabled": false,
        "external_id": {
          "system_name": "github",
          "resource_id": "123:123"
        },
        "external_metadata": ""
      },
      "commit": {
        "committer_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "author_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "config_type": "yaml",
        "sha": "aslpfzsv",
        "message": "Test commit",
        "author_id": "legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "author_name": "",
        "author_email": "",
        "committer_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "committer_name": "",
        "committer_email": "",
        "link": ""
      },
      "stages": null,
      "url": "http://localhost:3001/api/v1/builds/build:0c49b1c3-541c-48ad-8e1d-3e8c806e57d2",
      "id": "build:0c49b1c3-541c-48ad-8e1d-3e8c806e57d2",
      "build_name": "Running: 4",
      "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
      "created_at": "2022-05-11T09:40:39.224585Z",
      "updated_at": "2022-05-11T09:40:39.224585Z",
      "etag": "\"958022ebb082a64b\"",
      "commit_id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
      "log_descriptor_id": "log-descriptor:11f5bc2a-fe28-4fd9-9d74-d8ef96cffd4a",
      "ref": "refs/master/HEAD",
      "status": "running",
      "timings": {
        "queued_at": "2022-04-07T08:41:11.468787Z",
        "submitted_at": "2022-04-07T08:41:11.468787Z",
        "running_at": "2022-04-07T08:41:11.468787Z",
        "finished_at": null,
        "canceled_at": null
      },
      "error": null,
      "opts": {
        "nodes_to_run": []
      },
      "log_descriptor_url": "http://localhost:3001/api/v1/logs/log-descriptor:11f5bc2a-fe28-4fd9-9d74-d8ef96cffd4a"
    },
    {
      "repo": {
        "url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "build_search_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/builds/search",
        "secrets_url": "http://localhost:3001/api/v1/repos/repo:c8cbad4f-6721-437f-9ce3-253a989b7239/secrets",
        "id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "updated_at": "2022-05-11T09:40:39.035084Z",
        "etag": "\"466f440db6d4d3aa\"",
        "legal_entity_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "name": "buildbeaver",
        "description": "The most bestest CI",
        "ssh_url": "ssh://foobar.com/buildbeaver.git",
        "http_url": "https://foobar.com/buildbeaver.git",
        "link": "https://foobar.com/buildbeaver",
        "default_branch": "master",
        "enabled": false,
        "external_id": {
          "system_name": "github",
          "resource_id": "123:123"
        },
        "external_metadata": ""
      },
      "commit": {
        "committer_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "author_url": "http://localhost:3001/api/v1/legal-entities/legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
        "created_at": "2022-05-11T09:40:39.035084Z",
        "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
        "config_type": "yaml",
        "sha": "aslpfzsv",
        "message": "Test commit",
        "author_id": "legal-entity:8f721dfb-25e5-4ba7-8693-28227a334100",
        "author_name": "",
        "author_email": "",
        "committer_id": "legal-entity:8f721dfb-15e5-4ba7-8693-28227a334100",
        "committer_name": "",
        "committer_email": "",
        "link": ""
      },
      "stages": null,
      "url": "http://localhost:3001/api/v1/builds/build:16ccc492-3526-4392-8f8b-8ef08fcd7ac1",
      "id": "build:16ccc492-3526-4392-8f8b-8ef08fcd7ac1",
      "build_name": "Running: 6",
      "repo_id": "repo:c8cbad4f-6721-437f-9ce3-253a989b7239",
      "created_at": "2022-05-11T09:40:39.243084Z",
      "updated_at": "2022-05-11T09:40:39.243084Z",
      "etag": "\"1bfebd08bc38a080\"",
      "commit_id": "commit:c07471a9-44ad-4754-aba3-41deec6b5e5d",
      "log_descriptor_id": "log-descriptor:acc2f4a2-6fb2-4335-acbe-8134f415c810",
      "ref": "refs/master/HEAD",
      "status": "running",
      "timings": {
        "queued_at": "2022-04-07T08:41:11.468787Z",
        "submitted_at": "2022-04-07T08:41:11.468787Z",
        "running_at": "2022-04-07T08:41:11.468787Z",
        "finished_at": null,
        "canceled_at": null
      },
      "error": null,
      "opts": {
        "nodes_to_run": []
      },
      "log_descriptor_url": "http://localhost:3001/api/v1/logs/log-descriptor:acc2f4a2-6fb2-4335-acbe-8134f415c810"
    }
  ],
  "prev_url": "",
  "next_url": "http://localhost:3001/api/v1/repos/repo:38c09f9f-a56f-400e-8aa7-ae4e40191c78/builds/search?commit_author_id=legal-entity%253A8f721dfb-25e5-4ba7-8693-28227a334100&cursor=eyJkIjoibiIsIm0iOiJidWlsZDoxNmNjYzQ5Mi0zNTI2LTQzOTItOGY4Yi04ZWYwOGZjZDdhYzEifQ%253D%253D&limit=2"
}
```

## Error Response

**Condition** : Invalid parameter request

**Code** : `400 BAD REQUEST`

**Content** :

```json
{
  "code": "InvalidParameters",
  "http_status_code": 400,
  "message": "Provided status <status> does not exist",
  "details": "should be one of <workflowstatus list>"
}
```