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
In the kubernetes environment there needs to be the following environment variables
1. OIDC_PROVIDER_URL | this is the base url of the OIDC provider i.e. https://example.oidcprovider.com/
2. LISTEN_PORT | the port that the server will listen on. Should match port in deployment.yaml file
3. GROUPS_CLAIM | if the groups claim is different than just 'groups' this environment variable should be set. If not, you can edit the handleCliLogin function to use the string "groups"
4. USER_CLAIM | this user claim is most often an email. If the claim field is different than just 'email' then this environment variable should be set. If not, you can edit the handleCliLogin function to use the string "email"
5. CLIENT_ID | this should be set up as a secret in kubernetes that the deployment.yaml file looks for 
6. CLIENT_SECRET | same as client id
7. REDIRECT_URL | this is the url that the oidc provider will use to callback to this server
8. REDIS_URL | upon deploying redis in the same namespace as this server, this will be set
9. REDIS_PASSWORD | same as redis url

## Deploy

Deployment should be handled through helm charts. A makefile will help with setting the environment variables that are not secrets or redis based
Helm documentation: https://github.com/kubernetes/helm/blob/master/docs/index.md
