### POC for Cognito with Twitter OAuth2.0

Cognito doesn't officially support twitter OAuth2.0.
There is a way to add custom IDP using OIDC but twitter doesn't support OIDC.
This adds some way to transform the input data from cognito and proxy request to twitter.

JWKS endpoints are not called by cognito when using PKCS method of authentication.
Looks like any random values can be added in the cognito endpoint configuration as they are used.

#### Running

- copy `.env.example` to `.env` and update required values.
- run the server, open ngrok or any tunneling software.
- setup twitter oauth application to redirect to the tunnel address.
- setup cognito idp to use the urls from the tunnel address.

#### Working;

- Cognito requests login to this endpoint when doing the sign in operation.
- The endpoint adds necessary parameters and forwards the request to twitter.
- Twitter redirects back to our endpoint and we redirect back to cognito.
- Cognito calls `/token` and `/userinfo` endpoint for creating/updating users when logging in.

#### Endpoints

| Points       | Description                                                                                                                                              |
| ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/authorize` | Authorize calls the twitter oauth entrypoint that shows user consent screen.                                                                             |
| `/callback`  | Callback receives code and state from twitter server and processes accordingly, redirecting to aws cognito                                               |
| `/token`     | Responsible for getting code from cognito and then calling the twitter service to exchange code with auth token. Response is sent back to cognito as is  |
| `/userinfo`  | Cognito calls to map the idp attributes with the cognito attributes. The call is made using the bearer token from twitter server in the previous request |
