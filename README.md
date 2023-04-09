# Backend
Vencord API

## Hosting
The API has a Docker Compose configuration, so software-wise you shouldn't need much more than just Docker.
Docker is the official way of hosting the backend, and other setups (whilst technically supported) will be
up to you to manage.

1. Clone the repository
2. Copy `.env.example` to `.env`
3. Configure as necessary
  - Port and host are irrelevant since it's running in a container, but you can change them if you wish.
  - `REDIS_URI` should be changed to `redis:6379`.
  - `ROOT_REDIRECT` should be changed to whatever you want the `/` of the API to be set to a different site,
    like your own personal homepage.
  - `DISCORD_*` should be configured with your Discord application.
  - `PEPPER_*` should be unique values. These provide extra anonymity and make it more difficult to get user
    info. It [is recommended](https://stackoverflow.com/a/9622855) you use at least 32 bytes of randomness, e.g. through `openssl rand 32 -hex`
  - `SIZE_LIMIT` is up to you, but should usually be left as default. This is for the settings sync and how
    much data a user can store.
4. Create a `docker-compose.override.yml` that maps your ports, like so:
   ```yaml
   services:
     backend:
       ports:
         - HOST_PORT:8080
   ```
5. `docker compose up -d`

Please note that, due to mixed content requirements, you will need HTTPS on your self-hosted instance.
