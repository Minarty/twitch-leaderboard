# Twitch Leaderboard
Simple application to livestream our leaderboard to Twitch (Swedish target audience).  
This runs a simple local HTTP server with the leaderboard. To update the leaderboard, we use selenium running the chrome driver to take a screenshot.  

The screenshot will be streamed with FFMPEG through RTMP.

Quite tedious to set up, since all static files must be readily available as they are not packaged in the binary.

This includes:
- config.json
- Everything in the ``site`` folder
- bg folder with the backgrounds

### Setting up selenium
For selenium to work properly you need place the following inside the **selenium** directory.
- Selenium server jar called "selenium.jar"
- Chrome WebDriver called "chromedriver"