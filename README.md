galaxus-price-watcher
---

# Prep

You need to download `geckodriver` ([link](https://github.com/mozilla/geckodriver)) and `selenium-server.jar` ([link](https://www.selenium.dev/downloads/)) in the `driver` directory.


Edit `gpw.toml` to configure items and the paths to the vendor files.

# Build
```
go mod vendor
docker build --rm -t gpw:latest -f Dockerfile .
```

# Run
```
docker-compose up
```
