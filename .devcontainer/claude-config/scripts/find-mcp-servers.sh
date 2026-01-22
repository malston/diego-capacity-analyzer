#!/usr/bin/env bash
# ABOUTME: Discovers and displays all MCP server configuration sources
# ABOUTME: Shows user config, project config, plugins, and enterprise managed servers

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== MCP Server Configuration Discovery ===${NC}\n"

# Function to extract and display MCP servers from a JSON file
display_mcp_servers() {
    local file="$1"
    local scope="$2"

    if [[ ! -f "$file" ]]; then
        echo -e "${YELLOW}  Not found${NC}"
        return
    fi

    # Check if file has mcpServers field
    local servers=$(jq -r '.mcpServers // {} | keys[]' "$file" 2>/dev/null || echo "")

    if [[ -z "$servers" ]]; then
        echo -e "${YELLOW}  File exists but no mcpServers configured${NC}"
        return
    fi

    echo -e "${GREEN}  Found MCP servers:${NC}"
    while IFS= read -r server; do
        local type=$(jq -r ".mcpServers.\"$server\".type // \"unknown\"" "$file")
        local cmd=$(jq -r ".mcpServers.\"$server\".command // empty" "$file")
        local url=$(jq -r ".mcpServers.\"$server\".url // empty" "$file")

        echo -e "    ${BLUE}$server${NC} (type: $type)"
        if [[ -n "$cmd" ]]; then
            echo -e "      command: $cmd"
        fi
        if [[ -n "$url" ]]; then
            echo -e "      url: $url"
        fi
    done <<< "$servers"
}

# 1. Check user-scoped configuration
echo -e "${CYAN}[1] User Scope (Global)${NC}"
echo -e "    File: ~/.claude.json"
display_mcp_servers "$HOME/.claude.json" "user"
echo ""

# 2. Check project-scoped configuration
echo -e "${CYAN}[2] Project Scope (Current Project)${NC}"
echo -e "    File: $(pwd)/.mcp.json"
display_mcp_servers "$(pwd)/.mcp.json" "project"
echo ""

# 2b. Check parent directories for .mcp.json files
echo -e "${CYAN}[2b] Parent Directory .mcp.json Files (Inherited)${NC}"
current_dir="$(pwd)"
parent_found=false

# Walk up the directory tree looking for .mcp.json files
check_dir="$current_dir/.."
while [[ "$check_dir" != "/" ]] && [[ "$check_dir" != "." ]]; do
    abs_check_dir=$(cd "$check_dir" 2>/dev/null && pwd)
    if [[ -f "$abs_check_dir/.mcp.json" ]]; then
        parent_found=true
        echo -e "    ${GREEN}Found: $abs_check_dir/.mcp.json${NC}"
        display_mcp_servers "$abs_check_dir/.mcp.json" "parent"
        echo ""
    fi
    check_dir="$check_dir/.."
done

if [[ "$parent_found" == false ]]; then
    echo -e "${YELLOW}  No parent .mcp.json files found${NC}"
fi
echo ""

# 3. Check local-scoped configuration (project-specific in ~/.claude.json)
echo -e "${CYAN}[3] Local Scope (Project-Specific in User Config)${NC}"
echo -e "    Path: Project-specific entries in ~/.claude.json"
if [[ -f "$HOME/.claude.json" ]]; then
    local_servers=$(jq -r --arg pwd "$(pwd)" '.mcpServers // {} | to_entries[] | select(.value.scope == "local" and .value.cwd == $pwd) | .key' "$HOME/.claude.json" 2>/dev/null || echo "")
    if [[ -n "$local_servers" ]]; then
        echo -e "${GREEN}  Found local-scoped servers for this project:${NC}"
        echo "$local_servers" | while IFS= read -r server; do
            echo -e "    ${BLUE}$server${NC}"
        done
    else
        echo -e "${YELLOW}  No local-scoped servers for this project${NC}"
    fi
else
    echo -e "${YELLOW}  ~/.claude.json not found${NC}"
fi
echo ""

# 4. Check enterprise managed configuration
echo -e "${CYAN}[4] Enterprise Managed (System-Wide)${NC}"
case "$OSTYPE" in
    darwin*)
        managed_file="/Library/Application Support/ClaudeCode/managed-mcp.json"
        ;;
    linux*)
        managed_file="/etc/claude-code/managed-mcp.json"
        ;;
    msys*|cygwin*)
        managed_file="/c/Program Files/ClaudeCode/managed-mcp.json"
        ;;
    *)
        managed_file=""
        ;;
esac

if [[ -n "$managed_file" ]]; then
    echo -e "    File: $managed_file"
    if [[ -f "$managed_file" ]]; then
        echo -e "${RED}  ⚠️  Enterprise managed config found (overrides user configuration)${NC}"
        display_mcp_servers "$managed_file" "managed"
    else
        echo -e "${YELLOW}  Not configured${NC}"
    fi
else
    echo -e "${YELLOW}  Unknown OS type${NC}"
fi
echo ""

# 5. Check Claude Desktop configuration
echo -e "${CYAN}[5] Claude Desktop Configuration${NC}"
claude_desktop_configs=(
    "$HOME/Library/Application Support/Claude/claude_desktop_config.json"
    "$HOME/.config/Claude/claude_desktop_config.json"
    "$HOME/AppData/Roaming/Claude/claude_desktop_config.json"
)

