# kubelogin

Repo for kubelogin Server and CLI.

# CLI

## Usage

The intended usage of this CLI is to communicate with the kubelogin server to
set the token field of the kubectl config file. The kubernetes API server will
use this token for OIDC authentication.

The CLI accepts two verbs: **`login`** and **`config`**

How to use these verbs:

| Verb | Flags | Description | Example |
| :--- | :--- | :--- | :--- |
| `config` | `alias`, `server-url`, `kubectl-user` | If no alias flag is set, the alias is set as default. If kubectl-user isn't set, it defaults to kubelogin_user. Server **MUST** be set. If there is no existing config file, this verb will create one for you in your root directory and put the initial values in the file for you. If you give an alias that already exists, it will update the info of the given alias. If you give a new alias, it will add that to the existing list of aliases | `kubelogin config --alias=foo --server-url=bar --kubectl-user=foobar` |
| `login ALIAS` | no flags | this command will take the alias given and search for it in the config file. If no value is found, it will error out and ask you to check spelling or create a config file. | `kubelogin login foo` |
| `login` | `server-url`, `kubectl-user` | if you do not wish to create a config file and only intend on logging in just once, you can set the server URL directly using the `--server-url` flag which **MUST** be set; kubectl-user will still default to kubelogin_user if not supplied. The alias flag is not accepted here | `kubelogin login --server-url=foo --kubectl-user=bar ` |

## Pre-Deploy Action & Configuration

1. Download binary file from the server and move it into your bin directory.
Speak with your friendly neighborhood Kubernetes team to get the link :)

## Post-Deploy Action & Configuration

1. For first time use or a username change, you'll need to run the following:
`kubectl config set-context CLUSTER_ID --user=USER` where USER is what you
defined `kubectl-user` as when running `kubelogin config`. If you did not set
`kubectl-user` when running config, it will default to `kubelogin_user`.

### Note

If you experience timeout issues with the CLI, check your proxy settings.

# Server

## Usage

The kubelogin server acts as a way of retrieving a JWT from an OIDC provider
and sending it back to the kubelogin CLI client running locally.

- Prometheus metrics are handled through the `/metrics` endpoint

- A health check is provided through the `/health` endpoint

- The initial login to the server that redirects to the specified OIDC
  provider is handled through the `/login` endpoint

- The server listens for a response from the OIDC provider on the `/callback`
  endpoint

- The server listens for the custom token for JWT exchange request on the
  `/exchange` endpoint

- The server has a static site handled at root giving a brief description of
  the app as well as providing download links to the CLI

- Download links are provided through the `/download/` path and use the Docker
  image environment to search for the files

- Files are saved as `.tar.gz` for macOS & Linux and `.zip` for Windows

## Pre-Deploy Action & Configuration

The following are **REQUIRED** to be set up in the Kubernetes environment

| Environment Variables | Description |
| :--- | :--- |
| **OIDC_PROVIDER_URL** | this is the base URL of the OIDC provider i.e. https://example.oidcprovider.com/ |
| **LISTEN_PORT** | the port that the server will listen on. Should match port in deployment.yaml file |
| **GROUPS_CLAIM** | this is most often just `groups` however can differ depending on how authorization has the JWT claims configured. Used when specifying what scopes you want to receive from auth provider. Server will default this to `groups` if not set |
| **USER_CLAIM** | this is most often called `user` or `email` however can differ depending on how authorization has the JWT claims configured. Used when specifying what scopes you want to receive from auth provider. Server will default this to `email` if not set |
| **CLIENT_ID** | the OIDC client ID. Typically this should be provided via a Secret when deployed on Kubernetes  |
| **CLIENT_SECRET** | the OIDC client secret. Typically this should be provided via a Secret when deployed on Kubernetes |
| **REDIRECT_URL** | the URL that the OIDC provider will redirect users to callback to this server after authenticating. This URL must address the kubelogin server and be reachable by users. |
| **REDIS_ADDR** | address of the Redis server that will briefly hold JWTs between the underlying Authorization Server and the kubelogin CLI. This is set when Redis is deployed to Kubernetes and needs to be set as an environment variable in your Kubernetes deployment file |
| **REDIS_PASSWORD** | password to allow for connection to the Redis cache. Should be supplied via a secret in Kubernetes |
| **REDIS_TTL** | time to live for JWTs in Redis. Accepts a duration string (e.g., 1m, 2s). Defaults to 10s |
| **DOWNLOAD_DIR** | this is the overall directory to use when searching for the binary files. For example: `kubelogin/assets/`. Defaults to `/download` if not set |

Note about the download directory: We have standardized on each download file
residing inside a folder labeled as the respective operating system i.e.,
`/mac`, `/windows`, and `/linux` which are contained in an overarching folder
labeled `/download` which resides in the root of the Docker image. The name of
the download folder can change and the path to this folder can change as well.
However `/mac`, `/windows`, and `/linux` will not change and if you put the
download files in different folders, the links will not work unless you change
the HTML in the kubelogin server code; but these changes will remain local to
your deployment (or fork) and will not be merged into our `master` branch.

## Deploy

- Deployment should be handled through Helm charts. A Makefile will help with
  setting the environment variables that are not secrets or Redis based

- Helm documentation:
  https://github.com/kubernetes/helm/blob/master/docs/index.md
