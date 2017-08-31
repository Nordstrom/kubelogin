# Kubelogin [![Build Status](https://travis-ci.org/Nordstrom/kubelogin.svg)](https://travis-ci.org/Nordstrom/kubelogin)

Repo for the kubelogin Server and CLI


# CLI


## Usage 
The intended usage of this CLI is to communicate with the kubelogin server to set the token field of the kubectl config file. The kubernetes API server will use this token for OIDC authentication.

The url to the kubelogin server is formatted as: https://kubelogin-clustername.server_hostname/login?port= if the cluster flag is set, otherwise the format will be https://kubelogin.server_hostname/login?port=


## Pre-Deploy Action & Configuration
1. Move binary file into bin directory
2. Add an environment variable labeled SERVER_HOSTNAME to avoid setting the --host flag everytime. This server_hostname is a value that should not change often if at all.


## Deploy
1. --user is an optional flag that can be set which will specify what username you'd like to be set in the kube config file. This defaults to auth_user.
2. --cluster is an optional flag to specify an exact clustername for the path to the server. This is set to kubelogin if no cluster is specified.
3. --host is an optional flag to specify the rest of the URL to the server after clustername. This looks for an environment variable if not set.


# Server


## Usage
This Server is to be deployed on kubernetes and will act as a way of retrieving a JWT from an OIDC provider and sending it back to the client running the kubelogin CLI code.

- Prometheus metrics are handled through the /metrics endpoint

- A health check is provided through the /health endpoint

- The initial login to the server that redirects to the specified OIDC provider is handled through the /login endpoint

- The server listens for a response from the OIDC provider on the /callback endpoint

- The server listens for the custom token for JWT exchange request on the /exchange endpoint


## Pre-Deploy Action & Configuration
The following need to be set up in the Kubernetes environment

| Environment Variables | Description |
| :--- | :--- |
| **OIDC_PROVIDER_URL** | this is the base URL of the OIDC provider i.e. https://example.oidcprovider.com/ |
| **LISTEN_PORT** | the port that the server will listen on. Should match port in deployment.yaml file |
| **GROUPS_CLAIM** | if the groups claim is different than just 'groups' this environment variable should be set. If not, you can edit the handleCLILogin function to use the string "groups" |
| **USER_CLAIM** | this user claim is most often an email. If the claim field is different than just 'email' then this environment variable should be set. If not, you can edit the handleCliLogin function to use the string "email" |
| **CLIENT_ID** | this should be set up as a secret in Kubernetes that the deployment.yaml file looks for  |
| **CLIENT_SECRET** | same as client id |
| **REDIRECT_URL** | this is the URL that the OIDC provider will use to callback to this server |
| **REDIS_URL** | upon deploying Redis in the same namespace as this server, this will be set |
| **REDIS_PASSWORD** | same as Redis URL |
| **REDIS_TTL** | this sets the time to live for each entry into Redis. Defaults to 10 seconds if not overridden | 

## Deploy
- Deployment should be handled through helm charts. A Makefile will help with setting the environment variables that are not secrets or Redis based
- Helm documentation: https://github.com/kubernetes/helm/blob/master/docs/index.md
