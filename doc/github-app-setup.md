# GitHub App Setup

This document describes how to set up a GitHub app to work with a BuildBeaver server.

Every BuildBeaver server needs a GitHub app for users to install on their accounts and repos, with
a corresponding private key that needs to be added to the server.

## Basic Setup

To set up a new GitHub app for use with a BuildBeaver server, go to your organization's
Settings page on GitHub and select "Developer Settings | GitHub Apps". Then click
on "New GitHub App" and follow the prompts.

In the "Identifying and authorizing users" section, configure a callback URL for
OAuth authentication, and check the "Request user authorization (OAuth) during
installation" checkbox. For example, the callback URL for the staging environment is:

Callback URL: `https://app.staging.changeme.com/api/v1/authentication/github/callback`


## Webhook Setup

The GitHub app must be configured to send Webhook notifications to the BuildBeaver
server. This can be done on the 'Register New GitHub App' screen, or by clicking on
the Edit button beside an existing GitHub app.

On the General tab, scroll down to the 'Webhook section' and add the Webhook URL
for your server. For example, for the staging server the URL is:

Webhook URL: `https://app.staging.changeme.com/api/v1/webhooks/github`

You should replace 'app.staging.changeme.com' with the DNS name and port (if not
the default of 80 for HTTP or 443 for HTTPS) on which your server is serving up its Core API.

We intend to support Webhook secrets Real Soon Now... (but not as of the time of writing).

### Webhooks with smee

For servers running on developer machines you can use a smee endpoint URL and run
smee on the same machine as the server (e.g. a developer laptop).

To do this, go to the BuildBeaver Smee server: https://smee.dev.changeme.com:3000/
 and then click on the "Start a new channel" button. Then copy the Webhook Proxy URL
from the top of the page and use that to configure the GitHub webhook, e.g.

Webhook URL: `https://smee.dev.changeme.com:3000/cOfj15RMgIUoa98A`


### Permissions Setup

The BuildBeaver server must be granted permissions for user's repositories in order
to function as a CI system.

**NOTE: These instructions are subject to a final review; we may be able to reduce
the set of required permissions needed by the BuildBeaver GitHub app.**

To configure permissions, go to the "Permissions" section when creating a new app, or
edit an existing GitHub app and go to the "Permissions and Events" tab.  In the "Permissions"
section, set the following permissions (all others should be set to "No access":

- Repository Permissions:
  - Administration: Read and write
  - Commit statuses: Read and write
  - Contents: Read-only
  - Metadata: Read-only
  - Pull requests: Read and write

- Organization Permissions:
  - Administration: Read-only
  - Members: Read-only
  - Webhooks: Read and write

- Account Permissions: (no access)


### Webhook events setup

The BuildBeaver server must be sent Webhook events for a variety of situations, in
order to keep its local view of users, organizations, repos and commits up to date.

To configure events, go to the "Subscribe to events" section when creating a new app, or
edit an existing GitHub app and go to the "Permissions and events" tab. In the "Subscribe to events"
section, ensure the following events are checked:

- Meta
- Create
- Member
- Membership
- Organization
- Public
- Pull request
- Push
- Repository
- Team
- Team add
- Org block
