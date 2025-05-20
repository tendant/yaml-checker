# YAML GitHub Editor

A web application that allows users to update YAML files stored in GitHub repositories. This application consists of a Golang backend and a SolidJS frontend.

## Features

- Connect to GitHub repositories using a personal access token
- View YAML file structure (keys only) from GitHub repositories
- Set, delete, and add key-value pairs in YAML files without exposing existing values
- Check if keys exist in YAML files (without revealing values)
- View command history
- Push changes back to GitHub automatically
- Enhanced security by never exposing sensitive values

## Prerequisites

- Go 1.18 or higher
- Node.js 14 or higher
- npm or yarn
- [yq](https://github.com/mikefarah/yq) command-line tool installed and available in your PATH

## Environment Variables

The application can be configured using the following environment variables:

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `REPO_OWNER` | GitHub repository owner (username or organization) | "" (empty string) |
| `REPO_NAME` | GitHub repository name | "" (empty string) |
| `BRANCH` | GitHub branch name | "main" |
| `GITHUB_TOKEN` | GitHub personal access token | "" (empty string) |
| `FILE_PATHS` | Comma-separated list of YAML file paths to offer as options | "config.yaml,config/app.yaml,deploy/values.yaml" |
| `PORT` | Port number for the server | "4000" |

You can set these environment variables directly or use a `.env` file. A sample `.env.example` file is provided that you can copy and modify:

```bash
cp .env.example .env
# Edit .env with your preferred text editor
```

The `run-dev.sh` script will automatically load environment variables from the `.env` file if it exists.

## Installation

1. Clone the repository:

```bash
git clone https://github.com/yourusername/yaml-checker.git
cd yaml-checker
```

2. Install backend dependencies:

```bash
go mod download
```

3. Install frontend dependencies:

```bash
cd frontend
npm install
cd ..
```

## Running the Application

### Development Mode

#### Option 1: Using the convenience script

Run the provided script to start both the backend and frontend servers:

```bash
./run-dev.sh
```

This script will:
- Check if yq is installed
- Start the Go backend server
- Start the SolidJS frontend development server
- Automatically shut down both servers when you press Ctrl+C

#### Option 2: Manual startup

1. Start the backend server:

```bash
go run main.go
```

2. In a separate terminal, start the frontend development server:

```bash
cd frontend
npm run dev
```

3. Open your browser and navigate to http://localhost:3000

### Production Mode

#### Option 1: Standard deployment

1. Build the frontend:

```bash
cd frontend
npm run build
cd ..
```

2. Start the backend server:

```bash
go run main.go
```

3. The application will be available at http://localhost:8082

#### Option 2: Docker deployment

1. Build the Docker image:

```bash
docker build -t yaml-github-editor .
```

2. Run the Docker container with environment variables:

```bash
docker run -p 8082:8082 \
  -e REPO_OWNER=octocat \
  -e REPO_NAME=hello-world \
  -e BRANCH=main \
  -e GITHUB_TOKEN=your_github_token \
  -e FILE_PATHS=config.yaml,deploy/values.yaml \
  yaml-github-editor
```

Alternatively, you can use an environment file:

```bash
docker run -p 8082:8082 --env-file .env yaml-github-editor
```

3. The application will be available at http://localhost:8082

## Usage

1. Configure GitHub settings:
   - Enter your GitHub repository owner (username or organization)
   - Enter the repository name
   - Enter the branch name (defaults to "main")
   - Enter the path to the YAML file in the repository
   - Enter your GitHub personal access token with repo permissions
   - Click "Connect & Load YAML"

2. Edit YAML:
   - Use the "Edit YAML" tab to set key-value pairs
   - Enter the key path (e.g., "app.name") and the value
   - Click "Set" to update the value

3. Check Keys:
   - Use the "Check Keys" tab to check if a key exists
   - Enter the key path and click "Check"
   - You can also delete keys by clicking "Delete"

4. Advanced Commands:
   - Use the "Advanced" tab to execute custom YQ commands
   - Supported commands: set, delete, add
   - Examples:
     - `set app.name=MyApp`
     - `delete app.logging`
     - `add newkey=value`

5. View History:
   - Use the "History" tab to view your command history

## GitHub Token Permissions

The application requires a GitHub personal access token with the following permissions:
- `repo` (Full control of private repositories)

## Architecture

- **Backend**: Golang server that handles API requests, interacts with GitHub, and processes YAML files using the yq command-line tool.
  - Implements security measures to prevent exposure of sensitive values
  - Uses temporary files for YAML processing to avoid storing sensitive data
  - Extracts only key structures from YAML files, never returning values
  
- **Frontend**: SolidJS application that provides a user-friendly interface for editing YAML files.
  - Displays only key structures, never showing values
  - Provides a tree-like view of YAML keys for easy navigation
  - Implements a tab-based interface for different operations
  
- **API Endpoints**:
  - `/api/command`: Execute commands on YAML files and push changes to GitHub (returns only keys, not values)
  - `/api/check-key`: Check if a key exists in a YAML file (returns existence and length only, not the value)
  - `/api/content`: Fetch YAML keys structure without any values

## Security Considerations

- **Value Protection**: The application never displays or returns the values in YAML files, only the keys. This ensures sensitive information remains secure.
- **GitHub tokens** are sensitive information. The application does not store tokens, but they are sent with each request.
- **Token Storage**: Consider using environment variables or a secure vault for token storage in production environments.
- **CORS**: The application uses CORS headers to allow cross-origin requests in development mode.
- **Minimal Information**: When checking if a key exists, only the existence and length of the value is returned, not the actual value.
- **Secure Updates**: When updating values, the application only returns the updated keys structure, not the values.

## License

MIT
