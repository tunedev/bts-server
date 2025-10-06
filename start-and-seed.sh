#!/bin/bash

set -e

echo "Building the Go server..."
go build -o server .

echo "Building the seed script..."
go build -o seed-runner cmd/seed/.

echo "Starting the server in the background..."
./server &

SERVER_PID=$!

echo "SERVER started with PID ${SERVER_PID}. Waiting a moment for it to initialize..."

sleep 2

echo "Runnint the seed script..."
./seed-runner

echo "Seeding complete. The server is now running in the forground for systemd."
wait ${SERVER_PID}