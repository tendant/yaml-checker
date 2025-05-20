#!/bin/bash

# YAML GitHub Editor Development Runner
# This script starts both the backend and frontend in development mode

# Check if yq is installed
if ! command -v yq &> /dev/null; then
    echo "Error: yq is not installed or not in PATH"
    echo "Please install yq: https://github.com/mikefarah/yq#install"
    exit 1
fi

# Function to kill background processes on exit
cleanup() {
    echo "Shutting down servers..."
    kill $BACKEND_PID $FRONTEND_PID 2>/dev/null
    exit 0
}

# Set up trap to catch Ctrl+C and other termination signals
trap cleanup SIGINT SIGTERM

# Environment variables will be loaded by the Go application using godotenv
echo "Environment variables will be loaded from .env file by the Go application"

# Start the backend server
echo "Starting Go backend server..."
go run main.go &
BACKEND_PID=$!

# Check if backend started successfully
sleep 2
if ! ps -p $BACKEND_PID > /dev/null; then
    echo "Error: Failed to start backend server"
    exit 1
fi

echo "Backend server running on http://localhost:8082"

# Start the frontend development server
echo "Starting SolidJS frontend server..."
cd frontend && npm run dev &
FRONTEND_PID=$!

# Check if frontend started successfully
sleep 5
if ! ps -p $FRONTEND_PID > /dev/null; then
    echo "Error: Failed to start frontend server"
    kill $BACKEND_PID
    exit 1
fi

echo "Frontend server running on http://localhost:3000"
echo "YAML GitHub Editor is now running in development mode"
echo "Press Ctrl+C to stop both servers"

# Wait for user to press Ctrl+C
wait
