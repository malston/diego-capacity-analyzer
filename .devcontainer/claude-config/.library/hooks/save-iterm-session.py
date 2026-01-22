#!/usr/bin/env python3

import iterm2
import os
from datetime import datetime

async def main(connection):
    app = await iterm2.async_get_app(connection)

    # Get the active session
    session = app.current_terminal_window.current_tab.current_session

    # Get session contents (returns a list of LineContents objects)
    contents_list = await session.async_get_contents(0, 100000)

    # Create directory if needed
    save_dir = os.path.expanduser("~/Documents/iTerm2-Sessions")
    os.makedirs(save_dir, exist_ok=True)

    # Save to file with timestamp
    timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    filename = os.path.join(save_dir, f"iterm-session-{timestamp}.txt")

    with open(filename, 'w') as f:
        for line_contents in contents_list:
            f.write(line_contents.string + '\n')

    print(f"Session saved to {filename}")

iterm2.run_until_complete(main)