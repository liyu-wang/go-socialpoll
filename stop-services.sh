#!/bin/bash

# Script to stop all go-socialpoll services
# macOS: stops iTerm2 tabs | Linux: stops tmux windows
# Usage: ./stop-services.sh

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

SESSION_NAME="go-socialpoll"

echo -e "${BLUE}Stopping go-socialpoll services...${NC}\n"

# Detect OS
OS=$(uname -s)

if [ "$OS" = "Darwin" ]; then
    # ============================================
    # macOS: Kill processes by name
    # ============================================
    echo -e "${BLUE}Detected macOS - stopping iTerm2 services${NC}\n"
    
    # Function to kill a process by name
    kill_service() {
        local service_name="$1"
        if pgrep -f "$service_name" > /dev/null; then
            pkill -f "$service_name" || true
            echo -e "${GREEN}✓ Stopped: $service_name${NC}"
        else
            echo -e "${BLUE}• $service_name not running${NC}"
        fi
    }
    
    kill_service "nsqlookupd"
    kill_service "nsqd"
    kill_service "mongod"
    kill_service "chatvotes"
    kill_service "counter"

elif [ "$OS" = "Linux" ]; then
    # ============================================
    # Linux: Kill tmux session
    # ============================================
    echo -e "${BLUE}Detected Linux - stopping tmux session${NC}\n"
    
    # Kill tmux session if it exists
    if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
        tmux kill-session -t "$SESSION_NAME"
        echo -e "${GREEN}✓ Stopped tmux session: $SESSION_NAME${NC}"
    else
        echo -e "${BLUE}• tmux session $SESSION_NAME not running${NC}"
    fi
    
    # Function to kill a process by name (fallback)
    kill_service() {
        local service_name="$1"
        if pgrep -f "$service_name" > /dev/null; then
            pkill -f "$service_name" || true
            echo -e "${GREEN}✓ Stopped: $service_name${NC}"
        else
            echo -e "${BLUE}• $service_name not running${NC}"
        fi
    }
    
    # Fallback: kill any remaining processes
    echo -e "${BLUE}Cleaning up any remaining processes...${NC}"
    kill_service "nsqlookupd"
    kill_service "nsqd"
    kill_service "mongod"
    kill_service "chatvotes"
    kill_service "counter"

else
    echo -e "${RED}Unsupported OS: $OS${NC}"
    exit 1
fi

echo -e "\n${GREEN}All services stopped!${NC}"
