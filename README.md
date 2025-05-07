# AxI_Calendar_backend

This is a backend service for All In One Calendar

## Local Development Setup

This guide explains how to set up and run the AxiCalendar backend service locally for development purposes.

### Prerequisites

- **Go:** Version 1.18 or later. ([Installation Guide](https://go.dev/doc/install))
- **Docker:** To run DynamoDB Local. ([Installation Guide](https://docs.docker.com/get-docker/))
- **AWS CLI:** Version 2. ([Installation Guide](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html))

### 1. Clone the Repository

```bash
git clone <repository-url>
cd AxI_calendar
```

### 2. Configure AWS CLI for Local Development

We need to configure a specific AWS CLI profile to interact with DynamoDB Local.

```bash
aws configure --profile axicalendar-dev
# Enter the following when prompted:
# AWS Access Key ID [None]: dummy
# AWS Secret Access Key [None]: dummy
# Default region name [None]: ap-northeast-1  (or any valid region)
# Default output format [None]: json
```

Next, edit your AWS config file (`~/.aws/config`) and add the `endpoint_url` for the `axicalendar-dev` profile:

```ini
[profile axicalendar-dev]
region = ap-northeast-1
output = json
endpoint_url = http://localhost:8000
```

This tells the AWS CLI to send requests for the `axicalendar-dev` profile to your local DynamoDB instance instead of the actual AWS cloud.

### 3. Set Up Local Database

The `Makefile` provides convenient commands to manage the local development environment.

Run the following command to:

1.  Start a DynamoDB Local container using Docker.
2.  Create the necessary DynamoDB table (`AxiCalendarTable-dev` by default) using the `axicalendar-dev` AWS CLI profile.

```bash
make setup-db
```

- **Note:** This command might take a few seconds the first time as Docker downloads the `amazon/dynamodb-local` image.
- You can start/stop the database container separately using `make start-db` and `make stop-db`.
- You can create/delete the table separately using `make create-table` and `make delete-table`.

### 4. Run the Application

Once the database is set up, you can build and run the application using:

```bash
make run
```

This command will:

1.  Build the Go application (`./axicalendar-api`).
2.  Set the required environment variables (`DYNAMODB_TABLE_NAME`, `DUMMY_USER_ID`, `AWS_PROFILE`).
3.  Run the application.

The server will start on `http://localhost:8080` by default. You should see log output indicating the server has started and is using the dummy authentication middleware.

- **Note:** The `DUMMY_USER_ID` environment variable (default: `11111111-1111-1111-1111-111111111111`) is used by the dummy authentication middleware. All requests will be processed as if they belong to this user. You can override this when running: `make run DUMMY_USER_ID=<your-uuid>`
- Press `Ctrl+C` to stop the server.

### 5. Test the API

You can use tools like `curl` or Postman to send requests to the running server (e.g., `http://localhost:8080`).

**Note:** In a real deployment, most endpoints require authentication via Cognito. The local setup uses dummy authentication, so you don't need to provide tokens for these examples.

**Example using curl:**

- **Get Health Check:**

  ```bash
  curl http://localhost:8080/health
  ```

- **Get Themes:**
  ```bash
  curl http://localhost:8080/themes
  ```
- **Create a Theme:** (Includes `supported_features` as per V1.1 design)
  ```bash
  curl -X POST http://localhost:8080/themes \
  -H "Content-Type: application/json" \
  -d '{
    "theme_name": "My Daily Log",
    "fields": [
      {"name": "mood", "label": "Mood", "type": "select", "required": true},
      {"name": "notes", "label": "Notes", "type": "textarea", "required": false}
    ],
    "supported_features": ["monthly_summary"]
  }'
  ```
- **Get a Specific Theme (replace theme_id):**
  ```bash
  curl http://localhost:8080/themes/<your-theme-id>
  ```
- **Execute a Theme Feature (replace theme_id and feature_name):**
  ```bash
  curl http://localhost:8080/themes/<your-theme-id>/features/monthly_summary
  ```
- **Get Entries (replace dates):**
  ```bash
  curl "http://localhost:8080/entries?start_date=2025-01-01&end_date=2025-12-31"
  ```
- **Create an Entry (replace theme_id and date):**
  ```bash
  curl -X POST http://localhost:8080/entries \
  -H "Content-Type: application/json" \
  -d '{
    "theme_id": "<your-theme-id>",
    "entry_date": "2025-05-03",
    "data": {
      "mood": "Happy",
      "notes": "Had a productive day."
    }
  }'
  ```

### 6. Clean Up

- To stop and remove the DynamoDB Local Docker container:
  ```bash
  make stop-db
  ```
- To remove the built application binary:
  ```bash
  make clean
  ```

## Makefile Targets

- `make help`: Show available commands and variables.
- `make build`: Build the application.
- `make run`: Run the application (requires DB setup).
- `make clean`: Remove build artifacts.
- `make setup-db`: Start DynamoDB Local (Docker) and create the table.
- `make start-db`: Start DynamoDB Local (Docker).
- `make stop-db`: Stop and remove the DynamoDB Local Docker container.
- `make create-table`: Create the DynamoDB table locally.
- `make delete-table`: Delete the DynamoDB table locally.
