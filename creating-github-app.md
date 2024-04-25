
# Creating a BuildBeaver GitHub App

In order for a BuildBeaver Server to be able to build the code from GitHub repos, a GitHub app is used
to provide access rights to the server. By installing the GitHub app for a user account or repo a developer
can enable access to one or more repos.

## Creating and Configuring an App

To create a GitHub app within your GitHub account, go to the Settings for your account or GitHub Organization,
open *Developer Settings*, choose *GitHub Apps*, then click on the *New GitHub App* button.

Use the following settings for your GitHub App:

1. **Callback URL**: For running a BuildBeaver server on your development machine,
   use http://localhost:3000/api/v1/authentication/github/callback
   You could add a second callback URL https://localhost:3000/api/v1/authentication/github/callback (i.e. https).
   Check "Request user authorization (OAuth) during installation', but not "Enable Device Flow".

   For a running a BuildBeaver server on a Cloud-based VM or other server machine, substitute 'localhost:3000' for the real
   DNS name of your server, together with the port your server is listening on for the REST server API.

1. **Post installation**: Do not fill out "Setup URL (optional)"

1. **Webhook**: For running a BuildBeaver server on your development machine, paste in the smee URL you set up earlier
   for delivery of Webhook events to a developer machine, e.g. https://smee.io/sLCrZtAUgNUZZZ. This enables
   delivery of Webhook events without requiring your server to be contactable from the Internet.

   For running a BuildBeaver server listening on the Internet, put in your server's Webhook API address,
   as configured when running the server.

1. **SSL verification**: Leave "Enable SSL verification" checked

1. **Permissions**: Add the following permissions for the App (note that it may be possible to request a lesser
   set of permissions from the user, but would need to be carefully tested first):

    - Repository permissions:
        - Administration (read and write)
        - Commit statuses (read and write)
        - Contents (read-only)
        - Metadata (read-only)
        - Pull requests (read and write)
    - Organization permissions:
        - Administration (read-only)
        - Members (read-only)
        - Webhooks (read and write)
    - Account permissions: None

1. **Subscribe to events**: Check the following events to subscribe the app. This allows the server to build
   repos when things have changed, and to keep the list of Orgs, Repos and permissions in sync with GitHub:
    - Meta
    - Create
    - Member
    - Membership
    - Organisation
    - Public
    - Pull request
    - Push
    - Repository
    - Team
    - Team add
    - Org block

1. **Where can this GitHub App be installed?** Selecting "Any account" is recommended, unless you only want to
   use this BuildBeaver server with a single GitHub account.

Generate a **private key** for the GitHub app and download it.

Generate a **client secret** for the GitHub app and record it.

## Using the GitHub App

Once the GitHub App is created, it can be installed by any user into one or more Repos within their account or
within an Organization. The user will need the URL for your GitHub app.

## BuildBeaver Server Interactions with the GitHub App

The BuildBeaver server must be configured with the App ID, client secret, and private key for the GitHub app.
The server will receive Webhook notifications from GitHub each time the app is installed for a new
User, Org, or Repo.

The server will also periodically 'sync' with GitHub by using the GitHub API to list App installations it has
access to; this ensures that installations whose Webhook events that are missed will eventually be discovered.
