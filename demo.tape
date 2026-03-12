Output demo.gif

Set Shell "zsh"
Set FontSize 14
Set Width 1400
Set Height 800
Set Theme "TokyoNight"
Set Padding 20

# Launch gpk
Type "gpk"
Enter
Sleep 3s

# Show the table loaded
Sleep 2s

# --- Tab cycling ---
# Cycle through tabs
Tab
Sleep 1.5s

Tab
Sleep 1.5s

Tab
Sleep 1.5s

Tab
Sleep 1.5s

Tab
Sleep 1.5s

# Go back to ALL
Tab
Sleep 1s
Tab
Sleep 1.5s

# --- Fuzzy search by tool name ---
# Search for "ffmpeg"
Type "/"
Sleep 0.5s
Type "ffmpeg"
Sleep 2s

# Clear search
Escape
Sleep 1s

# Search for "btop"
Type "/"
Sleep 0.5s
Type "btop"
Sleep 2s

# Clear search
Escape
Sleep 1s

# Search for "cargo-edit"
Type "/"
Sleep 0.5s
Type "cargo-edit"
Sleep 2s

# Clear search
Escape
Sleep 1s

# --- Fuzzy search by description ---
# Search by description keyword "codec"
Type "/"
Sleep 0.5s
Type "codec"
Sleep 2s

# Clear search
Escape
Sleep 1s

# Search by description keyword "lossless"
Type "/"
Sleep 0.5s
Type "lossless"
Sleep 2s

# Clear search
Escape
Sleep 1s

# Search by description keyword "font"
Type "/"
Sleep 0.5s
Type "font"
Sleep 2s

# Clear search
Escape
Sleep 1.5s

# Quit
Type "q"
Sleep 0.5s
