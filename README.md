# kratos-demo-ui-go

This is just a quick and dirty demo of running Ory Kratos and Hydra with a Golang backend for the HTML app.

## Getting started

Start the Ory services and CockroachDB database:

```sh
docker compose up -d
```

### Create a OAuth 2.0 client

Use the Hydra API to create a client, make the following request:

```
POST http://127.0.0.1:4445/clients

{
  "client_id": "demo",
  "client_name": "",
  "client_secret": "secret",
  "redirect_uris": [
    "http://127.0.0.1:8080/callback"
  ],
  "grant_types": [
    "authorization_code"
  ],
  "response_types": [
    "code"
  ],
  "scope": "offline_access offline openid",
  "audience": [
    "https://dsjones.me"
  ],
  "owner": "dan",
  "policy_uri": "",
  "allowed_cors_origins": [],
  "tos_uri": "",
  "client_uri": "",
  "logo_uri": "",
  "contacts": [],
  "client_secret_expires_at": 0,
  "subject_type": "public",
  "jwks": {},
  "token_endpoint_auth_method": "client_secret_post",
  "userinfo_signed_response_alg": "none",
  "created_at": "2021-03-22T16:33:09Z",
  "updated_at": "2021-03-22T21:44:39.016924Z",
  "metadata": {}
}
```

### Run the login App

```
go run main.go 
```

Then open the following in a browser: [http://127.0.0.1:8080](http://127.0.0.1:8080)