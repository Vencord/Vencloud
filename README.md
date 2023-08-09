
# Vencord API Hosting Guide

Welcome to the Vencord API Hosting Guide. This guide will walk you through the process of setting up and hosting the Vencord API using Docker Compose. By following these steps, you'll have the API up and running on your preferred hosting environment.

## Prerequisites

Before you begin, ensure you have the following:

- Docker installed on your machine.
- A clone of the Vencord API repository.

## Configuration Steps

1. **Clone the Repository:**
   ```
   git clone https://github.com/vencord/api.git
   ```

2. **Configure Environment Variables:**
   - Copy the `.env.example` file to `.env`.
   - Open the `.env` file and configure the following variables as necessary:
     - `REDIS_URI`: Change to `redis:6379`.
     - `ROOT_REDIRECT`: Set this to the desired path for the API's root (e.g., your personal homepage).
     - `DISCORD_*`: Configure with your Discord application details. The redirect URI should be `https://<yourdomain>/v1/oauth/callback`.
     - `PEPPER_*`: Use unique values for extra anonymity. Generate randomness (e.g., `openssl rand 32 -hex`).
     - `SIZE_LIMIT`: Consider leaving it as default, unless you have specific requirements.
   
3. **Create Docker Compose Override:**
   - Create a `docker-compose.override.yml` file in the project directory.
   - Add the following content to map your preferred host port:
     ```yaml
     services:
       backend:
         ports:
           - HOST_PORT:8080
     ```

4. **Start Docker Compose:**
   Run the following command to start the API using Docker Compose:
   ```
   docker-compose up -d
   ```

## HTTPS Requirement

Please note that due to mixed content requirements, you must have HTTPS enabled on your self-hosted instance. Ensure you have a valid SSL certificate configured for your domain.

## Conclusion

Congratulations! You have successfully set up and hosted the Vencord API using Docker Compose. Your API should now be accessible via your domain over HTTPS.

For further support or troubleshooting, feel free to reach out to the Vencord community.


Feel free to modify and customize this guide according to your needs. Happy hosting!
