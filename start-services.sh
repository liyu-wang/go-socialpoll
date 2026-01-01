#!/bin/bash

# Script to start all go-socialpoll services
# macOS: separate iTerm2 tabs | Linux: separate tmux windows
# Usage: ./start-services.sh

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WORKSPACE_DIR="$SCRIPT_DIR"
DB_PATH="$SCRIPT_DIR/data"
SESSION_NAME="go-socialpoll"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}Starting go-socialpoll services...${NC}\n"

# Detect OS
OS=$(uname -s)

# Create db directory if it doesn't exist
mkdir -p "$DB_PATH"

if [ "$OS" = "Darwin" ]; then
    # ============================================
    # macOS: Use iTerm2 tabs
    # ============================================
    
    # Check if iTerm2 is available
    if ! osascript -e 'tell application "System Events" to count (every application process whose name is "iTerm")' &> /dev/null; then
        echo -e "${RED}Error: iTerm2 is not installed or not running${NC}"
        echo "Please install iTerm2 from: https://iterm2.com/"
        exit 1
    fi
    
    echo -e "${BLUE}Detected macOS - using iTerm2${NC}\n"
    
    # Function to open a new iTerm2 tab and run a command
    open_iterm_tab() {
        local tab_title="$1"
        local command="$2"
        
        osascript <<EOF
tell application "iTerm"
    activate
    tell current window
        create tab with default profile
        tell current session
            set name to "$tab_title"
            write text "cd '$WORKSPACE_DIR' && $command"
        end tell
    end tell
end tell
EOF
    }

    # Start the first tab with nsqlookupd (creates new window)
    echo -e "${BLUE}1. Starting nsqlookupd...${NC}"
    osascript <<EOF
tell application "iTerm"
    activate
    create window with default profile
    tell current window
        tell current session
            set name to "nsqlookupd"
            write text "cd '$WORKSPACE_DIR' && nsqlookupd"
        end tell
    end tell
end tell
EOF
    echo -e "${GREEN}✓ Started: nsqlookupd${NC}"
    sleep 1

    # Start remaining services in new tabs
    echo -e "${BLUE}2. Starting nsqd...${NC}"
    open_iterm_tab "nsqd" "nsqd --lookupd-tcp-address=localhost:4160 -data-path $SCRIPT_DIR/data"
    echo -e "${GREEN}✓ Started: nsqd${NC}"
    sleep 1

    echo -e "${BLUE}3. Starting MongoDB...${NC}"
    open_iterm_tab "mongod" "mongod --dbpath $DB_PATH"
    echo -e "${GREEN}✓ Started: mongod${NC}"
    sleep 2

    echo -e "${BLUE}4. Starting chatvotes application...${NC}"
    open_iterm_tab "chatvotes" "cd $WORKSPACE_DIR/chatvotes && go run ."
    echo -e "${GREEN}✓ Started: chatvotes${NC}"
    sleep 1

    echo -e "${BLUE}5. Starting counter application...${NC}"
    open_iterm_tab "counter" "cd $WORKSPACE_DIR/counter && go run ."
    echo -e "${GREEN}✓ Started: counter${NC}"
    sleep 1

    echo -e "\n${GREEN}All services started in separate iTerm2 tabs!${NC}"
    echo -e "${BLUE}To switch between tabs:${NC}"
    echo "  • Cmd+Right or Cmd+Left arrow"
    echo "  • Or click the tab name"

elif [ "$OS" = "Linux" ]; then
    # ============================================
    # Linux: Use tmux
    # ============================================
    echo -e "${BLUE}Detected Linux - using tmux${NC}\n"
    
    # Check if tmux is installed
    if ! command -v tmux &> /dev/null; then
        echo -e "${YELLOW}tmux is not installed. Installing tmux...${NC}"
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y tmux
        elif command -v yum &> /dev/null; then
            sudo yum install -y tmux
        else
            echo -e "${RED}Please install tmux manually${NC}"
            exit 1
        fi
    fi
    
    # Kill existing session if it exists
    tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true
    sleep 0.5
    
    # Create new tmux session with first window (nsqlookupd)
    echo -e "${BLUE}1. Starting nsqlookupd...${NC}"
    tmux new-session -s "$SESSION_NAME" -d -c "$WORKSPACE_DIR"
    tmux send-keys -t "$SESSION_NAME" "nsqlookupd; bash" Enter
    echo -e "${GREEN}✓ Started: nsqlookupd${NC}"
    sleep 1

    # Create window for nsqd
    echo -e "${BLUE}2. Starting nsqd...${NC}"
    tmux new-window -t "$SESSION_NAME" -c "$WORKSPACE_DIR"
    tmux send-keys -t "$SESSION_NAME" "nsqd --lookupd-tcp-address=localhost:4160 -data-path $SCRIPT_DIR/data; bash" Enter
    echo -e "${GREEN}✓ Started: nsqd${NC}"
    sleep 1

    # Create window for MongoDB
    echo -e "${BLUE}3. Starting MongoDB...${NC}"
    tmux new-window -t "$SESSION_NAME" -n "mongod" -c "$WORKSPACE_DIR"
    tmux send-keys -t "$SESSION_NAME:mongod" "mongod --dbpath $DB_PATH; bash" Enter
    echo -e "${GREEN}✓ Started: mongod${NC}"
    sleep 2

    # Create window for chatvotes
    echo -e "${BLUE}4. Starting chatvotes application...${NC}"
    tmux new-window -t "$SESSION_NAME" -n "chatvotes" -c "$WORKSPACE_DIR/chatvotes"
    tmux send-keys -t "$SESSION_NAME:chatvotes" "go run .; bash" Enter
    echo -e "${GREEN}✓ Started: chatvotes${NC}"
    sleep 1

    # Create window for counter
    echo -e "${BLUE}5. Starting counter application...${NC}"
    tmux new-window -t "$SESSION_NAME" -n "counter" -c "$WORKSPACE_DIR/counter"
    tmux send-keys -t "$SESSION_NAME:counter" "go run .; bash" Enter
    echo -e "${GREEN}✓ Started: counter${NC}"
    sleep 1

    echo -e "\n${GREEN}All services started in tmux windows!${NC}"
    echo -e "${BLUE}Tmux keyboard shortcuts:${NC}"
    echo "  • Next window: Ctrl+b n"
    echo "  • Previous window: Ctrl+b p"
    echo "  • List windows: Ctrl+b w"
    echo "  • Detach from tmux: Ctrl+b d"
    
    # Attach to the session
    tmux attach-session -t "$SESSION_NAME"

else
    echo -e "${RED}Unsupported OS: $OS${NC}"
    echo "Please start services manually"
    exit 1
fi

echo -e "\n${BLUE}Services running:${NC}"
echo "  • nsqlookupd (port 4160, 4161)"
echo "  • nsqd (port 4150)"
echo "  • MongoDB (port 27017)"
echo "  • chatvotes (WebSocket at localhost:8080/room)"
echo "  • counter (consuming votes from NSQ)"
echo ""
echo -e "${BLUE}To stop services:${NC}"
echo "  • Run: ./stop-services.sh"
echo ""
echo -e "${BLUE}To monitor votes:${NC}"
echo "  • nsq_tail --topic=\"votes\" --lookupd-http-address=localhost:4161"
