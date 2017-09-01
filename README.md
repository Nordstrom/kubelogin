# Kubelogin 

Repo for the kubelogin Server and CLI


# CLI


## Usage 
The intended usage of this CLI is to communicate with the kubelogin server to set the token field of the kubectl config file. The kubernetes API server will use this token for OIDC authentication.

The CLI accepts two verbs:
**login** and **config** 

How to use these verbs:

| Verb | Flags | Description | Example |
| :--- | :--- | :--- | :--- |
| `config` | alias, server, kubectl_user | If no alias flag is set, the alias is set as default. If kubectl_user isn't set, it defaults to kubelogin_user. Server **MUST** be set. If there is no existing config file, this verb will create one for you in your root directory and put the initial values in the file for you. If you give an alias that already exists, it will update that aliases information. If you give a new alias, it will add that to the existing list of aliases | `kubelogin config --alias=foo --server=bar --kubectl_user=foobar` |
| `login ALIAS` | no flags | this command will take the alias given and search for it in the config file. If no value is found, it will error out and ask you to check spelling or create a config file. | `kubelogin login foo` |
| `login` | server, kubectl_user | if you do not wish to create a config file and only intend on logging in just once, you can set the server directly using the --server flag which **MUST** be set; kubectl_user will still default to kubelogin_user if not supplied. The alias flag is not accepted here | `kubelogin login --server=foo --kubectl_user=bar ` |

## Pre-Deploy Action & Configuration
1. Download binary file and move it into your bin directory


# Server


## Usage
This Server is to be deployed on kubernetes and will act as a way of retrieving a JWT from an OIDC provider and sending it back to the client running the kubelogin CLI code.

- Prometheus metrics are handled through the /metrics endpoint

- A health check is provided through the /health endpoint

- The initial login to the server that redirects to the specified OIDC provider is handled through the /login endpoint

- The server listens for a response from the OIDC provider on the /callback endpoint

- The server listens for the custom token for JWT exchange request on the /exchange endpoint

- The server has a static site handled at root giving a brief description of the app as well as providing download links to the CLI

- Download links are provided through the /download/ path and use the Docker image environment to search for the files

- Files are saved as .zip for windows and .tar.gz for mac/linux

## Pre-Deploy Action & Configuration
The following are **REQUIRED** to be set up in the Kubernetes environment

| Environment Variables | Description |
| :--- | :--- |
| **OIDC_PROVIDER_URL** | this is the base URL of the OIDC provider i.e. https://example.oidcprovider.com/ |
| **LISTEN_PORT** | the port that the server will listen on. Should match port in deployment.yaml file |
| **GROUPS_CLAIM** | this is most often just "groups" however can differ depending on how authorization has the JWT claims configured. Used when specifying what scopes you want to receive from auth provider |
| **USER_CLAIM** | this is most often called "user" or "email" however can differ depending on how authorization has the JWT claims configured. Used when specifying what scopes you want to receive from auth provider |
| **CLIENT_ID** | this should be set up as a secret in Kubernetes that the deployment.yaml file looks for  |
| **CLIENT_SECRET** | same as client id |
| **REDIRECT_URL** | this is the URL that the OIDC provider will use to callback to this server |
| **REDIS_URL** | upon deploying Redis in the same namespace as this server, this will be set |
| **REDIS_PASSWORD** | same as Redis URL |
| **REDIS_TTL** | this sets the time to live for each entry into Redis. Defaults to 10 seconds if not set | 
| **DOWNLOAD_DIR** | this is the overall directory to use when searching for the binary files. For example: foo/bar/download. Defaults to /download if not set |

Note about the download directory: We have standardized on each download file residing inside a folder labeled as the respective operating system i.e., /mac /windows and /linux which are contained in an overarching folder labeled /download which resides in the root of the Docker image. The name of the download folder can change and the path to this folder can change as well. However /mac /windows and /linux will not change and if you put the download files in different folders, the links will not work unless you change the html in the code; but these changes will remain local to your machine and your deployment and will not be merged into our master branch.

## Deploy
- Deployment should be handled through helm charts. A Makefile will help with setting the environment variables that are not secrets or Redis based
- Helm documentation: https://github.com/kubernetes/helm/blob/master/docs/index.md
