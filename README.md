# kubelogin Server [![Build Status](https://travis-ci.org/Nordstrom/kubelogin.svg)](https://travis-ci.org/Nordstrom/kubelogin)

Repo for the kubelogin Server

# Usage
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

## Deploy

- Deployment should be handled through helm charts. A Makefile will help with setting the environment variables that are not secrets or Redis based
- Helm documentation: https://github.com/kubernetes/helm/blob/master/docs/index.md
