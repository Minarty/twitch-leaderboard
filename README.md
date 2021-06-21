# Twitch Leaderboard
Simple application to livestream our leaderboard to Twitch (Swedish target audience).  
This uses canvas to render a leaderboard image which will then be fed to FFMPEG running RTMP.

Static files that must be in the same directory as the binary:
- config.json
- font.ttf - Font to draw all text with (we use RocknRoll One, omitted for copyright)
- A "bg" folder with backgrounds (PNG or JPEG)