desktop_found=false
for config_file in "${claude_desktop_configs[@]}"; do
    if [[ -f "$config_file" ]]; then
        echo -e "    File: $config_file"
        display_mcp_servers "$config_file" "claude-desktop"
        desktop_found=true
        break
    fi
done

if [[ "$desktop_found" == false ]]; then
    echo -e "${YELLOW}  No Claude Desktop configuration found${NC}"
fi
echo ""

# 6. Check Claude Code's actual configured servers
echo -e "${CYAN}[6] Currently Connected Servers (from 'claude mcp list')${NC}"
if command -v claude &> /dev/null; then
    echo -e "${GREEN}  Active MCP servers:${NC}"
    claude mcp list 2>&1 | grep -E "^[a-zA-Z0-9_-]+:" | while IFS=: read -r server_name server_info; do
        # Extract status
        if echo "$server_info" | grep -q "✓ Connected"; then
            status="${GREEN}✓ Connected${NC}"
        elif echo "$server_info" | grep -q "✗ Failed"; then
            status="${RED}✗ Failed${NC}"
        else
            status="${YELLOW}? Unknown${NC}"
        fi
        echo -e "    ${BLUE}$server_name${NC} - $status"
        echo -e "      $(echo "$server_info" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | sed 's/ - .*//')"
    done
else
    echo -e "${YELLOW}  'claude' command not found${NC}"
fi
echo ""

# 7. Check installed plugins
echo -e "${CYAN}[7] Installed Plugins${NC}"
plugin_dirs=(
    "$HOME/.claude/plugins"
    "$HOME/.config/claude-code/plugins"
    "$HOME/Library/Application Support/ClaudeCode/plugins"
)

plugins_with_mcp=()
for plugin_dir in "${plugin_dirs[@]}"; do
    if [[ -d "$plugin_dir" ]]; then
        # Find all plugin.json files
        while IFS= read -r plugin_file; do
            has_mcp=false
            plugin_path=$(dirname "$plugin_file")

            # Get actual plugin name from the path
            if [[ "$plugin_path" =~ /cache/([^/]+)/([^/]+)/([^/]+) ]]; then
                # Format: cache/marketplace/plugin/version
                plugin_name="${BASH_REMATCH[2]}@${BASH_REMATCH[1]} (v${BASH_REMATCH[3]})"
            elif [[ "$plugin_path" =~ /marketplaces/([^/]+)/plugins/([^/]+) ]]; then
                # Format: marketplaces/marketplace/plugins/plugin
                plugin_name="${BASH_REMATCH[2]}@${BASH_REMATCH[1]}"
            elif [[ "$plugin_path" =~ /marketplaces/([^/]+)$ ]]; then
                # Format: marketplaces/plugin
                plugin_name="${BASH_REMATCH[1]}"
            else
                plugin_name=$(basename "$plugin_path")
            fi

            # Check for inline MCP servers in plugin.json
            inline_servers=$(jq -r '.mcpServers // {} | keys[]' "$plugin_file" 2>/dev/null || echo "")

            # Check for bundled .mcp.json
            plugin_mcp_file="$plugin_path/.mcp.json"
            bundled_servers=""
            if [[ -f "$plugin_mcp_file" ]]; then
                bundled_servers=$(jq -r '.mcpServers // {} | keys[]' "$plugin_mcp_file" 2>/dev/null || echo "")
            fi

            # Only display if plugin has MCP servers
            if [[ -n "$inline_servers" ]] || [[ -n "$bundled_servers" ]]; then
                has_mcp=true
                plugins_with_mcp+=("$plugin_name")

                echo -e "\n  ${GREEN}Plugin: $plugin_name${NC}"

                if [[ -n "$inline_servers" ]]; then
                    echo -e "    ${BLUE}Inline MCP servers:${NC}"
                    echo "$inline_servers" | while IFS= read -r server; do
                        echo -e "      - $server"
                    done
                fi

                if [[ -n "$bundled_servers" ]]; then
                    echo -e "    ${BLUE}Bundled MCP servers:${NC}"
                    echo "$bundled_servers" | while IFS= read -r server; do
                        echo -e "      - $server"
                    done
                fi
            fi

        done < <(find "$plugin_dir" -name "plugin.json" 2>/dev/null)
    fi
done

if [[ ${#plugins_with_mcp[@]} -eq 0 ]]; then
    echo -e "${YELLOW}  No plugins with MCP servers found${NC}"
fi
echo ""

# 8. Summary
echo -e "${CYAN}=== Summary ===${NC}"
echo -e "To add MCP servers:"
echo -e "  ${BLUE}claude mcp add <server-name>${NC}  (adds to local scope)"
echo -e "  ${BLUE}claude mcp add <server-name> --scope user${NC}  (adds to user scope)"
echo -e "  ${BLUE}claude mcp add <server-name> --scope project${NC}  (adds to project .mcp.json)"
echo ""
echo -e "To list configured servers:"
echo -e "  ${BLUE}claude mcp list${NC}"
echo ""
echo -e "To reset project choices:"
echo -e "  ${BLUE}claude mcp reset-project-choices${NC}"
