#!/usr/bin/bash

echo "Starting deployment"

BACKEND_PATH="/home/app/"
FRONTEND_BUILD_PATH="/home/app/build"

if [ ! -f "$BACKEND_PATH/ucp" ]; then
  echo "Error: Backend build not found at $BACKEND_PATH/ucp"
  exit 1
fi

if [ ! -d "$FRONTEND_BUILD_PATH" ]; then
  echo "Error: Frontend build directory not found at $FRONTEND_BUILD_PATH"
  exit 1
fi

cd $BACKEND_PATH
chmod +x ./ucp

echo "Restarting ucp service..."
sudo systemctl stop ucp || echo "Service ucp not running"
sudo systemctl start ucp || echo "Error: could not start backend service"

echo "Deployment finished."